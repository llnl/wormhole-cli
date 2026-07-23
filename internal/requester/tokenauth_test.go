package requester

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/llnl/wormhole-cli/test/mocks/mock_http"
	"github.com/llnl/wormhole-cli/test/testutil"
)

func TestTokenAuthRequester_Do(t *testing.T) {
	type testResponse struct {
		Status string `json:"status"`
	}
	tests := map[string]struct {
		method         string
		path           string
		requestBody    any
		responseStatus int
		responseBody   string
		expectedUrl    string
		verifyRequest  func(t *testing.T, req *http.Request)
		wantErr        bool
		expectedErr    string
	}{
		"Get success": {
			method:         http.MethodGet,
			path:           "test",
			responseStatus: http.StatusOK,
			responseBody:   `{"status": "ok"}`,
			expectedUrl:    testutil.TestEndpointUrl + "/test",
			wantErr:        false,
		},
		"Post success": {
			method:         http.MethodPost,
			path:           "test",
			requestBody:    map[string]string{"foo": "bar"},
			responseStatus: http.StatusCreated,
			responseBody:   `{"status": "ok"}`,
			expectedUrl:    testutil.TestEndpointUrl + "/test",
			wantErr:        false,
		},
		"Put success": {
			method:         http.MethodPut,
			path:           "test",
			requestBody:    map[string]string{"foo": "bar"},
			responseStatus: http.StatusOK,
			responseBody:   `{"status": "ok"}`,
			expectedUrl:    testutil.TestEndpointUrl + "/test",
			wantErr:        false,
		},
		"Delete success": {
			method:         http.MethodDelete,
			path:           "test",
			responseStatus: http.StatusOK,
			responseBody:   `{"status": "ok"}`,
			expectedUrl:    testutil.TestEndpointUrl + "/test",
			wantErr:        false,
		},
		"HTTP error response": {
			method:         http.MethodGet,
			path:           "test",
			responseStatus: http.StatusInternalServerError,
			responseBody:   "internal server error",
			expectedUrl:    testutil.TestEndpointUrl + "/test",
			wantErr:        true,
			expectedErr:    "HTTP status 500",
		},
		"Decode error": {
			method:         http.MethodGet,
			path:           "test",
			responseStatus: http.StatusOK,
			responseBody:   `{invalid json`,
			expectedUrl:    testutil.TestEndpointUrl + "/test",
			wantErr:        true,
			expectedErr:    "decode response",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRT := mock_http.NewMockRoundTripper(ctrl)

			mockRT.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(func(req *http.Request) (*http.Response, error) {
				assert.Equal(t, tt.method, req.Method)
				assert.Equal(t, testutil.TestToken, req.Header.Get("X-Token"))
				assert.Equal(t, tt.expectedUrl, req.URL.String())

				if tt.requestBody != nil {
					body, err := io.ReadAll(req.Body)
					assert.NoError(t, err)
					expectedJSON, _ := json.Marshal(tt.requestBody)
					assert.JSONEq(t, string(expectedJSON), string(body))
				}

				if tt.verifyRequest != nil {
					tt.verifyRequest(t, req)
				}
				return testutil.NewMockResponse(tt.responseStatus, tt.responseBody), nil
			})

			client := &http.Client{Transport: mockRT}
			r := NewTokenAuthRequester[any, testResponse](testutil.TestToken, testutil.TestEndpointUrl, client, nil)

			var err error
			ctx := t.Context()
			switch tt.method {
			case http.MethodGet:
				_, err = r.Get(ctx, tt.path)
			case http.MethodPost:
				_, err = r.Post(ctx, tt.path, &tt.requestBody)
			case http.MethodPut:
				_, err = r.Put(ctx, tt.path, &tt.requestBody)
			case http.MethodDelete:
				_, err = r.Delete(ctx, tt.path)
			}

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != "" {
					assert.Contains(t, err.Error(), tt.expectedErr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
