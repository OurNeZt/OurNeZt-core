package server

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// InputLimitInterceptor bounds strings and repeated inputs before authentication,
// password hashing, unmarshalling JSON, or repository work.
func InputLimitInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
	if message, ok := req.(proto.Message); ok {
		if err := validateMessage(message.ProtoReflect(), 0); err != nil {
			return nil, toStatusError(err)
		}
	}
	return next(ctx, req)
}

func validateMessage(message protoreflect.Message, depth int) error {
	if depth > 8 {
		return fmt.Errorf("%w: input nesting is too deep", apperror.ErrInvalidArgument)
	}
	var failure error
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		name := string(field.Name())
		check := func(v protoreflect.Value) error {
			if field.Kind() == protoreflect.MessageKind {
				return validateMessage(v.Message(), depth+1)
			}
			if field.Kind() != protoreflect.StringKind {
				return nil
			}
			limit := 256
			switch name {
			case "email":
				limit = 254
			case "name", "display_name", "unit_type":
				limit = 120
			case "relationship_label":
				limit = 64
			case "location":
				limit = 255
			case "description":
				limit = 2000
			case "notes":
				limit = 10000
				if strings.HasSuffix(string(message.Descriptor().FullName()), ".HousingAnswer") {
					limit = 5000
				}
			case "page_token":
				limit = 512
			case "password", "current_password", "new_password":
				if len(v.String()) > domain.MaxPasswordBytes {
					return inputTooLong(name, domain.MaxPasswordBytes, "bytes")
				}
				limit = domain.MaxPasswordBytes
			}
			if !utf8.ValidString(v.String()) || utf8.RuneCountInString(v.String()) > limit {
				return inputTooLong(name, limit, "characters")
			}
			return nil
		}
		if field.IsList() {
			list := value.List()
			limit := 50
			if list.Len() > limit {
				failure = fmt.Errorf("%w: %s allows at most %d entries", apperror.ErrInvalidArgument, name, limit)
				return false
			}
			for i := 0; i < list.Len(); i++ {
				if failure = check(list.Get(i)); failure != nil {
					return false
				}
			}
		} else {
			failure = check(value)
		}
		return failure == nil
	})
	return failure
}

func inputTooLong(field string, limit int, unit string) error {
	return fmt.Errorf("%w: %s must not exceed %d %s", apperror.ErrInvalidArgument, field, limit, unit)
}
