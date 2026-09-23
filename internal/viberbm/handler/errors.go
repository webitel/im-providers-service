package handler

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

func toStatus(err error, internalMsg string) error {
	var ve *vibbmmodel.ValidationError

	switch {
	case errors.As(err, &ve):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, sharedstore.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, sharedstore.ErrConflict):
		return status.Error(codes.AlreadyExists, "already exists")
	case errors.Is(err, vibbmmodel.ErrKeyInvalid):
		return status.Error(codes.Unauthenticated, "viber_bm api key invalid or revoked")
	case errors.Is(err, vibbmmodel.ErrReceiverNotReachable):
		return status.Error(codes.FailedPrecondition, "viber_bm receiver not reachable")
	default:
		return status.Errorf(codes.Internal, "%s: %v", internalMsg, err)
	}
}
