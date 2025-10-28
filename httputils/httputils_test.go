package httputils

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUrlBuilder(t *testing.T) {
	type urlAndEndpoint struct {
		url      string
		endpoint string
		expected string
	}

	urlsAndEndpoints := []urlAndEndpoint{
		{url: "http://10.1.2.3", endpoint: "/fabric_manager/api/v1", expected: "http://10.1.2.3/fabric_manager/api/v1"},
		{url: "http://10.1.2.3/", endpoint: "/fabric_manager/api/v1", expected: "http://10.1.2.3/fabric_manager/api/v1"},
		{url: "http://10.1.2.3/", endpoint: "", expected: "http://10.1.2.3"},
		{url: "", endpoint: "/fabric_manager/api/v1", expected: "/fabric_manager/api/v1"},
		{url: "", endpoint: "", expected: ""},
		{url: "/", endpoint: "", expected: ""},
	}

	for _, uae := range urlsAndEndpoints {
		t.Run(fmt.Sprintf("test: url: '[%s]', endpoint: '[%s]'", uae.url, uae.endpoint),
			func(t *testing.T) {
				observed := UrlBuilder(uae.url, uae.endpoint)
				if observed != uae.expected {
					t.Errorf("Error: expected '%s' but got '[%s]", uae.expected, observed)
				}
			})
	}
}

type Response struct {
	Message string `json:"message"`
}

func TestStandardCdiHTTPClient_Post_Success(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/test", r.URL.Path)
		assert.Equal(t, "someTenant", r.URL.Query().Get("tenantID"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
		response := Response{Message: "success"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewStandardCdiHTTPClient(server.URL)
	payload := []byte(`{"key": "value"}`)
	queryParams := map[string]string{"tenantID": "someTenant"}
	headers := map[string]string{"Content-Type": "application/json"}
	var response Response

	statusCode, err := client.Post(payload, "/test", queryParams, &response, headers)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
	assert.Equal(t, "success", response.Message)
}

func TestStandardCdiHTTPClient_Post_DecodeError(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": 123}`)) // Incorrect type - int instead of string
	}))
	defer server.Close()

	client := NewStandardCdiHTTPClient(server.URL)
	payload := []byte(`{"key": "value"}`)
	var response Response

	statusCode, err := client.Post(payload, "/test", nil, &response, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling JSON response")
	assert.Equal(t, http.StatusOK, statusCode)
}

func TestStandardCdiHTTPClient_Post_ErrorResponse(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"message": "conflict"}`))
	}))
	defer server.Close()

	client := NewStandardCdiHTTPClient(server.URL)
	payload := []byte(`{"key": "value"}`)
	var response Response

	statusCode, err := client.Post(payload, "/test", nil, &response, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), `request failed: {"message": "conflict"}`)
	assert.Equal(t, http.StatusConflict, statusCode)
}

func TestStandardCdiHTTPClient_Post_NetworkError(t *testing.T) {
	client := NewStandardCdiHTTPClient("invalid-url")        // Invalid URL to cause a send error
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	payload := []byte(`{"key": "value"}`)
	var response Response
	client.Client.Timeout = 100 * time.Millisecond

	statusCode, err := client.Post(payload, "/test", nil, &response, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sending request")
	assert.Equal(t, -1, statusCode) // -1 indicates an error before receiving a response
}

func TestStandardCdiHTTPClient_Put_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/test", r.URL.Path)
		assert.Equal(t, "someTenant", r.URL.Query().Get("tenantID"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
		response := Response{Message: "success"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewStandardCdiHTTPClient(server.URL)
	payload := []byte(`{"key": "value"}`)
	queryParams := map[string]string{"tenantID": "someTenant"}
	headers := map[string]string{"Content-Type": "application/json"}
	var response Response

	statusCode, err := client.Put(payload, "/test", queryParams, &response, headers)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
	assert.Equal(t, "success", response.Message)
}

func TestStandardCdiHTTPClient_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/test", r.URL.Path)
		assert.Equal(t, "someTenant", r.URL.Query().Get("tenantID"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	client := NewStandardCdiHTTPClient(server.URL)
	queryParams := map[string]string{"tenantID": "someTenant"}
	headers := map[string]string{"Content-Type": "application/json"}
	var response Response

	statusCode, err := client.Get("/test", queryParams, &response, headers)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
	assert.Equal(t, "success", response.Message)
}

func TestStandardCdiHTTPClient_Delete_SuccessWithNilResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/test", r.URL.Path)
		assert.Equal(t, "someTenant", r.URL.Query().Get("tenantID"))

		w.WriteHeader(http.StatusNoContent) // 204 No Content
	}))
	defer server.Close()

	client := NewStandardCdiHTTPClient(server.URL)
	queryParams := map[string]string{"tenantID": "someTenant"}
	headers := map[string]string{"Content-Type": "application/json"}

	statusCode, err := client.Delete("/test", queryParams, nil, headers) // nil response

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, statusCode)
}

func TestStandardCdiHTTPClient_doRequest_ParseURLError(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	client := NewStandardCdiHTTPClient(":invalid-url")
	payload := []byte(`{"key": "value"}`)
	var response Response

	statusCode, err := client.doRequest("", "/test", payload, nil, &response, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing URL failed")
	assert.Equal(t, -1, statusCode) // -1 indicates an error before sending a request
}

func TestStandardCdiHTTPClient_doRequest_CreateRequestError(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // Suppress slog output in test
	client := NewStandardCdiHTTPClient("http://example.com")
	// Using an invalid method to force an error during request creation
	statusCode, err := client.doRequest("INVALID@METHOD", "/test", nil, nil, nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creating request failed")
	assert.Equal(t, -1, statusCode)
}

func TestGetAuthorizationHeader(t *testing.T) {
	bearerToken := "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJjT21qTFYwQVhCWVFycFJBMnh5MVh1akRWX250TTZQY3dyZ2ZWTWJyRGRFIn0.eyJleHAiOjE3MzE1MDk3ODMsImlhdCI6MTczMTUwOTcyMywianRpIjoiNTEzNjRjMTYtNzJjZi00ZmJiLTk0YTctYTE4MzUyZGQwYzM4IiwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL3JlYWxtcy9tYXN0ZXIiLCJzdWIiOiIxM2E5MjE3Zi1kYjIzLTQ4YTYtOWVmYS1hYmNkYWM4MWZmZWQiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhZG1pbi1jbGkiLCJzZXNzaW9uX3N0YXRlIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIiwiYWNyIjoiMSIsInNjb3BlIjoicHJvZmlsZSBlbWFpbCIsInNpZCI6ImQwNmY4ZDVmLTQxNDYtNGE3Ny05MTZkLTUwNTg0NjY2ZGNhYiIsImVtYWlsX3ZlcmlmaWVkIjpmYWxzZSwicHJlZmVycmVkX3VzZXJuYW1lIjoiYWRtaW4ifQ.X_sIDgZNKn-3cu-aOS0vwgF2a0DBldP4PjHaJtfnZzq4744C3MSN5YDtYeqNOn3-pgwS-yTKArTLqJZgwGk3Edv4oqVc59uMDWRfATzS9JQh_NMI8ZvxapHCBwIlpkc0xtTqu-bGbuswfH5QhDWwcny5Et3LMOtu6KOVuscdKnFRgQpHOcyeT7LehdAVhbRb1ZGOsTiAsOpR3E8wAU3SzqXQwXbfvK5pixzGxbjjOPc7MN5HWPwXamMoOZQTLsxCAEqe1X138LkPmWXhV4b9bU6hrfmic27C18ME6QbcY45UxnkOMsy0IXhc_d5GLsEZGTIy5za7zlE8QDXZm0pjOw"
	expectedBearerTokenMap := map[string]string{"Authorization": "Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJjT21qTFYwQVhCWVFycFJBMnh5MVh1akRWX250TTZQY3dyZ2ZWTWJyRGRFIn0.eyJleHAiOjE3MzE1MDk3ODMsImlhdCI6MTczMTUwOTcyMywianRpIjoiNTEzNjRjMTYtNzJjZi00ZmJiLTk0YTctYTE4MzUyZGQwYzM4IiwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL3JlYWxtcy9tYXN0ZXIiLCJzdWIiOiIxM2E5MjE3Zi1kYjIzLTQ4YTYtOWVmYS1hYmNkYWM4MWZmZWQiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhZG1pbi1jbGkiLCJzZXNzaW9uX3N0YXRlIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIiwiYWNyIjoiMSIsInNjb3BlIjoicHJvZmlsZSBlbWFpbCIsInNpZCI6ImQwNmY4ZDVmLTQxNDYtNGE3Ny05MTZkLTUwNTg0NjY2ZGNhYiIsImVtYWlsX3ZlcmlmaWVkIjpmYWxzZSwicHJlZmVycmVkX3VzZXJuYW1lIjoiYWRtaW4ifQ.X_sIDgZNKn-3cu-aOS0vwgF2a0DBldP4PjHaJtfnZzq4744C3MSN5YDtYeqNOn3-pgwS-yTKArTLqJZgwGk3Edv4oqVc59uMDWRfATzS9JQh_NMI8ZvxapHCBwIlpkc0xtTqu-bGbuswfH5QhDWwcny5Et3LMOtu6KOVuscdKnFRgQpHOcyeT7LehdAVhbRb1ZGOsTiAsOpR3E8wAU3SzqXQwXbfvK5pixzGxbjjOPc7MN5HWPwXamMoOZQTLsxCAEqe1X138LkPmWXhV4b9bU6hrfmic27C18ME6QbcY45UxnkOMsy0IXhc_d5GLsEZGTIy5za7zlE8QDXZm0pjOw"}

	assert.EqualValues(t, GetAuthorizationHeader(bearerToken), expectedBearerTokenMap)
}

func TestGetAuthorizationHeaderWithContentType(t *testing.T) {
	bearerToken := "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJjT21qTFYwQVhCWVFycFJBMnh5MVh1akRWX250TTZQY3dyZ2ZWTWJyRGRFIn0.eyJleHAiOjE3MzE1MDk3ODMsImlhdCI6MTczMTUwOTcyMywianRpIjoiNTEzNjRjMTYtNzJjZi00ZmJiLTk0YTctYTE4MzUyZGQwYzM4IiwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL3JlYWxtcy9tYXN0ZXIiLCJzdWIiOiIxM2E5MjE3Zi1kYjIzLTQ4YTYtOWVmYS1hYmNkYWM4MWZmZWQiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhZG1pbi1jbGkiLCJzZXNzaW9uX3N0YXRlIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIiwiYWNyIjoiMSIsInNjb3BlIjoicHJvZmlsZSBlbWFpbCIsInNpZCI6ImQwNmY4ZDVmLTQxNDYtNGE3Ny05MTZkLTUwNTg0NjY2ZGNhYiIsImVtYWlsX3ZlcmlmaWVkIjpmYWxzZSwicHJlZmVycmVkX3VzZXJuYW1lIjoiYWRtaW4ifQ.X_sIDgZNKn-3cu-aOS0vwgF2a0DBldP4PjHaJtfnZzq4744C3MSN5YDtYeqNOn3-pgwS-yTKArTLqJZgwGk3Edv4oqVc59uMDWRfATzS9JQh_NMI8ZvxapHCBwIlpkc0xtTqu-bGbuswfH5QhDWwcny5Et3LMOtu6KOVuscdKnFRgQpHOcyeT7LehdAVhbRb1ZGOsTiAsOpR3E8wAU3SzqXQwXbfvK5pixzGxbjjOPc7MN5HWPwXamMoOZQTLsxCAEqe1X138LkPmWXhV4b9bU6hrfmic27C18ME6QbcY45UxnkOMsy0IXhc_d5GLsEZGTIy5za7zlE8QDXZm0pjOw"
	expectedBearerTokenMap := map[string]string{"Content-Type": "application/json", "Authorization": "Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJjT21qTFYwQVhCWVFycFJBMnh5MVh1akRWX250TTZQY3dyZ2ZWTWJyRGRFIn0.eyJleHAiOjE3MzE1MDk3ODMsImlhdCI6MTczMTUwOTcyMywianRpIjoiNTEzNjRjMTYtNzJjZi00ZmJiLTk0YTctYTE4MzUyZGQwYzM4IiwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwL3JlYWxtcy9tYXN0ZXIiLCJzdWIiOiIxM2E5MjE3Zi1kYjIzLTQ4YTYtOWVmYS1hYmNkYWM4MWZmZWQiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhZG1pbi1jbGkiLCJzZXNzaW9uX3N0YXRlIjoiZDA2ZjhkNWYtNDE0Ni00YTc3LTkxNmQtNTA1ODQ2NjZkY2FiIiwiYWNyIjoiMSIsInNjb3BlIjoicHJvZmlsZSBlbWFpbCIsInNpZCI6ImQwNmY4ZDVmLTQxNDYtNGE3Ny05MTZkLTUwNTg0NjY2ZGNhYiIsImVtYWlsX3ZlcmlmaWVkIjpmYWxzZSwicHJlZmVycmVkX3VzZXJuYW1lIjoiYWRtaW4ifQ.X_sIDgZNKn-3cu-aOS0vwgF2a0DBldP4PjHaJtfnZzq4744C3MSN5YDtYeqNOn3-pgwS-yTKArTLqJZgwGk3Edv4oqVc59uMDWRfATzS9JQh_NMI8ZvxapHCBwIlpkc0xtTqu-bGbuswfH5QhDWwcny5Et3LMOtu6KOVuscdKnFRgQpHOcyeT7LehdAVhbRb1ZGOsTiAsOpR3E8wAU3SzqXQwXbfvK5pixzGxbjjOPc7MN5HWPwXamMoOZQTLsxCAEqe1X138LkPmWXhV4b9bU6hrfmic27C18ME6QbcY45UxnkOMsy0IXhc_d5GLsEZGTIy5za7zlE8QDXZm0pjOw"}

	assert.EqualValues(t, GetAuthorizationHeaderWithContentType(bearerToken), expectedBearerTokenMap)
}
