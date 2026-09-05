package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	testhelper "github.com/MamangRust/monolith-graphql-apigateway/testhelper"
	"github.com/MamangRust/monolith-payment-gateway-pkg/auth"
	"github.com/MamangRust/monolith-payment-gateway-pkg/logger"
	"github.com/redis/go-redis/v9"
)

// GraphQLTestClient wraps an HTTP handler for sending GraphQL requests.
type GraphQLTestClient struct {
	Handler http.Handler
}

// NewGraphQLTestClient creates a new GraphQL test client from gRPC connections.
func NewGraphQLTestClient(conns *testhelper.ServiceConnections, log logger.LoggerInterface, redisClient *redis.Client) *GraphQLTestClient {
	resolver := testhelper.NewResolverWithRedis(conns, log, redisClient)
	h := testhelper.NewGraphQLHTTPHandler(resolver)
	return &GraphQLTestClient{Handler: h}
}

// NewGraphQLTestClientWithAuth creates a new GraphQL test client with auth middleware.
func NewGraphQLTestClientWithAuth(conns *testhelper.ServiceConnections, log logger.LoggerInterface, redisClient *redis.Client, tokenManager auth.TokenManager) *GraphQLTestClient {
	resolver := testhelper.NewResolverWithRedis(conns, log, redisClient)
	h := testhelper.NewGraphQLHTTPHandler(resolver)
	authedHandler := AuthMiddleware(tokenManager, log)(h)
	return &GraphQLTestClient{Handler: authedHandler}
}

// GraphQLRequest represents a GraphQL request body.
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL response body.
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error.
type GraphQLError struct {
	Message string `json:"message"`
}

// Do sends a GraphQL request and returns the response recorder and parsed response.
func (c *GraphQLTestClient) Do(query string) (*httptest.ResponseRecorder, *GraphQLResponse) {
	body := GraphQLRequest{Query: query}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	c.Handler.ServeHTTP(rec, req)

	var resp GraphQLResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	return rec, &resp
}

// DoWithHeader sends a GraphQL request with custom headers and returns the response.
func (c *GraphQLTestClient) DoWithHeader(query string, headers map[string]string) (*httptest.ResponseRecorder, *GraphQLResponse) {
	body := GraphQLRequest{Query: query}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()

	c.Handler.ServeHTTP(rec, req)

	var resp GraphQLResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	return rec, &resp
}

type userIDContextKey struct{}

// DummyConns creates a ServiceConnections with dummy connections for all services.
func DummyConns() *testhelper.ServiceConnections {
	return &testhelper.ServiceConnections{
		AuthClient:        testhelper.CreateDummyConn(),
		RoleClient:        testhelper.CreateDummyConn(),
		UserClient:        testhelper.CreateDummyConn(),
		CardClient:        testhelper.CreateDummyConn(),
		MerchantClient:    testhelper.CreateDummyConn(),
		SaldoClient:       testhelper.CreateDummyConn(),
		TopupClient:       testhelper.CreateDummyConn(),
		TransactionClient: testhelper.CreateDummyConn(),
		TransferClient:    testhelper.CreateDummyConn(),
		WithdrawClient:    testhelper.CreateDummyConn(),
	}
}

// ParseGraphQLData unmarshals the data field of a GraphQL response into the target.
func ParseGraphQLData(resp *GraphQLResponse, target interface{}) error {
	if resp.Data == nil {
		return fmt.Errorf("no data in response")
	}
	return json.Unmarshal(resp.Data, target)
}

// GetResponseBody reads and returns the body from a response recorder.
func GetResponseBody(rec *httptest.ResponseRecorder) string {
	body, _ := io.ReadAll(rec.Body)
	return string(body)
}

// Background returns a context.Background().
func Background() context.Context {
	return context.Background()
}

// AuthMiddleware extracts user ID from Bearer token and sets it in context.
func AuthMiddleware(tm auth.TokenManager, logger logger.LoggerInterface) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for public operations
			bodyBytes, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			var bodyMap map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &bodyMap); err == nil {
				if q, ok := bodyMap["query"].(string); ok {
					queryLower := strings.ToLower(q)
					if strings.Contains(queryLower, "loginuser") ||
						strings.Contains(queryLower, "registeruser") ||
						strings.Contains(queryLower, "refreshtoken") {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				next.ServeHTTP(w, r)
				return
			}

			userIDStr, err := tm.ValidateToken(parts[1])
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			userID, _ := strconv.Atoi(userIDStr)
			ctx := context.WithValue(r.Context(), userIDContextKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
