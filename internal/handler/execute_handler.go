// internal/handlers/execute_handlers.go
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"open_api_to_mcp_server/internal/builder"
	"open_api_to_mcp_server/internal/database"
	"open_api_to_mcp_server/pkg/openapi"
	"open_api_to_mcp_server/pkg/utils"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ExecuteHandler struct {
	db *database.DB
}

type BuildRequest struct {
	SpecID string `json:"spec_id"`
}

// urlValues là type alias để làm việc với URL parameters
type urlValues url.Values

// Add method cho urlValues
func (uv urlValues) Add(key, value string) {
	url.Values(uv).Add(key, value)
}

// Encode method cho urlValues
func (uv urlValues) Encode() string {
	return url.Values(uv).Encode()
}

func NewExecuteHandler(db *database.DB) *ExecuteHandler {
	return &ExecuteHandler{db: db}
}

// POST /api/execute
func (h *ExecuteHandler) Execute(w http.ResponseWriter, r *http.Request) {
	developer := r.Context().Value("developer").(*database.Developer)

	defer r.Body.Close()
	
	// Extract end-user token (pass-through)
	endUserToken := r.Header.Get("Authorization")
	if endUserToken == "" {
		utils.SendError(w, 400, "Authorization header required for end-user")
		return
	}

	// Parse request
	var req struct {
		ToolName  string                 `json:"tool_name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, 400, "Invalid request body")
		return
	}

	// Load tool from database
	tool, err := h.db.GetTool(developer.ID, req.ToolName)
	if err != nil {
		utils.SendError(w, 404, "Tool not found")
		return
	}

	// Create handler from tool metadata
	handler := MakeHandler(tool.Method, tool.URLPath, tool.RequiresAuth)

	// Create context with auth token
	ctx := context.WithValue(r.Context(), "auth_token", endUserToken)

	// Execute tool
	// NOTE: Remove unsupported fields from CallToolRequest construction and
	//       build only with known fields (Params with Name and Arguments).
	startTime := time.Now()
	callReq := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      req.ToolName,
			Arguments: req.Arguments,
		},
	}
	result, err := handler(ctx, callReq)
	executionTime := time.Since(startTime).Milliseconds()

	// Log execution
	go h.db.LogExecution(developer.ID, req.ToolName, executionTime, err == nil)

	// Handle errors
	if err != nil {
		utils.SendError(w, 500, err.Error())
		return
	}

	// Success response
	// NOTE: Return the tool result object directly. The handler can return
	//       either a text result or structured content; avoid unsafe string
	//       assumptions and let the client handle the shape.
	utils.SendJSON(w, 200, utils.Response{
		Success: true,
		Data:    result,
		Meta: &utils.Meta{
			ExecutionTimeMs: executionTime,
			Timestamp:       time.Now().Format(time.RFC3339),
		},
	})
}

func (h *ExecuteHandler) Build(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	//developer := r.Context().Value("developer").(*database.Developer)
	var specId BuildRequest
	if err := json.NewDecoder(r.Body).Decode(&specId); err != nil {
		utils.SendError(w, 400, "Invalid request body" + err.Error())
		return
	}
    // get spec from database by specId
	spec, err := h.db.GetOpenAPISpecByID(specId.SpecID)
	if err != nil {
		utils.SendError(w, 404, "Spec not found")
		return
	}
	// parse spec.content to OpenAPI spec by json.Unmarshal
	var openapiSpec openapi.Spec
	if err := json.Unmarshal(spec.SpecContent, &openapiSpec); err != nil {
		utils.SendError(w, 400, "Failed to parse OpenAPI spec: " + err.Error())
		return
	}
	// call BuildExecutable with openAPI 
	executablePath, err := builder.BuildExecutable(openapiSpec)
	
	if err != nil {
		utils.SendError(w, 500, err.Error())
		return
	}

    downloadURL := fmt.Sprintf("http://%s/api/build/download/%s", r.Host, url.PathEscape(filepath.Base(executablePath)))

	
	utils.SendJSON(w, 200, utils.Response{
		Success: true,
		Data: map[string]interface{}{
			"executable_path": downloadURL,
		},
	})
	
}

func (h *ExecuteHandler) Download(w http.ResponseWriter, r *http.Request) {
	fileName := filepath.Base(r.URL.Path)
	if fileName == "" {
		utils.SendError(w, 400, "File name not provided")
		return
	}

	// An toàn: chỉ cho phép tên file đơn giản
	safeFileName := filepath.Base(fileName) // chỉ lấy phần tên file

	// Đường dẫn đầy đủ tới file trong thư mục tạm của hệ điều hành
	filePath := filepath.Join(os.TempDir(), safeFileName)

	// Kiểm tra xem file có tồn tại không
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Thử tìm trong các thư mục con của TempDir (vì MkdirTemp tạo thư mục ngẫu nhiên)
		found := false
		walkErr := filepath.Walk(os.TempDir(), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && info.Name() == safeFileName {
				filePath = path
				found = true
				return filepath.SkipDir // Dừng tìm kiếm khi đã thấy
			}
			return nil
		})

		if walkErr != nil || !found {
			utils.SendError(w, 404, fmt.Sprintf("File not found: %s", safeFileName))
			return
		}
	}

	// Set header để trình duyệt tải file về
	w.Header().Set("Content-Disposition", "attachment; filename=\""+safeFileName+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")

	// Serve file
	http.ServeFile(w, r, filePath)
}

func MakeHandler(method, urlPath string, requiresAuth bool) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Validate input method
		validMethods := map[string]bool{
			"GET": true, "POST": true, "PUT": true, "DELETE": true,
			"PATCH": true, "HEAD": true, "OPTIONS": true,
		}

		method = strings.ToUpper(method)
		if !validMethods[method] {
			return mcp.NewToolResultError(fmt.Sprintf("INVALID_METHOD HTTP method không hợp lệ: %s", method)), nil
		}

		// Lấy arguments từ request
		// Theo MCP protocol, arguments nằm trực tiếp trong request
		var args map[string]interface{}
		if request.Params.Arguments != nil {
			args = request.GetArguments()
		} else {
			args = make(map[string]interface{})
		}

		// Thay thế path parameters {id} → giá trị thực
		finalURL := urlPath
		for k, v := range args {
			placeholder := "{" + k + "}"
			if strings.Contains(finalURL, placeholder) {
				// URL encode giá trị để tránh lỗi với ký tự đặc biệt
				finalURL = strings.ReplaceAll(finalURL, placeholder, url.PathEscape(fmt.Sprintf("%v", v)))
				delete(args, k)
			}
		}

		var httpReq *http.Request
		var err error

		switch method {
		case "GET", "DELETE", "HEAD", "OPTIONS":
			// Xử lý query parameters
			q := make(urlValues)
			for k, v := range args {
				if k != "body" { // Bỏ qua body parameter cho GET/DELETE
					q.Add(k, fmt.Sprintf("%v", v))
				}
			}
			if len(q) > 0 {
				finalURL += "?" + q.Encode()
			}
			httpReq, err = http.NewRequestWithContext(ctx, method, finalURL, nil)

		case "POST", "PUT", "PATCH":
			var body io.Reader

			// Kiểm tra xem có body parameter không
			if bodyValue, exists := args["body"]; exists {
				if bodyStr, ok := bodyValue.(string); ok && bodyStr != "" {
					body = strings.NewReader(bodyStr)
				} else {
					// Nếu body không phải string, convert thành JSON
					jsonData, jsonErr := json.Marshal(bodyValue)
					if jsonErr != nil {
						return mcp.NewToolResultError(fmt.Sprintf("INVALID_BODY Không thể serialize body: %v", jsonErr)), nil
					}
					body = strings.NewReader(string(jsonData))
				}
			} else {
				// Nếu không có body parameter, tạo JSON từ các args còn lại
				if len(args) > 0 {
					jsonData, jsonErr := json.Marshal(args)
					if jsonErr != nil {
						return mcp.NewToolResultError(fmt.Sprintf("INVALID_ARGS Không thể serialize arguments: %v", jsonErr)), nil
					}
					body = strings.NewReader(string(jsonData))
				} else {
					body = strings.NewReader("{}")
				}
			}

			httpReq, err = http.NewRequestWithContext(ctx, method, finalURL, body)
			if err == nil {
				httpReq.Header.Set("Content-Type", "application/json")
			}

		default:
			return mcp.NewToolResultError(fmt.Sprintf("UNSUPPORTED_METHOD HTTP method không được hỗ trợ: %s", method)), nil
		}

		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("REQUEST_CREATE_FAILED Lỗi tạo HTTP request: %v", err)), nil
		}

		// Xử lý authentication
		if requiresAuth {
			authHeaderSet := false

			// Ưu tiên token được truyền từ end-user qua context
			// NOTE: `Execute` đặt token Authorization của end-user trong context với key "auth_token"
			if ctxToken, ok := ctx.Value("auth_token").(string); ok {
				trimmed := strings.TrimSpace(ctxToken)
				if trimmed != "" {
					// Giá trị đã là header hoàn chỉnh (ví dụ: "Bearer x.y.z"); giữ nguyên
					httpReq.Header.Set("Authorization", trimmed)
					authHeaderSet = true
				}
			}

			// Fallback: Bearer token từ biến môi trường
			if !authHeaderSet {
				if token := strings.TrimSpace(os.Getenv("BEARER_TOKEN")); token != "" {
					httpReq.Header.Set("Authorization", "Bearer "+token)
					authHeaderSet = true
				}
			}

			// Fallback bổ sung: API Key header
			if apiKey := strings.TrimSpace(os.Getenv("API_KEY")); apiKey != "" {
				httpReq.Header.Set("X-API-Key", apiKey)
				authHeaderSet = true
			}

			// Kiểm tra xem có credential nào được cung cấp không
			if !authHeaderSet {
				return mcp.NewToolResultError("AUTH_REQUIRED Yêu cầu authentication nhưng không tìm thấy Authorization, BEARER_TOKEN hoặc API_KEY"), nil
			}
		}

		// Thực hiện HTTP request với timeout
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		resp, err := client.Do(httpReq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("REQUEST_FAILED Lỗi khi gọi API: %v", err)), nil
		}
		defer resp.Body.Close()

		// Đọc response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("RESPONSE_READ_FAILED Lỗi đọc response: %v", err)), nil
		}

		// Kiểm tra HTTP status code
		if resp.StatusCode >= 400 {
			return mcp.NewToolResultError(fmt.Sprintf("HTTP_ERROR API trả về lỗi %d: %s", resp.StatusCode, string(body))), nil
		}

		// Trả về kết quả thành công
		return mcp.NewToolResultText(string(body)), nil
	}
}
