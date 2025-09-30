# MCP Server for OpenAPI Integration

This project allows developers to expose their website's API to AI agents (LLMs) using the Model Context Protocol (MCP). It dynamically generates MCP tools from an OpenAPI (Swagger) specification and serves them via both HTTP and stdio.

## Features
- Upload your OpenAPI spec to generate MCP tools automatically
- Supports both HTTP and stdio for MCP protocol
- AI agents can interact with your API endpoints as tools
- Authentication via Bearer token or API key (optional)

## Getting Started

### 1. Build and Run
```sh
# Build the server
cd open_api_to_mcp_server
# If you have Go installed:
go build -o open_api_to_mcp_server.exe main.go

# Run the server
./open_api_to_mcp_server.exe
```

The server will start:
- HTTP server on `http://localhost:8080`
- MCP stdio server (for integration with compatible clients)

### 2. Upload Your OpenAPI Spec
Send a POST request to `/upload` with your OpenAPI JSON file:
```sh
curl -X POST http://localhost:8080/upload -F "spec=@path/to/openapi.json"
```

### 3. Call API Endpoints via MCP
Send a POST request to `/mcp` with the tool name (operationId) and arguments:
```sh
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "params": {
      "name": "show_all_divisions__get",
      "arguments": { "depth": 2 }
    }
  }'
```

- Replace `show_all_divisions__get` with the desired operationId from your OpenAPI spec.
- Set arguments as needed for each endpoint.

### 4. Integrate with AI Agents
- Point your AI agent or LLM client to the MCP server (HTTP or stdio).
- Use the tool names and argument schema generated from your OpenAPI spec.
- Parse the MCP response to get API results.

## Authentication
Set environment variables for authentication if your API requires it:
- `BEARER_TOKEN` for Bearer token
- `API_KEY` for API key

## Example OpenAPI Spec
See `openapi.json` for a sample spec for Vietnam Provinces API.

## License
MIT

---
For more details on MCP, see [Model Context Protocol documentation](https://modelcontextprotocol.io/).
