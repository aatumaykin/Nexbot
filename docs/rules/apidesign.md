# API Design Rules

## Current Architecture

### Telegram Connector

```
Telegram ──► Connector ──► Message Bus ──► Agent Loop ──► Outbound Queue ──► Telegram
```

### Message Types

```go
type Message struct {
    ID        string
    SessionID string
    Role      Role (User/Assistant/System/Tool)
    Content   string
    ToolCalls []ToolCall
    ToolCallID string
}
```

## LLM Provider Interface

```go
type Provider interface {
    // Get default model
    GetDefaultModel() string
    
    // Check tool calling support
    SupportsToolCalling() bool
    
    // Send request to LLM
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}

type ChatRequest struct {
    Messages    []Message
    Model       string
    Temperature float64
    MaxTokens   int
    Tools       []ToolDefinition
}

type ChatResponse struct {
    Content      string
    FinishReason FinishReason
    ToolCalls    []ToolCall
}
```

## Future REST API (not implemented)

### URL Structure (when implemented)

```
GET    /api/v1/sessions              # List sessions
POST   /api/v1/sessions              # Create session
GET    /api/v1/sessions/{id}         # Get session
DELETE /api/v1/sessions/{id}         # Delete session

POST   /api/v1/sessions/{id}/messages # Send message
GET    /api/v1/sessions/{id}/messages # Get history

GET    /api/v1/tools                  # List tools
POST   /api/v1/tools/{name}/execute   # Execute tool

GET    /api/v1/skills                  # List skills
GET    /api/v1/skills/{name}           # Get skill

GET    /api/v1/health                  # Health check
```

### HTTP Status Codes (when implemented)

- 200 OK
- 201 Created
- 400 Bad Request
- 404 Not Found
- 500 Internal Server Error

### Rate Limiting (when implemented)

- 100 requests/minute per session
- 1000 requests/minute per IP
- Use X-RateLimit-* headers

### Authentication (when implemented)

- API Key in header: `Authorization: Bearer <api_key>`
- Validate via middleware

### WebSocket (optional)

For real-time communication (when implemented):
- URL: wss://api.nexbot.io/v1/ws?session_id=abc123
- Events: message, tool_call, done

## Security Rules

See `docs/rules/security.md`:
- HTTPS only
- API keys via Authorization header
- Rate limiting
- Input validation
- Output sanitization
- Mask secrets in logs
