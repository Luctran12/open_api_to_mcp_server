// pkg/utils/response.go
package utils

import (
    "encoding/json"
    "net/http"
)

type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"`
}

type Meta struct {
    RequestID       string `json:"request_id,omitempty"`
    ExecutionTimeMs int64  `json:"execution_time_ms,omitempty"`
    Timestamp       string `json:"timestamp"`
}

func SendJSON(w http.ResponseWriter, status int, response Response) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(response)
}

func SendSuccess(w http.ResponseWriter, data interface{}) {
    SendJSON(w, http.StatusOK, Response{
        Success: true,
        Data:    data,
    })
}

func SendError(w http.ResponseWriter, status int, message string) {
    SendJSON(w, status, Response{
        Success: false,
        Error:   message,
    })
}