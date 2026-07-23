package routeregistry

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/llnl/wormhole-cli/test/mocks/mock_http"
	"github.com/llnl/wormhole-cli/test/testutil"
)

func setupRegistryClient(t *testing.T, token, endpoint string) (*RegistryClient, *mock_http.MockRoundTripper) {
	ctrl := gomock.NewController(t)
	mockRT := mock_http.NewMockRoundTripper(ctrl)
	client := NewRegistryClient(token, endpoint, &http.Client{Transport: mockRT}, nil)
	return client, mockRT
}

func assertError(t *testing.T, err error, wantErr bool) {
	t.Helper()
	if wantErr {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
	}
}

func expectRoundTrip(mockRT *mock_http.MockRoundTripper, t *testing.T, method, path string, resp *http.Response, err error) {
	mockRT.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, method, req.Method)
		assert.Equal(t, path, req.URL.Path)
		return resp, err
	})
}

func TestRegistryClient_Communities(t *testing.T) {
	ctx := t.Context()

	t.Run("AddAndRemove", func(t *testing.T) {
		tests := map[string]struct {
			action   func(client *RegistryClient, ctx context.Context) error
			mockFunc func(mockRT *mock_http.MockRoundTripper, t *testing.T)
		}{
			"AddCommunity": {
				action: func(c *RegistryClient, ctx context.Context) error {
					comm, err := c.AddCommunity(ctx, "test-comm")
					if err != nil {
						return err
					}
					assert.Equal(t, "test-comm", comm.Name)
					assert.Equal(t, "comm-12345678", *comm.ID)
					return nil
				},
				mockFunc: func(mockRT *mock_http.MockRoundTripper, t *testing.T) {
					mockRT.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(func(req *http.Request) (*http.Response, error) {
						assert.Equal(t, http.MethodPost, req.Method)
						assert.Equal(t, "/api/v1/community", req.URL.Path)
						var body Community
						err := json.NewDecoder(req.Body).Decode(&body)
						assert.NoError(t, err)
						assert.Equal(t, "test-comm", body.Name)
						return testutil.NewMockResponse(http.StatusOK, `{"name": "test-comm", "id": "comm-12345678"}`), nil
					})
				},
			},
			"RemoveCommunity": {
				action: func(c *RegistryClient, ctx context.Context) error {
					return c.RemoveCommunity(ctx, "comm-12345678")
				},
				mockFunc: func(mockRT *mock_http.MockRoundTripper, t *testing.T) {
					mockRT.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(func(req *http.Request) (*http.Response, error) {
						assert.Equal(t, http.MethodDelete, req.Method)
						assert.Equal(t, "/api/v1/community/comm-12345678", req.URL.Path)
						return testutil.NewMockResponse(http.StatusOK, ""), nil
					})
				},
			},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				client, mockRT := setupRegistryClient(t, testutil.TestToken, testutil.TestEndpointUrl)
				tt.mockFunc(mockRT, t)
				assert.NoError(t, tt.action(client, ctx))
			})
		}
	})

	t.Run("ListCommunities", func(t *testing.T) {
		tests := map[string]struct {
			filter         string
			mockResponse   string
			responseStatus int
			expectedCount  int
			expectedName   string
			wantErr        bool
		}{
			"All": {
				filter:         "",
				mockResponse:   `[{"name": "comm1", "id": "id12345678"}, {"name": "comm2", "id": "id87654321"}]`,
				responseStatus: http.StatusOK,
				expectedCount:  2,
				wantErr:        false,
			},
			"Filtered": {
				filter:         "comm1",
				mockResponse:   `[{"name": "comm1", "id": "id12345678"}, {"name": "comm2", "id": "id87654321"}]`,
				responseStatus: http.StatusOK,
				expectedCount:  1,
				expectedName:   "comm1",
				wantErr:        false,
			},
			"NotFound": {
				filter:         "comm2",
				mockResponse:   `[{"name": "comm1", "id": "id12345678"}]`,
				responseStatus: http.StatusOK,
				wantErr:        true,
			},
			"Unauthorized": {
				filter:         "",
				mockResponse:   `{"error": "unauthorized"}`,
				responseStatus: http.StatusUnauthorized,
				wantErr:        true,
			},
			"ServerError": {
				filter:         "",
				mockResponse:   `{"error": "internal server error"}`,
				responseStatus: http.StatusInternalServerError,
				wantErr:        true,
			},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				client, mockRT := setupRegistryClient(t, testutil.TestToken, testutil.TestEndpointUrl)
				mockRT.EXPECT().RoundTrip(gomock.Any()).Return(testutil.NewMockResponse(tt.responseStatus, tt.mockResponse), nil)

				comms, err := client.ListCommunities(ctx, tt.filter)
				assertError(t, err, tt.wantErr)
				if !tt.wantErr {
					assert.Equal(t, tt.expectedCount, len(comms))
					if tt.expectedName != "" && len(comms) > 0 {
						assert.Equal(t, tt.expectedName, comms[0].Name)
					}
				}
			})
		}
	})

	t.Run("ResolveCommunity", func(t *testing.T) {
		tests := map[string]struct {
			id             string
			mockResponse   string
			responseStatus int
			expectedName   string
			wantErr        bool
		}{
			"Success": {
				id:             "comm1",
				mockResponse:   `[{"name": "comm1", "id": "id12345678"}]`,
				responseStatus: http.StatusOK,
				expectedName:   "comm1",
				wantErr:        false,
			},
			"NotUnique": {
				id:             "id123",
				mockResponse:   `[{"name": "comm1", "id": "id12345678"}, {"name": "comm1-alt", "id": "id12345679"}]`,
				responseStatus: http.StatusOK,
				wantErr:        true,
			},
			"Unauthorized": {
				id:             "comm1",
				mockResponse:   `{"error": "unauthorized"}`,
				responseStatus: http.StatusUnauthorized,
				wantErr:        true,
			},
			"ServerError": {
				id:             "comm1",
				mockResponse:   `{"error": "internal server error"}`,
				responseStatus: http.StatusInternalServerError,
				wantErr:        true,
			},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				client, mockRT := setupRegistryClient(t, testutil.TestToken, testutil.TestEndpointUrl)
				mockRT.EXPECT().RoundTrip(gomock.Any()).Return(testutil.NewMockResponse(tt.responseStatus, tt.mockResponse), nil)

				comm, err := client.ResolveCommunity(ctx, tt.id)
				assertError(t, err, tt.wantErr)
				if !tt.wantErr {
					assert.Equal(t, tt.expectedName, comm.Name)
				}
			})
		}
	})
}

func TestRegistryClient_Routes(t *testing.T) {
	ctx := t.Context()

	t.Run("RegisterRoute", func(t *testing.T) {
		tests := map[string]struct {
			responseStatus int
			mockResponse   string
			wantErr        bool
			expectedURL    string
		}{
			"Success": {
				responseStatus: http.StatusOK,
				mockResponse:   `{"url": "http://tunnel.test", "airlock": {}, "tunnel": {}}`,
				wantErr:        false,
				expectedURL:    "http://tunnel.test",
			},
			"Unauthorized": {
				responseStatus: http.StatusUnauthorized,
				mockResponse:   `{"error": "unauthorized"}`,
				wantErr:        true,
			},
			"ServerError": {
				responseStatus: http.StatusInternalServerError,
				mockResponse:   `{"error": "internal server error"}`,
				wantErr:        true,
			},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				client, mockRT := setupRegistryClient(t, testutil.TestToken, testutil.TestEndpointUrl)
				mockRT.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, http.MethodPost, req.Method)
					assert.Equal(t, "/api/v1/route", req.URL.Path)

					var body RegistrationRequest
					err := json.NewDecoder(req.Body).Decode(&body)
					assert.NoError(t, err)
					assert.Equal(t, "test-route", body.Name)

					return testutil.NewMockResponse(tt.responseStatus, tt.mockResponse), nil
				})

				resp, err := client.RegisterRoute(ctx, "test-comm", "test-route")
				assertError(t, err, tt.wantErr)
				if !tt.wantErr {
					assert.Equal(t, tt.expectedURL, resp.URL)
				}
			})
		}
	})

	t.Run("ListRoutes", func(t *testing.T) {
		tests := map[string]struct {
			filter         string
			mockResponse   string
			responseStatus int
			expectedCount  int
			wantErr        bool
		}{
			"All": {
				filter:         "",
				mockResponse:   `[{"name": "route1", "id": "rid12345678"}, {"name": "route2", "id": "rid87654321"}]`,
				responseStatus: http.StatusOK,
				expectedCount:  2,
				wantErr:        false,
			},
			"Filtered": {
				filter:         "rid12345",
				mockResponse:   `[{"name": "route1", "id": "rid12345678"}, {"name": "route2", "id": "rid87654321"}]`,
				responseStatus: http.StatusOK,
				expectedCount:  1,
				wantErr:        false,
			},
			"Unauthorized": {
				filter:         "",
				mockResponse:   `{"error": "unauthorized"}`,
				responseStatus: http.StatusUnauthorized,
				wantErr:        true,
			},
			"ServerError": {
				filter:         "",
				mockResponse:   `{"error": "internal server error"}`,
				responseStatus: http.StatusInternalServerError,
				wantErr:        true,
			},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				client, mockRT := setupRegistryClient(t, testutil.TestToken, testutil.TestEndpointUrl)
				mockRT.EXPECT().RoundTrip(gomock.Any()).Return(testutil.NewMockResponse(tt.responseStatus, tt.mockResponse), nil)

				routes, err := client.ListRoutes(ctx, tt.filter)
				assertError(t, err, tt.wantErr)
				if !tt.wantErr {
					assert.Equal(t, tt.expectedCount, len(routes))
				}
			})
		}
	})

	t.Run("ResolveRoute", func(t *testing.T) {
		tests := map[string]struct {
			id             string
			mockResponse   string
			responseStatus int
			expectedName   string
			wantErr        bool
		}{
			"Success": {
				id:             "rid12345678",
				mockResponse:   `[{"name": "route1", "id": "rid12345678"}]`,
				responseStatus: http.StatusOK,
				expectedName:   "route1",
				wantErr:        false,
			},
			"NotUnique": {
				id:             "rid123",
				mockResponse:   `[{"name": "route1", "id": "rid12345678"}, {"name": "route1-alt", "id": "rid12345679"}]`,
				responseStatus: http.StatusOK,
				wantErr:        true,
			},
			"Unauthorized": {
				id:             "rid12345678",
				mockResponse:   `{"error": "unauthorized"}`,
				responseStatus: http.StatusUnauthorized,
				wantErr:        true,
			},
			"ServerError": {
				id:             "rid12345678",
				mockResponse:   `{"error": "internal server error"}`,
				responseStatus: http.StatusInternalServerError,
				wantErr:        true,
			},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				client, mockRT := setupRegistryClient(t, testutil.TestToken, testutil.TestEndpointUrl)
				mockRT.EXPECT().RoundTrip(gomock.Any()).Return(testutil.NewMockResponse(tt.responseStatus, tt.mockResponse), nil)

				route, err := client.ResolveRoute(ctx, tt.id)
				assertError(t, err, tt.wantErr)
				if !tt.wantErr {
					assert.Equal(t, tt.expectedName, route.Name)
				}
			})
		}
	})
}

func TestRegistryClient_CommunityRoutes(t *testing.T) {
	ctx := t.Context()

	tests := map[string]struct {
		method         string
		path           string
		mockResponse   string
		responseStatus int
		wantErr        bool
		fn             func(client *RegistryClient, ctx context.Context, commID, routeID string) error
	}{
		"AddCommunityRoute": {
			method:         http.MethodPut,
			path:           "/api/v1/community/comm-id/route/route-id",
			mockResponse:   `{}`,
			responseStatus: http.StatusOK,
			wantErr:        false,
			fn: func(c *RegistryClient, ctx context.Context, commID, routeID string) error {
				return c.AddCommunityRoute(ctx, commID, routeID)
			},
		},
		"RemoveCommunityRoute": {
			method:         http.MethodDelete,
			path:           "/api/v1/community/comm-id/route/route-id",
			mockResponse:   ``,
			responseStatus: http.StatusOK,
			wantErr:        false,
			fn: func(c *RegistryClient, ctx context.Context, commID, routeID string) error {
				return c.RemoveCommunityRoute(ctx, commID, routeID)
			},
		},
		"AddCommunityRouteUnauthorized": {
			method:         http.MethodPut,
			path:           "/api/v1/community/comm-id/route/route-id",
			mockResponse:   `{"error": "unauthorized"}`,
			responseStatus: http.StatusUnauthorized,
			wantErr:        true,
			fn: func(c *RegistryClient, ctx context.Context, commID, routeID string) error {
				return c.AddCommunityRoute(ctx, commID, routeID)
			},
		},
		"RemoveCommunityRouteServerError": {
			method:         http.MethodDelete,
			path:           "/api/v1/community/comm-id/route/route-id",
			mockResponse:   `{"error": "internal server error"}`,
			responseStatus: http.StatusInternalServerError,
			wantErr:        true,
			fn: func(c *RegistryClient, ctx context.Context, commID, routeID string) error {
				return c.RemoveCommunityRoute(ctx, commID, routeID)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			client, mockRT := setupRegistryClient(t, testutil.TestToken, testutil.TestEndpointUrl)
			expectRoundTrip(mockRT, t, tt.method, tt.path, testutil.NewMockResponse(tt.responseStatus, tt.mockResponse), nil)

			err := tt.fn(client, ctx, "comm-id", "route-id")
			assertError(t, err, tt.wantErr)
		})
	}
}
