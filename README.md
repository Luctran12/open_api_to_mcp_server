## Open API → MCP Server (Go)

A Go service that ingests an OpenAPI spec, generates MCP tools, persists them in Postgres, and exposes HTTP endpoints to manage specs, tools, and execute tool calls.

### Features
- Upload OpenAPI specs and auto-generate tools
- Persist tools and usage logs in Postgres
- Execute generated tools via an HTTP endpoint
- Authentication and pass-through Authorization when required by tools
- MCP integration via `github.com/mark3labs/mcp-go`

### Stack
- Go 1.24+
- Postgres 13+
- Modules: `github.com/lib/pq`, `github.com/mark3labs/mcp-go`, `github.com/joho/godotenv`

---

### Setup

1) Create database schema

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS developers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  company_name TEXT,
  plan TEXT,
  password_hash TEXT NOT NULL,
  api_key_hash TEXT NOT NULL UNIQUE,
  api_key_prefix TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS openapi_specs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  developer_id UUID NOT NULL REFERENCES developers(id) ON DELETE CASCADE,
  spec_name TEXT NOT NULL,
  spec_content JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tools (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  developer_id UUID NOT NULL REFERENCES developers(id) ON DELETE CASCADE,
  spec_id UUID NOT NULL REFERENCES openapi_specs(id) ON DELETE CASCADE,
  tool_name TEXT NOT NULL,
  description TEXT,
  method TEXT NOT NULL,
  url_path TEXT NOT NULL,
  input_schema JSONB NOT NULL,
  requires_auth BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_tools_developer_toolname
  ON tools (developer_id, tool_name);

CREATE TABLE IF NOT EXISTS execution_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  developer_id UUID NOT NULL REFERENCES developers(id) ON DELETE CASCADE,
  tool_name TEXT NOT NULL,
  execution_time_ms BIGINT NOT NULL,
  status TEXT NOT NULL,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_execution_logs_dev_created
  ON execution_logs (developer_id, created_at DESC);

CREATE TABLE IF NOT EXISTS monthly_usage (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  developer_id UUID NOT NULL REFERENCES developers(id) ON DELETE CASCADE,
  year_month TEXT NOT NULL,
  api_calls INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_monthly_usage_dev_month
  ON monthly_usage (developer_id, year_month);
```

2) Configure environment

Create `.env` in the project root:

```
DATABASE_URL=postgresql://postgres:password@localhost:5432/postgres?sslmode=disable
PORT=8080
```

`main.go` loads `.env` with `godotenv.Load()`; in production, set real env vars.

3) Run

```bash
go run .
```

Windows PowerShell one-off port override:

```powershell
$env:PORT="8081"; go run .
```

---

### API

- Public
  - POST `/api/auth/register`
  - POST `/api/auth/login`

- Protected (requires developer auth)
  - POST `/api/specs` — upload OpenAPI JSON
  - GET `/api/specs` — list specs
  - GET `/api/tools` — list tools
  - POST `/api/execute` — execute a tool
    - Body example:
      ```json
      {
        "tool_name": "get_users",
        "arguments": { "id": 123, "body": "{\"foo\":\"bar\"}" }
      }
      ```
    - If a tool requires auth, include end-user `Authorization` header; the service forwards it.

---

### Troubleshooting

- Port already in use: set `PORT` or stop the process using 8080.
- `.env` not read: ensure `.env` is in the working dir or set env vars in shell/IDE.
- DB connection errors: verify `DATABASE_URL` and Postgres is running.

---

### Build

```bash
go build -o open_api_to_mcp_server.exe
```

---

### License
MIT
