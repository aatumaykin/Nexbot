# Code Quality Rules

## Principles

### SOLID
- **S**ingle Responsibility — one function does one thing
- **O**pen/Closed — open for extension, closed for modification
- **L**iskov Substitution — subtypes must be replaceable
- **I**nterface Segregation — interfaces must be minimal
- **D**ependency Inversion — depend on abstractions, not implementations

### DRY (Don't Repeat Yourself)
- Avoid code duplication
- Extract common logic into functions
- Use composition instead of inheritance

### KISS (Keep It Simple, Stupid)
- Write simple, understandable code
- Avoid premature optimization
- Simple solution is better than complex

### YAGNI (You Aren't Gonna Need It)
- Don't write code for future requirements
- Implement only current tasks
- Avoid over-engineering

## Naming Conventions

### Packages
- Lowercase, single word
- Examples: agent, config, tools

### Functions and Variables
- CamelCase for exported (MyFunction)
- camelCase for unexported (myFunction)
- Use verbs for functions (Get, Set, Create, Update, Delete)

### Types
- PascalCase for types (MyType, MyStruct)
- Interfaces with -er suffix if action (Reader, Writer)
- Without -er suffix if object (Config, Session)

### Constants
- UPPER_SNAKE_CASE for constants (MAX_RETRIES)

### Files
- lowercase with underscores (context_builder.go)
- Tests: <file>_test.go

## File Structure

```go
package package_name

import (
    // Standard library
    "fmt"
    "os"

    // External dependencies
    "github.com/aatumaykin/nexbot/internal/llm"
)

// Constants
const (
    MaxIterations = 10
)

// Variables
var (
    defaultModel = "glm-4.7-flash"
)

// Types
type MyStruct struct {
    Field1 string
    Field2 int
}

// Constructors
func NewMyStruct() *MyStruct {
    return &MyStruct{}
}

// Methods
func (m *MyStruct) Method() error {
    return nil
}

// Package functions
func PackageFunction() {
}
```

## Formatting

- Use gofmt
- Max line length: 120 characters
- Indentation: tabs
- Blank lines between functions and logical blocks

## Error Handling

- Always handle errors
- Never ignore errors with `_`
- Use fmt.Errorf for wrapping with context
- Return error as last value

```go
// Good
func MyFunction() error {
    data, err := os.ReadFile("file.txt")
    if err != nil {
        return fmt.Errorf("failed to read file: %w", err)
    }
    return nil
}

// Bad
func MyFunction() error {
    data, _ := os.ReadFile("file.txt")  // Ignoring error
    return nil
}
```

## Contexts

- First parameter: ctx context.Context
- Always pass ctx to called functions
- Use ctx for cancellation and timeouts

```go
func MyFunction(ctx context.Context) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // ...
    }
    return nil
}
```

## Logging

- Use structured logging
- Add contextual fields
- Mask secrets

```go
l.logger.DebugCtx(ctx, "Processing message",
    logger.Field{Key: "session_id", Value: sessionID},
    logger.Field{Key: "message_length", Value: len(msg)})
```

## Configuration

- Use TOML for configuration
- Secrets via environment variables
- Validate configuration on load

```toml
[agent]
model = "glm-4.7-flash"
max_tokens = 8192

[llm.zai]
api_key = "${ZAI_API_KEY:}"
```

## Testing

- Unit tests for each package
- Use table-driven tests
- Mock external dependencies
- Test names must describe scenario

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "success case",
            input:   "test",
            want:    "result",
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("MyFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("MyFunction() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Anti-patterns

❌ Magic numbers — use constants
❌ Deep nesting — use early returns
❌ Large functions — break into small ones
❌ Code duplication — extract to functions
❌ Ignored errors — handle all errors
❌ Naked panic/recover — use errors
❌ Global variables — avoid
❌ Functions with many parameters — use structs

## Refactoring

**When to refactor:**
- Code duplication
- Complex logic
- Large functions
- Bad names
- Principle violations

**How to refactor:**
1. Write tests
2. Refactor in small steps
3. Run tests after each step
4. Don't change API
