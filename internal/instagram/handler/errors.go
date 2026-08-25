package handler

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
)

// toStatus maps domain sentinel errors to the appropriate gRPC status code.
// Unknown errors are wrapped as Internal so the client gets a safe, non-leaking message.
func toStatus(err error, internalMsg string) error {
	var ve *igmodel.ValidationError
	switch {
	case errors.As(err, &ve):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, sharedstore.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, sharedstore.ErrConflict):
		return status.Error(codes.AlreadyExists, "already exists")
	default:
		return status.Errorf(codes.Internal, "%s: %v", internalMsg, err)
	}
}
