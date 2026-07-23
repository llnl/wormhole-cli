package requester

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

func NewTokenAuthRequester[Request any, Response any](token, endpoint string, client *http.Client, logger *slog.Logger) Requester[Request, Response] {
	if client == nil {
		client = &http.Client{
			Timeout: 5 * time.Second,
		}
	}

	return &TokenAuthRequester[Request, Response]{
		token:    token,
		endpoint: endpoint,
		client:   client,
		logger:   logger,
	}
}

type TokenAuthRequester[Request any, Response any] struct {
	client   *http.Client
	token    string
	endpoint string
	logger   *slog.Logger
}

func (r *TokenAuthRequester[Request, Response]) Get(ctx context.Context, path string) (Response, error) {
	return r.do(ctx, http.MethodGet, path, nil)
}

func (r *TokenAuthRequester[Request, Response]) Put(ctx context.Context, path string, request *Request) (Response, error) {
	return r.do(ctx, http.MethodPut, path, request)
}

func (r *TokenAuthRequester[Request, Response]) Post(ctx context.Context, path string, request *Request) (Response, error) {
	return r.do(ctx, http.MethodPost, path, request)
}

func (r *TokenAuthRequester[Request, Response]) Delete(ctx context.Context, path string) (Response, error) {
	return r.do(ctx, http.MethodDelete, path, nil)
}

func (r *TokenAuthRequester[Request, Response]) do(ctx context.Context, method string, path string, request *Request) (Response, error) {
	var zero Response

	body, err := r.marshalRequest(method, request)
	if err != nil {
		return zero, fmt.Errorf("marshal request: %w", err)
	}

	var fullPath string
	if path != "" {
		fullPath, err = url.JoinPath(r.endpoint, path)
		if err != nil {
			return zero, err
		}
	} else {
		fullPath = r.endpoint
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		fullPath,
		body,
	)
	if err != nil {
		return zero, fmt.Errorf("create request: %w", err)
	}

	r.setHeaders(req)

	resp, err := r.client.Do(req)
	if err != nil {
		return zero, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if r.logger != nil {
		r.logger.Info("http request completed",
			slog.String("method", method),
			slog.String("url", fullPath),
			slog.Int("status", resp.StatusCode),
		)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return zero, newHttpError(resp.StatusCode)
	}

	if resp.Body == nil {
		return zero, nil
	}

	if err := json.NewDecoder(resp.Body).Decode(&zero); err != nil {
		if err == io.EOF {
			return zero, nil
		}
		return zero, fmt.Errorf("decode response: %w", err)
	}

	return zero, nil
}

func (r *TokenAuthRequester[Request, Response]) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Token", r.token)
	req.Header.Set("accept", "application/json")
}

func (r *TokenAuthRequester[Request, Response]) marshalRequest(method string, request *Request) (io.Reader, error) {
	if method == http.MethodGet || method == http.MethodDelete || request == nil {
		return nil, nil
	}

	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(data), nil
}
