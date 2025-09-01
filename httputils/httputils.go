package httputils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	slog "github.com/fujitsu/docker-machine-driver-fsas/logger"
)

// UrlBuilder constructs a complete URL by combining a base URL and an endpoint.
// It ensures that the base URL does not end with a trailing slash ('/') before concatenating it with the endpoint.
// This prevents the creation of URLs with double slashes (e.g., "http://10.1.2.3//api/v1/users").
//
// Parameters:
//
//	url: The base URL (e.g., "http://10.1.2.3").
//	endpoint: The endpoint to append to the base URL (e.g., "/users", "/products/123").
//
// Returns:
//
//	A complete URL string.
func UrlBuilder(url, endpoint string) string {
	url = strings.TrimSuffix(url, "/")
	return fmt.Sprintf("%s%s", url, endpoint)
}

type CdiHTTPClient interface {
	Post(payload []byte, endpoint string, queryParams map[string]string, responseAddress any, headers map[string]string) (int, error)
	Put(payload []byte, endpoint string, queryParams map[string]string, responseAddress any, headers map[string]string) (int, error)
	Delete(endpoint string, queryParams map[string]string, responseAddress any, headers map[string]string) (int, error)
	Get(endpoint string, queryParams map[string]string, responseAddress any, headers map[string]string) (int, error)
}

type StandardCdiHTTPClient struct {
	BaseURI string
	Client  *http.Client
}

// This makes StandardCdiHTTPClient implement the CdiHTTPClient interface
var _ CdiHTTPClient = (*StandardCdiHTTPClient)(nil)

func NewStandardCdiHTTPClient(baseURI string) *StandardCdiHTTPClient {
	return &StandardCdiHTTPClient{
		BaseURI: baseURI,
		Client:  http.DefaultClient,
	}
}

func (c *StandardCdiHTTPClient) doRequest(method, endpoint string, payload []byte, queryParams map[string]string, responseAddress any, headers map[string]string) (int, error) {
	slog.Debug(fmt.Sprintf("Initiating %s request: ", method), "endpoint", endpoint, "payload", string(payload))

	// Construct full URL
	u, err := url.Parse(c.BaseURI + endpoint)
	if err != nil {
		return -1, fmt.Errorf("parsing URL failed: %w", err)
	}

	q := u.Query()
	for k, v := range queryParams {
		q.Add(k, v)
	}
	u.RawQuery = q.Encode()

	slog.Debug("Generated full URL: ", "url", u.String())

	// Send the request
	req, err := http.NewRequest(method, u.String(), bytes.NewBuffer(payload))
	if err != nil {
		slog.Error("Error creating request: ", "error", err)
		return -1, fmt.Errorf("creating request failed: %w", err)
	}

	// Add headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	slog.Debug(fmt.Sprintf("Sending %s request: ", method), "url", u.String(), "headers", req.Header)

	resp, err := c.Client.Do(req)
	if err != nil {
		slog.Error("Error sending request: ", "error", err)
		return -1, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	slog.Debug("Received response: ", "status_code", resp.StatusCode)

	// Handle response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Error reading response body: ", "error", err)
		return resp.StatusCode, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		slog.Error("Request failed: ", "status_code", resp.StatusCode, "response_body", string(body))
		return resp.StatusCode, fmt.Errorf("request failed: %s", string(body))
	}

	slog.Debug("Received response: ", "status_code", resp.StatusCode, "response_body", string(body))

	if responseAddress != nil {
		slog.Debug("Decoding response body")
		err := json.Unmarshal(body, responseAddress)
		if err != nil {
			slog.Error("Error unmarshalling JSON response: ", "err", err)
			return resp.StatusCode, fmt.Errorf("unmarshalling JSON response: %w", err)
		}
	}

	slog.Debug(fmt.Sprintf("%s request completed successfully", method))
	return resp.StatusCode, nil
}

func (c *StandardCdiHTTPClient) Post(payload []byte, endpoint string, queryParams map[string]string, responseAddress any, headers map[string]string) (int, error) {
	return c.doRequest(http.MethodPost, endpoint, payload, queryParams, responseAddress, headers)
}

func (c *StandardCdiHTTPClient) Put(payload []byte, endpoint string, queryParams map[string]string, responseAddress any, headers map[string]string) (int, error) {
	return c.doRequest(http.MethodPut, endpoint, payload, queryParams, responseAddress, headers)
}

func (c *StandardCdiHTTPClient) Get(endpoint string, queryParams map[string]string, responseAddress any, headers map[string]string) (int, error) {
	return c.doRequest(http.MethodGet, endpoint, nil, queryParams, responseAddress, headers)
}

func (c *StandardCdiHTTPClient) Delete(endpoint string, queryParams map[string]string, responseAddress any, headers map[string]string) (int, error) {
	return c.doRequest(http.MethodDelete, endpoint, nil, queryParams, responseAddress, headers)
}

func GetAuthorizationHeader(bearerToken string) map[string]string {
	return map[string]string{"Authorization": fmt.Sprintf("Bearer %s", bearerToken)}
}

func GetAuthorizationHeaderWithContentType(bearerToken string) map[string]string {
	headers := GetAuthorizationHeader(bearerToken)
	headers["Content-Type"] = "application/json"
	return headers
}
