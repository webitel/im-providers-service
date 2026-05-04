package storage

import (
	"context"
	"log/slog"

	"github.com/webitel/im-providers-service/gen/go/storage"
	webitel "github.com/webitel/im-providers-service/infra/client/grpc"
	infratls "github.com/webitel/im-providers-service/infra/tls"
	"github.com/webitel/webitel-go-kit/infra/discovery"
	rpc "github.com/webitel/webitel-go-kit/infra/transport/gRPC"
	"github.com/webitel/webitel-go-kit/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const ServiceName string = "storage"

var _ storage.FileServiceClient = (*Client)(nil)

type Client struct {
	rpc *rpc.Client[storage.FileServiceClient]
}

// BulkGenerateFileLink implements storage.FileServiceClient.
func (c *Client) BulkGenerateFileLink(ctx context.Context, in *storage.BulkGenerateFileLinkRequest, opts ...grpc.CallOption) (*storage.BulkGenerateFileLinkResponse, error) {
	panic("unimplemented")
}

// DeleteFiles implements storage.FileServiceClient.
func (c *Client) DeleteFiles(ctx context.Context, in *storage.DeleteFilesRequest, opts ...grpc.CallOption) (*storage.DeleteFilesResponse, error) {
	panic("unimplemented")
}

// DeleteQuarantineFiles implements storage.FileServiceClient.
func (c *Client) DeleteQuarantineFiles(ctx context.Context, in *storage.DeleteQuarantineFilesRequest, opts ...grpc.CallOption) (*storage.DeleteFilesResponse, error) {
	panic("unimplemented")
}

// DeleteScreenRecordings implements storage.FileServiceClient.
func (c *Client) DeleteScreenRecordings(ctx context.Context, in *storage.DeleteScreenRecordingsRequest, opts ...grpc.CallOption) (*storage.DeleteFilesResponse, error) {
	panic("unimplemented")
}

// DeleteScreenRecordingsByAgent implements storage.FileServiceClient.
func (c *Client) DeleteScreenRecordingsByAgent(ctx context.Context, in *storage.DeleteScreenRecordingsByAgentRequest, opts ...grpc.CallOption) (*storage.DeleteFilesResponse, error) {
	panic("unimplemented")
}

// DownloadFile implements storage.FileServiceClient.
func (c *Client) DownloadFile(ctx context.Context, in *storage.DownloadFileRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[storage.StreamFile], error) {
	panic("unimplemented")
}

// GenerateFileLink implements storage.FileServiceClient.
func (c *Client) GenerateFileLink(ctx context.Context, in *storage.GenerateFileLinkRequest, opts ...grpc.CallOption) (*storage.GenerateFileLinkResponse, error) {
	panic("unimplemented")
}

// RestoreFiles implements storage.FileServiceClient.
func (c *Client) RestoreFiles(ctx context.Context, in *storage.RestoreFilesRequest, opts ...grpc.CallOption) (*storage.RestoreFilesResponse, error) {
	panic("unimplemented")
}

// SafeUploadFile implements storage.FileServiceClient.
func (c *Client) SafeUploadFile(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[storage.SafeUploadFileRequest, storage.SafeUploadFileResponse], error) {
	panic("unimplemented")
}

// SearchFiles implements storage.FileServiceClient.
func (c *Client) SearchFiles(ctx context.Context, in *storage.SearchFilesRequest, opts ...grpc.CallOption) (*storage.ListFile, error) {
	panic("unimplemented")
}

// SearchFilesByCall implements storage.FileServiceClient.
func (c *Client) SearchFilesByCall(ctx context.Context, in *storage.SearchFilesByCallRequest, opts ...grpc.CallOption) (*storage.ListFile, error) {
	panic("unimplemented")
}

// SearchScreenRecordings implements storage.FileServiceClient.
func (c *Client) SearchScreenRecordings(ctx context.Context, in *storage.SearchScreenRecordingsRequest, opts ...grpc.CallOption) (*storage.ListFile, error) {
	panic("unimplemented")
}

// SearchScreenRecordingsByAgent implements storage.FileServiceClient.
func (c *Client) SearchScreenRecordingsByAgent(ctx context.Context, in *storage.SearchScreenRecordingsByAgentRequest, opts ...grpc.CallOption) (*storage.ListFile, error) {
	panic("unimplemented")
}

// UploadFile implements storage.FileServiceClient.
func (c *Client) UploadFile(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[storage.UploadFileRequest, storage.UploadFileResponse], error) {
	var response grpc.ClientStreamingClient[storage.UploadFileRequest, storage.UploadFileResponse]
	err := c.rpc.Execute(ctx, func(fsc storage.FileServiceClient) error {
		var err error
		response, err = fsc.UploadFile(ctx, opts...)
		if err != nil {
			return errors.Wrap(err, errors.WithID("storage.storage.upload_file"))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

// UploadFileUrl implements storage.FileServiceClient.
func (c *Client) UploadFileUrl(ctx context.Context, in *storage.UploadFileUrlRequest, opts ...grpc.CallOption) (*storage.UploadFileUrlResponse, error) {
	panic("unimplemented")
}

func New(logger *slog.Logger, discovery discovery.DiscoveryProvider, tls *infratls.Config) (*Client, error) {
	factory := func(conn *grpc.ClientConn) storage.FileServiceClient {
		return storage.NewFileServiceClient(conn)
	}
	c, err := webitel.New(logger, discovery, ServiceName, nil, factory)
	if err != nil {
		return nil, errors.New("initializing storage client", errors.WithCause(err), errors.WithID("storage.storage.new"), errors.WithCode(codes.Unavailable))
	}

	return &Client{
		rpc: c,
	}, nil
}
