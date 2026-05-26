package server

import (
	"errors"

	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toStatusError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err
	}

	switch {
	case errors.Is(err, apperror.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, apperror.ErrUnauthenticated), errors.Is(err, apperror.ErrDisabledUser):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, apperror.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, apperror.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, apperror.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, apperror.ErrPasswordChangeRequired):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
