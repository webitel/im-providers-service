package handler

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

func toStatus(err error, internalMsg string) error {
	var (
		ve *custommodel.ValidationError
		fe *custommodel.FieldError
	)

	switch {
	case errors.As(err, &ve):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.As(err, &fe):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, sharedstore.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, sharedstore.ErrConflict):
		return status.Error(codes.AlreadyExists, "already exists")
	default:
		return status.Errorf(codes.Internal, "%s: %v", internalMsg, err)
	}
}
