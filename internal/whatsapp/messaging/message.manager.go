package messaging

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/webitel/im-providers-service/internal/whatsapp/client"
	"github.com/webitel/webitel-go-kit/pkg/errors"
	"google.golang.org/grpc/codes"
)

type ApiCompatibleJsonConverterConfigs interface {
	ReplyToMessageID() string
	SendToPhoneNumber() string
}

type apiCompatibleJsonConverterConfigs struct {
	SendPhoneNumber string
	ReplyToMessage  string
}

func (apiCompatibleJsonConverterConfigs *apiCompatibleJsonConverterConfigs) ReplyToMessageID() string {
	return apiCompatibleJsonConverterConfigs.ReplyToMessage
}

func (apiCompatibleJsonConverterConfigs *apiCompatibleJsonConverterConfigs) SendToPhoneNumber() string {
	return apiCompatibleJsonConverterConfigs.SendPhoneNumber
}

type BaseMessage interface {
	ToJson(configs ApiCompatibleJsonConverterConfigs) ([]byte, error)
}

type MessageSendResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
	Error *MessageSendError `json:"error,omitempty"`
}

type MessageSendError struct {
	Message   string `json:"message"`
	Type      string `json:"type"`
	Code      int    `json:"code"`
	ErrorData struct {
		MessagingProduct string `json:"messaging_product"`
		Details          string `json:"details"`
	} `json:"error_data"`
	ErrorSubcode int    `json:"error_subcode"`
	FbtraceID    string `json:"fbtrace_id"`
}

func (messageSendError MessageSendError) ToGRPCError() error {
	var code codes.Code
	switch messageSendError.Code {
	case 190:
		code = codes.Unauthenticated
	case 100:
		code = codes.InvalidArgument
	case 4, 17, 80007:
		code = codes.ResourceExhausted
	case 10:
		code = codes.PermissionDenied
	default:
		code = codes.Internal
	}

	return errors.New(
		messageSendError.Message,
		errors.WithCode(code),
		errors.WithValue("fbtrace_id", messageSendError.FbtraceID),
		errors.WithValue("meta_code", messageSendError.Code),
		errors.WithValue("meta_subcode", messageSendError.ErrorSubcode),
	)
}

type StatusResponse struct {
	Success bool              `json:"success"`
	Error   *MessageSendError `json:"error,omitempty"`
}

func UnmarshalStatusResponse(responseStr string) (StatusResponse, error) {
	var statusResponse StatusResponse
	if err := json.Unmarshal([]byte(responseStr), &statusResponse); err != nil {
		return StatusResponse{}, errors.Internal("unmarshaling status response", errors.WithID("message.manager.unmarshal_status_response"), errors.WithCause(err))
	}
	return statusResponse, nil
}

type MessageManager struct {
	requester     client.RequestClient
	PhoneNumberID string
}

func newMessageManager(requester client.RequestClient, phoneNumberID string) *MessageManager {
	return &MessageManager{requester: requester, PhoneNumberID: phoneNumberID}
}

func (messageManager *MessageManager) Send(ctx context.Context, message BaseMessage, phoneNumber string) (*MessageSendResponse, error) {
	body, err := message.ToJson(&apiCompatibleJsonConverterConfigs{
		SendPhoneNumber: phoneNumber,
	})

	if err != nil {
		return nil, errors.Internal("converting message to json", errors.WithCause(err), errors.WithID("message.manager.send"), errors.WithValue("to_phone_number", phoneNumber))
	}

	apiRequest := messageManager.requester.NewApiRequest(
		strings.Join(
			[]string{messageManager.PhoneNumberID, "messages"}, "/",
		),
		http.MethodPost,
	)
	apiRequest.SetBody(string(body))

	responseStr, err := apiRequest.ExecuteWithContext(ctx)
	if err != nil {
		return nil, err
	}

	var sendResponse MessageSendResponse
	if err = json.Unmarshal([]byte(responseStr), &sendResponse); err != nil {
		return nil, errors.Internal("unmarshaling response", errors.WithCause(err), errors.WithID("message.manager.send"), errors.WithValue("response", responseStr))
	}

	if sendResponse.Error != nil {
		return &sendResponse, errors.Wrap(sendResponse.Error.ToGRPCError(), errors.WithID("message.manager.send"))
	}

	return &sendResponse, nil
}

func (messageManager *MessageManager) Reply(ctx context.Context, message BaseMessage, phoneNumber, replyTo string) (*MessageSendResponse, error) {
	body, err := message.ToJson(&apiCompatibleJsonConverterConfigs{
		SendPhoneNumber: phoneNumber,
		ReplyToMessage:  replyTo,
	})

	if err != nil {
		return nil, errors.Internal("converting message to json", errors.WithCause(err), errors.WithID("message.manager.reply"))
	}

	apiRequest := messageManager.requester.NewApiRequest(
		strings.Join(
			[]string{messageManager.PhoneNumberID, "messages"}, "/",
		),
		http.MethodPost,
	)
	apiRequest.SetBody(string(body))

	responseStr, err := apiRequest.ExecuteWithContext(ctx)
	if err != nil {
		return nil, err
	}

	var sendResponse MessageSendResponse
	if err = json.Unmarshal([]byte(responseStr), &sendResponse); err != nil {
		return nil, errors.Internal("unmarshaling message send response", errors.WithCause(err), errors.WithID("message.manager.reply"), errors.WithValue("response", responseStr))
	}

	if sendResponse.Error != nil {
		return &sendResponse, errors.Wrap(sendResponse.Error.ToGRPCError(), errors.WithID("message.manager.reply"))
	}

	return &sendResponse, nil
}

func (messageManager *MessageManager) readMessage(ctx context.Context, messageID string, showTyping bool) error {
	body := map[string]any{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        messageID,
	}

	if showTyping {
		body["typing_indicator"] = map[string]string{
			"type": "text",
		}
	}

	marshalled, err := json.Marshal(body)
	if err != nil {
		return errors.Internal("marshaling read message body", errors.WithID("message.manager.read_message"), errors.WithCause(err))
	}

	apiRequest := messageManager.requester.NewApiRequest(
		strings.Join(
			[]string{messageManager.PhoneNumberID, "messages"}, "/",
		),
		http.MethodPost,
	)
	apiRequest.SetBody(string(marshalled))

	responseStr, err := apiRequest.ExecuteWithContext(ctx)
	if err != nil {
		return errors.Wrap(err, errors.WithID("message.manager.read_message"))
	}

	var statusResponse StatusResponse
	if err = json.Unmarshal([]byte(responseStr), &statusResponse); err != nil {
		return errors.Internal("unmarshaling status response", errors.WithCause(err), errors.WithID("message.manager.read_message"), errors.WithValue("response", responseStr))
	}

	if statusResponse.Error != nil {
		return errors.Wrap(statusResponse.Error.ToGRPCError(), errors.WithID("message.manager.read_message"))
	}

	return nil
}

func (messageManager *MessageManager) ReadMessageOnly(ctx context.Context, messageID string) error {
	return messageManager.readMessage(ctx, messageID, false)
}

func (messageManager *MessageManager) ReadMessageWithTyping(ctx context.Context, messageID string) error {
	return messageManager.readMessage(ctx, messageID, true)
}
