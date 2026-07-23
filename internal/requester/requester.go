package requester

import (
	"context"
	"fmt"
)

type ErrHttpResponse struct {
	Code int
}

func (e ErrHttpResponse) Error() string {
	return fmt.Sprintf("HTTP status %d", e.Code)
}

func newHttpError(code int) error {
	return ErrHttpResponse{Code: code}
}

type Requester[Request any, Response any] interface {
	Get(ctx context.Context, path string) (response Response, err error)
	Put(ctx context.Context, path string, request *Request) (response Response, err error)
	Post(ctx context.Context, path string, request *Request) (response Response, err error)
	Delete(ctx context.Context, path string) (response Response, err error)
}
