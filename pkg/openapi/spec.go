package openapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Spec represents an OpenAPI specification
type Spec struct {
	OpenAPI    string              `json:"openapi"`
	Info       Info                `json:"info"`
	Servers    []Server            `json:"servers"`
	Paths      map[string]PathItem `json:"paths"`
	Components *Components         `json:"components,omitempty"`
}

type Info struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
}

type Operation struct {
	OperationID string                 `json:"operationId,omitempty"`
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	Parameters  []Parameter            `json:"parameters,omitempty"`
	RequestBody *RequestBody           `json:"requestBody,omitempty"`
	Responses   map[string]interface{} `json:"responses,omitempty"`
	Security    []map[string][]string  `json:"security,omitempty"`
}

type Parameter struct {
	Name        string                 `json:"name"`
	In          string                 `json:"in"`
	Required    bool                   `json:"required,omitempty"`
	Description string                 `json:"description,omitempty"`
	Schema      map[string]interface{} `json:"schema,omitempty"`
}

type RequestBody struct {
	Required bool                              `json:"required,omitempty"`
	Content  map[string]map[string]interface{} `json:"content,omitempty"`
}

type Components struct {
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

type SecurityScheme struct {
	Type   string `json:"type"`
	Scheme string `json:"scheme,omitempty"`
	Name   string `json:"name,omitempty"`
	In     string `json:"in,omitempty"`
}

// Loader handles loading OpenAPI specifications
type Loader struct {
	httpClient *http.Client
}

// NewLoader creates a new OpenAPI spec loader
func NewLoader() *Loader {
	return &Loader{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Load loads an OpenAPI spec from URL or file path
func (l *Loader) Load(specURLOrPath string) (*Spec, error) {
	data, err := l.fetchData(specURLOrPath)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spec: %w", err)
	}

	var spec Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	return &spec, nil
}

func (l *Loader) fetchData(source string) ([]byte, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return l.fetchFromURL(source)
	}
	return l.fetchFromFile(source)
}

func (l *Loader) fetchFromURL(url string) ([]byte, error) {
	resp, err := l.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (l *Loader) fetchFromFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// GetBaseURL returns the base URL from the spec's servers
func (s *Spec) GetBaseURL() string {
	if len(s.Servers) > 0 {
		return strings.TrimSuffix(s.Servers[0].URL, "/")
	}
	return ""
}

// HasAuthentication checks if the spec requires authentication
func (s *Spec) HasAuthentication() bool {
	return s.Components != nil && len(s.Components.SecuritySchemes) > 0
}
