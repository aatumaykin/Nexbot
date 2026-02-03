# Testing Rules

## Strategy

Three-level testing strategy:
1. Unit tests — test individual functions and methods
2. Integration tests — test component interactions
3. E2E tests — test complete workflow

## Unit Tests

### Structure

```
internal/
├── config/
│   ├── config.go
│   └── config_test.go       # Unit tests for config
├── agent/
│   ├── loop/
│   │   ├── loop.go
│   │   └── loop_test.go     # Unit tests for loop
│   └── context/
│       ├── context.go
│       └── context_test.go  # Unit tests for context
```

### Rules

✅ Use table-driven tests
✅ Mock external dependencies
✅ Test positive and negative scenarios
✅ Verify error handling
✅ Test edge cases

### Table-driven Test Pattern

```go
func TestSafePath(t *testing.T) {
    tests := []struct {
        name     string
        baseDir  string
        filePath string
        want     string
        wantErr  bool
    }{
        {
            name:     "valid path",
            baseDir:  "/tmp",
            filePath: "file.txt",
            want:     "/tmp/file.txt",
            wantErr:  false,
        },
        {
            name:     "path traversal attempt",
            baseDir:  "/tmp",
            filePath: "../etc/passwd",
            want:     "",
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := safePath(tt.baseDir, tt.filePath)
            if (err != nil) != tt.wantErr {
                t.Errorf("safePath() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("safePath() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Integration Tests

### Structure

```
tests/
├── integration/
│   ├── agent_integration_test.go    # Integration tests for agent
│   ├── bus_integration_test.go      # Integration tests for message bus
│   └── tools_integration_test.go    # Integration tests for tools
```

### Rules

✅ Test component interactions
✅ Use real implementations with mock external APIs
✅ Clean state after each test
✅ Use temporary directory for workspace

### Integration Test Pattern

```go
func TestAgentLoopIntegration(t *testing.T) {
    tmpDir := t.TempDir()
    workspaceDir := filepath.Join(tmpDir, "workspace")
    sessionDir := filepath.Join(tmpDir, "sessions")

    os.MkdirAll(workspaceDir, 0755)
    os.WriteFile(filepath.Join(workspaceDir, "IDENTITY.md"), []byte("Test identity"), 0644)

    mockLLM := &MockLLMProvider{
        MockChat: func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
            return &llm.ChatResponse{Content: "Integration test response"}, nil
        },
    }

    loop, err := loop.NewLoop(loop.Config{
        Workspace:   workspaceDir,
        SessionDir:  sessionDir,
        LLMProvider: mockLLM,
        Logger:      logger.New(),
    })
    if err != nil {
        t.Fatal(err)
    }

    response, err := loop.Process(context.Background(), "session-123", "Hello")
    if err != nil {
        t.Fatalf("Process() error = %v", err)
    }
}
```

## E2E Tests

### Structure

```
tests/
└── e2e_test.go    # End-to-end tests
```

### Rules

✅ Test complete workflow
✅ Use real configurations
✅ Test through public API
✅ Clean state after tests

### E2E Test Pattern

```go
func TestE2EFullWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }

    config, err := config.Load("config.toml")
    if err != nil {
        t.Fatal(err)
    }

    loop, err := loop.NewLoop(loop.Config{
        Workspace:   config.Agent.Workspace,
        SessionDir:  config.Agent.SessionDir,
        LLMProvider: &llm.ZaiProvider{},
        Logger:      logger.New(),
    })
    if err != nil {
        t.Fatal(err)
    }

    response, err := loop.Process(context.Background(), "test-session", "Read file test.txt")
    if err != nil {
        t.Fatalf("Process() error = %v", err)
    }

    if !strings.Contains(response, "File content") {
        t.Errorf("Response does not contain expected content")
    }
}
```

## Coverage Targets

- Minimum: 70%
- Target: 80%
- Optimal: 90%

## Coverage Exclusions

- Generated code
- Mock implementations
- Test code

## Best Practices

✅ Write tests before code (TDD)
✅ Test edge cases and error handling
✅ Keep tests fast (<1s for unit tests)
✅ Isolate tests from each other
✅ Use temporary directory (t.TempDir())
✅ Mock external dependencies
✅ Use table-driven tests
✅ Descriptive test names

❌ Don't test standard library
❌ Don't test implementations (test interfaces)
❌ Don't use sleep in tests
❌ Don't depend on test execution order
❌ Don't test via public API in unit tests
