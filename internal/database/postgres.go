// internal/database/postgres.go
package database

import (
	"database/sql"
	"encoding/json"
	"log"
	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

func NewDB(connString string) (*DB, error) {
	conn, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}
	log.Println("Connected to database")
	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// Developer operations
func (db *DB) CreateDeveloper(email, passwordHash, apiKeyHash, apiKeyPrefix string) (string, error) {
	var id string
	err := db.conn.QueryRow(`
        INSERT INTO developers (email, password_hash, api_key_hash, api_key_prefix)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `, email, passwordHash, apiKeyHash, apiKeyPrefix).Scan(&id)
	if err != nil {
		log.Printf("CreateDeveloper failed for email %s: %v", email, err)
	} else {
		log.Printf("CreateDeveloper succeeded: id=%s, email=%s", id, email)
	}
	return id, err
}

func (db *DB) GetDeveloperByAPIKey(apiKeyHash string) (*Developer, error) {
	var dev Developer
	err := db.conn.QueryRow(`
        SELECT id, email, company_name, plan, created_at
        FROM developers
        WHERE api_key_hash = $1
    `, apiKeyHash).Scan(&dev.ID, &dev.Email, &dev.CompanyName, &dev.Plan, &dev.CreatedAt)
	return &dev, err
}

// Spec operations
func (db *DB) SaveOpenAPISpec(developerID, specName string, specContent []byte) (string, error) {
	var specID string
	err := db.conn.QueryRow(`
        INSERT INTO openapi_specs (developer_id, spec_name, spec_content)
        VALUES ($1, $2, $3)
        RETURNING id
    `, developerID, specName, specContent).Scan(&specID)
	return specID, err
}

// List specs for developer
func (db *DB) GetDeveloperSpecs(developerID string) ([]OpenAPISpec, error) {
	rows, err := db.conn.Query(`
        SELECT id, developer_id, spec_name, spec_content, created_at, updated_at
        FROM openapi_specs
        WHERE developer_id = $1
        ORDER BY created_at DESC
    `, developerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var specs []OpenAPISpec
	for rows.Next() {
		var s OpenAPISpec
		if scanErr := rows.Scan(&s.ID, &s.DeveloperID, &s.SpecName, &s.SpecContent, &s.CreatedAt, &s.UpdatedAt); scanErr != nil {
			continue
		}
		specs = append(specs, s)
	}
	return specs, nil
}

// Tool operations
func (db *DB) SaveTool(developerID, specID, toolName, description, method, urlPath string, inputSchema map[string]interface{}, requiresAuth bool) error {
	schemaJSON, _ := json.Marshal(inputSchema)
	_, err := db.conn.Exec(`
        INSERT INTO tools (developer_id, spec_id, tool_name, description, method, url_path, input_schema, requires_auth)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (developer_id, tool_name) DO UPDATE
        SET description = $4, method = $5, url_path = $6, input_schema = $7, requires_auth = $8
    `, developerID, specID, toolName, description, method, urlPath, schemaJSON, requiresAuth)
	return err
}

func (db *DB) GetDeveloperTools(developerID string) ([]Tool, error) {
	rows, err := db.conn.Query(`
        SELECT tool_name, description, method, url_path, input_schema, requires_auth
        FROM tools
        WHERE developer_id = $1
    `, developerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tools []Tool
	for rows.Next() {
		var tool Tool
		var schemaJSON []byte
		err := rows.Scan(&tool.Name, &tool.Description, &tool.Method, &tool.URLPath, &schemaJSON, &tool.RequiresAuth)
		if err != nil {
			continue
		}
		json.Unmarshal(schemaJSON, &tool.InputSchema)
		tools = append(tools, tool)
	}
	return tools, nil
}

// Find tool by name
func (db *DB) GetTool(developerID, toolName string) (*Tool, error) {
	var tools, err = db.GetDeveloperTools(developerID)
	for _, tool := range tools {
		if tool.Name == toolName {
			return &tool, nil
		}
	}
	return nil, err
}

// Logging
func (db *DB) LogExecution(developerID, toolName string, executionTime int64, success bool) error {
	status := "success"
	if !success {
		status = "error"
	}
	_, err := db.conn.Exec(`
        INSERT INTO execution_logs (developer_id, tool_name, execution_time_ms, status)
        VALUES ($1, $2, $3, $4)
    `, developerID, toolName, executionTime, status)

	// Update monthly usage
	db.conn.Exec(`
        INSERT INTO monthly_usage (developer_id, year_month, api_calls)
        VALUES ($1, TO_CHAR(NOW(), 'YYYY-MM'), 1)
        ON CONFLICT (developer_id, year_month)
        DO UPDATE SET api_calls = monthly_usage.api_calls + 1
    `, developerID)

	return err
}

// GetAllDevelopers returns all developers in the database
func (db *DB) GetAllDevelopers() ([]*Developer, error) {
	rows, err := db.conn.Query(`
		SELECT id, email, password_hash, api_key_hash, api_key_prefix, created_at
		FROM developers
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var developers []*Developer
	for rows.Next() {
		var dev Developer
		err := rows.Scan(&dev.ID, &dev.Email, &dev.APIKeyHash, &dev.CreatedAt)
		if err != nil {
			continue
		}
		developers = append(developers, &dev)
	}
	return developers, nil
}

