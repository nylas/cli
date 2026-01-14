# Debugging Guide

Tips and techniques for debugging the Nylas CLI.

---

## Quick Debugging

### Enable Verbose Output

```bash
# Run command with verbose flag (if supported)
nylas --debug email list

# Check logs
tail -f ~/.config/nylas/nylas.log
```

### Test API Directly

```bash
# Test authentication
curl -H "Authorization: Bearer YOUR_API_KEY" \
     https://api.nylas.com/v3/grants/YOUR_GRANT_ID

# Test specific endpoint
curl -H "Authorization: Bearer YOUR_API_KEY" \
     https://api.nylas.com/v3/grants/YOUR_GRANT_ID/messages?limit=1
```

---

## Common Issues

### "Command not found"

```bash
# Check installation
which nylas

# Verify PATH
echo $PATH

# Reinstall if needed
go install github.com/nylas/cli/cmd/nylas@latest
```

### "401 Unauthorized"

```bash
# Check config
cat ~/.config/nylas/config.yaml

# Verify API key
nylas auth status

# Reconfigure
nylas auth config
```

### "No such file or directory"

```bash
# Check file exists
ls -la /path/to/file

# Use absolute path
nylas email send --attach "/absolute/path/to/file.pdf"
```

---

## Development Debugging

### Using Delve Debugger

```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug CLI
dlv debug github.com/nylas/cli/cmd/nylas -- email list

# Set breakpoint
(dlv) break main.main
(dlv) continue
(dlv) print variableName
```

### Print Debugging

```go
import "fmt"

func myFunction() {
    // Add debug prints
    fmt.Printf("DEBUG: value = %+v\n", value)
    fmt.Printf("DEBUG: entering function with args: %v\n", args)
}
```

### Logging

```go
import "log"

log.Printf("Processing email: %s", emailID)
log.Printf("API response: %+v", response)
```

---

## Testing Debugging

### Run Single Test

```bash
# Run specific test
go test ./internal/cli/email/ -run TestSendEmail -v

# With race detector
go test -race ./internal/cli/email/ -run TestSendEmail
```

### Debug Test

```bash
# Debug specific test
dlv test ./internal/cli/email/ -- -test.run TestSendEmail
```

### Print Test Output

```go
func TestMyFunction(t *testing.T) {
    // Use t.Log instead of fmt.Print
    t.Logf("DEBUG: testing with value: %v", value)
    
    // Force test failure to see logs
    t.Errorf("Debug output: %+v", result)
}
```

---

## Network Debugging

### Capture HTTP Traffic

```bash
# Use mitmproxy
pip install mitmproxy
mitmproxy -p 8080

# Configure proxy
export HTTP_PROXY=http://localhost:8080
export HTTPS_PROXY=http://localhost:8080

# Run CLI
nylas email list
```

### Check DNS

```bash
# Verify DNS resolution
nslookup api.nylas.com
dig api.nylas.com

# Test connectivity
ping api.nylas.com
curl -I https://api.nylas.com
```

---

## Build Issues

### Clean Build

```bash
# Remove artifacts
make clean

# Clear Go cache
go clean -cache -testcache -modcache

# Rebuild
make build
```

### Dependency Issues

```bash
# Update dependencies
go mod tidy

# Verify go.mod
go mod verify

# Download dependencies
go mod download
```

---

## IDE Debugging

### VS Code

**.vscode/launch.json:**
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug CLI",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/nylas",
            "args": ["email", "list"],
            "env": {
                "NYLAS_API_KEY": "your-api-key",
                "NYLAS_GRANT_ID": "your-grant-id"
            }
        }
    ]
}
```

### GoLand

1. Run → Edit Configurations
2. Add New Configuration → Go Build
3. Set Program arguments: `email list`
4. Set Environment variables: `NYLAS_API_KEY=...`
5. Run → Debug

---

## Performance Debugging

### Profile CPU

```bash
# Run with CPU profiling
go test -cpuprofile=cpu.prof -bench=.

# Analyze profile
go tool pprof cpu.prof
```

### Profile Memory

```bash
# Run with memory profiling
go test -memprofile=mem.prof -bench=.

# Analyze profile
go tool pprof mem.prof
```

### Benchmark

```go
func BenchmarkFunction(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Function()
    }
}
```

---

## Useful Commands

```bash
# Check Go environment
go env

# List dependencies
go list -m all

# Show module info
go list -m -json github.com/nylas/cli

# Find where CLI is installed
which nylas

# Check version
nylas version
go version
```

---

## More Resources

- **Delve Docs:** https://github.com/go-delve/delve
- **Go Debugging:** https://go.dev/doc/diagnostics
- **pprof:** https://pkg.go.dev/runtime/pprof
