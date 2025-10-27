package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"open_api_to_mcp_server/pkg/config"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// HTTPHandler executes HTTP requests based on tool definitions
type HTTPHandler struct {
	client *http.Client
	auth   *config.AuthConfig
}



// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(auth *config.AuthConfig) *HTTPHandler {
	return &HTTPHandler{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		auth: auth,
	}
}

// Execute performs an HTTP request based on the provided parameters
func (h *HTTPHandler) Execute(ctx context.Context, method, urlPath string, requiresAuth bool, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if err := h.validateMethod(method); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	finalURL := h.buildURL(urlPath, method, args)
	
	req, err := h.createRequest(ctx, method, finalURL, args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("REQUEST_CREATE_FAILED: %v", err)), nil
	}

	if requiresAuth {
		if err := h.addAuthentication(req); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}

	return h.executeRequest(req)
}

func (h *HTTPHandler) validateMethod(method string) error {
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true,
		"PATCH": true, "HEAD": true, "OPTIONS": true,
	}

	method = strings.ToUpper(method)
	if !validMethods[method] {
		return fmt.Errorf("INVALID_METHOD: %s", method)
	}

	return nil
}

func (h *HTTPHandler) buildURL(urlPath, method string, args map[string]interface{}) string {
	// Replace path parameters
	finalURL := urlPath
	for k, v := range args {
		placeholder := "{" + k + "}"
		if strings.Contains(finalURL, placeholder) {
			finalURL = strings.ReplaceAll(finalURL, placeholder, url.PathEscape(fmt.Sprintf("%v", v)))
			delete(args, k)
		}
	}

	// Add query parameters for GET/DELETE
	if method == "GET" || method == "DELETE" || method == "HEAD" || method == "OPTIONS" {
		if queryString := h.buildQueryString(args); queryString != "" {
			finalURL += "?" + queryString
		}
	}

	return finalURL
}

func (h *HTTPHandler) buildQueryString(args map[string]interface{}) string {
	values := url.Values{}
	for k, v := range args {
		if k != "body" {
			values.Add(k, fmt.Sprintf("%v", v))
		}
	}
	return values.Encode()
}

func (h *HTTPHandler) createRequest(ctx context.Context, method, url string, args map[string]interface{}) (*http.Request, error) {
	var body io.Reader

	if method == "POST" || method == "PUT" || method == "PATCH" {
		body = h.createRequestBody(args)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (h *HTTPHandler) createRequestBody(args map[string]interface{}) io.Reader {
	if bodyValue, exists := args["body"]; exists {
		if bodyStr, ok := bodyValue.(string); ok && bodyStr != "" {
			return strings.NewReader(bodyStr)
		}
		
		jsonData, _ := json.Marshal(bodyValue)
		return strings.NewReader(string(jsonData))
	}

	if len(args) > 0 {
		jsonData, _ := json.Marshal(args)
		return strings.NewReader(string(jsonData))
	}

	return strings.NewReader("{}")
}

func (h *HTTPHandler) addAuthentication(req *http.Request) error {
	authHeaderSet := false

	if token := strings.TrimSpace(h.auth.BearerToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		authHeaderSet = true
	}

	if apiKey := strings.TrimSpace(h.auth.APIKey); apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
		authHeaderSet = true
	}

	if !authHeaderSet {
		return fmt.Errorf("AUTH_REQUIRED: No BEARER_TOKEN or API_KEY found")
	}

	return nil
}

func (h *HTTPHandler) executeRequest(req *http.Request) (*mcp.CallToolResult, error) {
	resp, err := h.client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("REQUEST_FAILED: %v", err)), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("RESPONSE_READ_FAILED: %v", err)), nil
	}

	if resp.StatusCode >= 400 {
		return mcp.NewToolResultError(fmt.Sprintf("HTTP_ERROR %d: %s", resp.StatusCode, string(body))), nil
	}

	return mcp.NewToolResultText(string(body)), nil
}
