package database

import "time"

// Developer represents a developer account
type Developer struct {
    ID          string    `json:"id"`
    Email       string    `json:"email"`
    CompanyName string    `json:"company_name"`
    Plan        string    `json:"plan"`
    APIKeyHash  string    `json:"-"` // Don't expose in JSON
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// Tool represents a generated tool from OpenAPI spec
type Tool struct {
    ID           string                 `json:"id"`
    DeveloperID  string                 `json:"developer_id"`
    SpecID       string                 `json:"spec_id"`
    Name         string                 `json:"name"`
    Description  string                 `json:"description"`
    Method       string                 `json:"method"`
    URLPath      string                 `json:"url_path"`
    InputSchema  map[string]interface{} `json:"input_schema"`
    RequiresAuth bool                   `json:"requires_auth"`
    CreatedAt    time.Time              `json:"created_at"`
}

// OpenAPISpec represents a stored OpenAPI specification
type OpenAPISpec struct {
    ID          string    `json:"id"`
    DeveloperID string    `json:"developer_id"`
    SpecName    string    `json:"spec_name"`
    SpecContent []byte    `json:"spec_content"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// ExecutionLog represents a tool execution record
type ExecutionLog struct {
    ID              string    `json:"id"`
    DeveloperID     string    `json:"developer_id"`
    ToolName        string    `json:"tool_name"`
    ExecutionTimeMs int64     `json:"execution_time_ms"`
    Status          string    `json:"status"`
    ErrorMessage    *string   `json:"error_message,omitempty"`
    CreatedAt       time.Time `json:"created_at"`
}

// MonthlyUsage represents monthly API usage statistics
type MonthlyUsage struct {
    ID          string `json:"id"`
    DeveloperID string `json:"developer_id"`
    YearMonth   string `json:"year_month"` // Format: "2025-10"
    APICalls    int    `json:"api_calls"`
}