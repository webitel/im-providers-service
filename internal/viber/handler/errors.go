package handler

import (
	"errors"

	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toStatus(err error, internalMsg string) error {
	var ve *vibmodel.ValidationError
	switch {
	case errors.As(err, &ve):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, sharedstore.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, sharedstore.ErrConflict):
		return status.Error(codes.AlreadyExists, "already exists")
	case errors.Is(err, vibmodel.ErrTokenInvalid):
		return status.Error(codes.Unauthenticated, "viber auth token invalid or revoked")
	case errors.Is(err, vibmodel.ErrReceiverNotSubscribed):
		return status.Error(codes.FailedPrecondition, "viber receiver not subscribed or unreachable")
	default:
		return status.Errorf(codes.Internal, "%s: %v", internalMsg, err)
	}
}
