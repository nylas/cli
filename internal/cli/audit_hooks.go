package cli

import (
	"os"
	"os/user"
	"strings"
	"sync"
	"time"

	"github.com/nylas/cli/internal/adapters/audit"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// AuditContext holds audit state during command execution.
type AuditContext struct {
	StartTime  time.Time
	Command    string
	Args       []string
	GrantID    string
	GrantEmail string
	RequestID  string
	HTTPStatus int

	// Invoker tracking
	Invoker       string // Username: "alice", "dependabot[bot]"
	InvokerSource string // Source: "claude-code", "github-actions", "terminal"
}

var (
	auditMu      sync.Mutex
	currentAudit *AuditContext
)

// SetAuditRequestInfo sets API request information for the current audit entry.
// This should be called by the HTTP client after making API calls.
func SetAuditRequestInfo(requestID string, httpStatus int) {
	auditMu.Lock()
	defer auditMu.Unlock()
	if currentAudit != nil {
		currentAudit.RequestID = requestID
		currentAudit.HTTPStatus = httpStatus
	}
}

// SetAuditGrantInfo sets grant information for the current audit entry.
func SetAuditGrantInfo(grantID, grantEmail string) {
	auditMu.Lock()
	defer auditMu.Unlock()
	if currentAudit != nil {
		currentAudit.GrantID = grantID
		currentAudit.GrantEmail = grantEmail
	}
}

// initAuditHooks sets up the audit logging hooks on the root command.
func initAuditHooks(rootCmd *cobra.Command) {
	rootCmd.PersistentPreRunE = auditPreRun
	rootCmd.PersistentPostRunE = auditPostRun

	// Set up grant tracking hook
	common.AuditGrantHook = func(grantID string) {
		SetAuditGrantInfo(grantID, "")
	}

	// Set up request tracking hook
	ports.AuditRequestHook = SetAuditRequestInfo
}

// auditPreRun is called before every command execution.
func auditPreRun(cmd *cobra.Command, args []string) error {
	// Don't audit help, version, or completion commands
	if isExcludedCommand(cmd) {
		return nil
	}

	// Don't audit audit commands (avoid recursion)
	commandPath := getCommandPath(cmd)
	if strings.HasPrefix(commandPath, "audit") {
		return nil
	}

	// Detect invoker identity
	invoker, invokerSource := getInvokerIdentity()

	auditMu.Lock()
	currentAudit = &AuditContext{
		StartTime:     time.Now(),
		Command:       commandPath,
		Args:          sanitizeArgs(args),
		Invoker:       invoker,
		InvokerSource: invokerSource,
	}
	auditMu.Unlock()

	return nil
}

// auditPostRun is called after every command execution.
func auditPostRun(cmd *cobra.Command, args []string) error {
	auditMu.Lock()
	ctx := currentAudit
	currentAudit = nil
	auditMu.Unlock()

	if ctx == nil {
		return nil
	}

	logAuditEntry(ctx, domain.AuditStatusSuccess, "")
	return nil
}

// LogAuditError logs a command execution failure.
// Call this from error handlers to record failed commands.
func LogAuditError(err error) {
	auditMu.Lock()
	ctx := currentAudit
	auditMu.Unlock()

	if ctx == nil {
		return
	}

	logAuditEntry(ctx, domain.AuditStatusError, err.Error())
}

// logAuditEntry creates and logs an audit entry from the context.
func logAuditEntry(ctx *AuditContext, status domain.AuditStatus, errMsg string) {
	store, err := audit.NewFileStore("")
	if err != nil {
		return
	}

	cfg, err := store.GetConfig()
	if err != nil || !cfg.Enabled {
		return
	}

	entry := &domain.AuditEntry{
		Timestamp:     ctx.StartTime,
		Command:       ctx.Command,
		Args:          ctx.Args,
		GrantID:       ctx.GrantID,
		GrantEmail:    ctx.GrantEmail,
		Status:        status,
		Duration:      time.Since(ctx.StartTime),
		Error:         errMsg,
		Invoker:       ctx.Invoker,
		InvokerSource: ctx.InvokerSource,
	}

	if cfg.LogRequestID && ctx.RequestID != "" {
		entry.RequestID = ctx.RequestID
	}
	if cfg.LogAPIDetails {
		entry.HTTPStatus = ctx.HTTPStatus
	}

	_ = store.Log(entry)
}

// getCommandPath returns the full command path (e.g., "email list").
func getCommandPath(cmd *cobra.Command) string {
	path := cmd.Name()
	for p := cmd.Parent(); p != nil && p.Name() != "nylas"; p = p.Parent() {
		path = p.Name() + " " + path
	}
	return path
}

// isExcludedCommand returns true for commands that shouldn't be audited.
func isExcludedCommand(cmd *cobra.Command) bool {
	name := cmd.Name()
	return name == "help" ||
		name == "version" ||
		name == "completion" ||
		name == "__complete" ||
		name == "__completeNoDesc"
}

// sensitiveFlags contains flag names whose values should be redacted.
var sensitiveFlags = map[string]bool{
	"--api-key":       true,
	"--password":      true,
	"--token":         true,
	"--secret":        true,
	"--client-secret": true,
	"--access-token":  true,
	"--refresh-token": true,
	"--body":          true,
	"--subject":       true,
	"--html":          true,
	"-p":              true,
}

// sanitizeArgs removes sensitive information from arguments.
func sanitizeArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	result := make([]string, len(args))
	redactNext := false

	for i, arg := range args {
		if redactNext {
			result[i] = "[REDACTED]"
			redactNext = false
			continue
		}

		if sensitiveFlags[arg] {
			result[i] = arg
			redactNext = true
			continue
		}

		// Check --flag=value format
		if strings.HasPrefix(arg, "--") && strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			if sensitiveFlags[parts[0]] {
				result[i] = parts[0] + "=[REDACTED]"
				continue
			}
		}

		// Check for API key patterns
		if strings.HasPrefix(arg, "nyk_") || isLongBase64(arg) {
			result[i] = "[REDACTED]"
			continue
		}

		result[i] = arg
	}

	return result
}

// isLongBase64 checks if a string looks like a long base64 token.
func isLongBase64(s string) bool {
	if len(s) < 40 {
		return false
	}
	for _, c := range s {
		isAlpha := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
		isDigit := c >= '0' && c <= '9'
		isBase64Char := c == '+' || c == '/' || c == '=' || c == '-' || c == '_'
		if !isAlpha && !isDigit && !isBase64Char {
			return false
		}
	}
	return true
}

// getInvokerIdentity returns the username and source platform.
// Detection is based on environment variables set by each tool.
func getInvokerIdentity() (invoker, source string) {
	invoker = getUsername()

	// 1. AI Agents (check first - most specific)
	// Claude Code: CLAUDE_PROJECT_DIR is set in hooks/commands per official docs
	// See: https://code.claude.com/docs/en/settings
	if os.Getenv("CLAUDE_PROJECT_DIR") != "" || hasClaudeCodeEnv() {
		return invoker, "claude-code"
	}
	// GitHub Copilot CLI: COPILOT_MODEL is used for model selection
	// See: https://docs.github.com/en/copilot/how-tos/use-copilot-agents/use-copilot-cli
	if os.Getenv("COPILOT_MODEL") != "" || os.Getenv("GH_COPILOT") != "" {
		return invoker, "github-copilot"
	}
	// Note: Cursor, Windsurf, Aider detection is speculative - no official docs confirm these
	// Users can set NYLAS_INVOKER_SOURCE=<tool> to override detection
	if override := os.Getenv("NYLAS_INVOKER_SOURCE"); override != "" {
		return invoker, override
	}

	// 2. SSH
	if os.Getenv("SSH_CLIENT") != "" {
		return invoker, "ssh"
	}

	// 3. Non-interactive (script/automation)
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return invoker, "script"
	}

	// 4. Default: terminal
	return invoker, "terminal"
}

// hasClaudeCodeEnv checks for any CLAUDE_CODE_ prefixed environment variable.
func hasClaudeCodeEnv() bool {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "CLAUDE_CODE_") {
			return true
		}
	}
	return false
}

// getUsername returns the current username.
func getUsername() string {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		return sudoUser
	}
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return "unknown"
}
