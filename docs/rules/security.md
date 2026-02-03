# Security Rules

These rules are MANDATORY for ALL code changes. Violating security rules is unacceptable.

## Critical Rules

### No Hardcoded Secrets

**PROHIBITED:**
- API keys in code
- JWT tokens in code
- Passwords in code
- Private keys in code
- Connection strings in code
- Database URLs in code
- Encryption keys in code

**REQUIRED:**
- Use environment variables: `os.Getenv("ZAI_API_KEY")`
- Use config with env substitution: `api_key = "${ZAI_API_KEY:}"` in TOML
- Validate environment variables are set
- Never log secrets

### Environment Variables

**Read from environment:**
```go
apiKey := os.Getenv("ZAI_API_KEY")
if apiKey == "" {
    return fmt.Errorf("ZAI_API_KEY not set")
}
```

**Config substitution (TOML):**
```toml
[llm.zai]
api_key = "${ZAI_API_KEY:}"
```

### Masking Secrets in Logs

**REQUIRED:**
- Always mask secrets before logging
- Use `config.MaskSensitiveFields()` for config
- Manually mask individual values
- Never log API keys, tokens, passwords, connection strings

### File Operations

**PROHIBITED:**
- Read/write outside allowed directories
- Path traversal attacks (../)

**REQUIRED:**
- Validate paths using `filepath.Clean()`
- Check path is inside base directory using `filepath.Rel()`
- Reject paths starting with `..`
- Use whitelist of allowed directories

### Shell Commands

**PROHIBITED:**
- Commands not in whitelist
- Arbitrary user input in commands
- Command injection vulnerabilities

**REQUIRED:**
- Use whitelist of allowed commands
- Validate all arguments
- Reject arguments containing `..`
- Log all command executions
- Use `exec.CommandContext()` with timeout

### External API Calls

**REQUIRED:**
- Always use HTTPS
- Validate all input data
- Set timeouts on requests
- Handle errors properly
- Never log sensitive data (headers, bodies)
- Use context with cancellation

### Secret Detection Patterns

Search code for these patterns and verify they contain NO real secrets:
- `password.*=`
- `token.*=`
- `apikey.*=`
- `api_key.*=`
- `secret.*=`
- `privatekey.*=`
- `private_key.*=`
- `connectionstring.*=`
- `connection_string.*=`
- `databaseurl.*=`
- `database_url.*=`
- `jwt.*=`
- `bearer.*=`

## Error Handling

**When secret is found in code:**
1. Remove secret immediately
2. Replace with environment variable
3. Revoke and regenerate secret if committed/pushed
4. Update environment variables

## Commit Rules

**NEVER commit:**
- API keys
- Tokens
- Passwords
- Private keys
- Secrets
- .env files with real values
