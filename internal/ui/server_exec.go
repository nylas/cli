package ui

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/httputil"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	httputil.WriteJSON(w, status, data)
}

// limitedBody wraps a request body with a size limit.
// Returns an error response if the body exceeds the limit.
func limitedBody(w http.ResponseWriter, r *http.Request) io.ReadCloser {
	return httputil.LimitedBody(w, r, httputil.MaxRequestBodySize)
}

// ExecRequest represents a command execution request.
type ExecRequest struct {
	Command string `json:"command"`
}

// ExecResponse represents a command execution response.
type ExecResponse struct {
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// dangerousChars are shell metacharacters that could indicate injection attempts.
// While exec.CommandContext doesn't use a shell, we reject these for defense in depth.
var dangerousChars = []string{";", "|", "&", "`", "$", "(", ")", "<", ">", "\\", "\n", "\x00"}

// containsDangerousChars checks if a command contains shell metacharacters.
func containsDangerousChars(cmd string) bool {
	for _, char := range dangerousChars {
		if strings.Contains(cmd, char) {
			return true
		}
	}
	return false
}

// Allowed commands for security (whitelist approach).
var allowedCommands = map[string]bool{
	// Auth commands
	"auth login":     true,
	"auth logout":    true,
	"auth status":    true,
	"auth whoami":    true,
	"auth list":      true,
	"auth show":      true,
	"auth switch":    true,
	"auth add":       true,
	"auth remove":    true,
	"auth revoke":    true,
	"auth config":    true,
	"auth providers": true,
	"auth detect":    true,
	"auth scopes":    true,
	"auth token":     true,
	"auth migrate":   true,
	// Email commands
	"email list":          true,
	"email read":          true,
	"email send":          true,
	"email search":        true,
	"email delete":        true,
	"email mark":          true,
	"email drafts":        true,
	"email folders":       true,
	"email threads":       true,
	"email scheduled":     true,
	"email attachments":   true,
	"email metadata":      true,
	"email tracking-info": true,
	"email ai":            true,
	"email smart-compose": true,
	// Email folder subcommands
	"email folders list":   true,
	"email folders show":   true,
	"email folders create": true,
	"email folders rename": true,
	"email folders delete": true,
	// Email drafts subcommands
	"email drafts list":   true,
	"email drafts show":   true,
	"email drafts create": true,
	"email drafts delete": true,
	"email drafts send":   true,
	// Email threads subcommands
	"email threads list":   true,
	"email threads show":   true,
	"email threads search": true,
	"email threads delete": true,
	"email threads mark":   true,
	// Email scheduled subcommands
	"email scheduled list":   true,
	"email scheduled show":   true,
	"email scheduled cancel": true,
	// Email attachments subcommands
	"email attachments list":     true,
	"email attachments show":     true,
	"email attachments download": true,
	// Calendar commands
	"calendar list":         true,
	"calendar show":         true,
	"calendar create":       true,
	"calendar update":       true,
	"calendar delete":       true,
	"calendar events":       true,
	"calendar availability": true,
	"calendar find-time":    true,
	"calendar recurring":    true,
	"calendar schedule":     true,
	"calendar virtual":      true,
	"calendar ai":           true,
	// Calendar events subcommands
	"calendar events list":   true,
	"calendar events show":   true,
	"calendar events create": true,
	"calendar events update": true,
	"calendar events delete": true,
	"calendar events rsvp":   true,
	// Calendar availability subcommands
	"calendar availability check": true,
	"calendar availability find":  true,
	// Contacts commands
	"contacts list":   true,
	"contacts show":   true,
	"contacts create": true,
	"contacts update": true,
	"contacts delete": true,
	"contacts groups": true,
	"contacts search": true,
	"contacts photo":  true,
	"contacts sync":   true,
	// Contacts groups subcommands
	"contacts groups list":   true,
	"contacts groups show":   true,
	"contacts groups create": true,
	"contacts groups delete": true,
	// Scheduler commands
	"scheduler configurations": true,
	"scheduler sessions":       true,
	"scheduler bookings":       true,
	"scheduler pages":          true,
	// Scheduler configurations subcommands
	"scheduler configurations list":   true,
	"scheduler configurations show":   true,
	"scheduler configurations create": true,
	"scheduler configurations update": true,
	"scheduler configurations delete": true,
	// Scheduler sessions subcommands
	"scheduler sessions list":   true,
	"scheduler sessions show":   true,
	"scheduler sessions create": true,
	"scheduler sessions delete": true,
	// Scheduler bookings subcommands
	"scheduler bookings list":    true,
	"scheduler bookings show":    true,
	"scheduler bookings create":  true,
	"scheduler bookings confirm": true,
	"scheduler bookings cancel":  true,
	"scheduler bookings delete":  true,
	// Scheduler pages subcommands
	"scheduler pages list":   true,
	"scheduler pages show":   true,
	"scheduler pages create": true,
	"scheduler pages update": true,
	"scheduler pages delete": true,
	// Timezone commands (offline utilities)
	"timezone list":         true,
	"timezone info":         true,
	"timezone convert":      true,
	"timezone find-meeting": true,
	"timezone dst":          true,
	// Webhook commands
	"webhook list":     true,
	"webhook show":     true,
	"webhook create":   true,
	"webhook update":   true,
	"webhook delete":   true,
	"webhook triggers": true,
	"webhook test":     true,
	"webhook server":   true,
	// OTP commands
	"otp get":      true,
	"otp watch":    true,
	"otp list":     true,
	"otp messages": true,
	// Admin commands
	"admin applications": true,
	"admin connectors":   true,
	"admin credentials":  true,
	"admin grants":       true,
	// Admin applications subcommands
	"admin applications list": true,
	"admin applications show": true,
	// Admin connectors subcommands
	"admin connectors list":   true,
	"admin connectors show":   true,
	"admin connectors create": true,
	"admin connectors update": true,
	"admin connectors delete": true,
	// Admin credentials subcommands
	"admin credentials list":   true,
	"admin credentials show":   true,
	"admin credentials create": true,
	"admin credentials delete": true,
	// Admin grants subcommands
	"admin grants list":   true,
	"admin grants show":   true,
	"admin grants delete": true,
	// Notetaker commands
	"notetaker list":   true,
	"notetaker show":   true,
	"notetaker create": true,
	"notetaker delete": true,
	"notetaker media":  true,
	// Other
	"version": true,
}

func (s *Server) handleExecCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecRequest
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ExecResponse{
			Error: "Invalid request body",
		})
		return
	}

	// Demo mode: return sample output
	if s.demoMode {
		writeJSON(w, http.StatusOK, ExecResponse{
			Output: getDemoCommandOutput(req.Command),
		})
		return
	}

	// Validate command is allowed (check base command)
	cmd := strings.TrimSpace(req.Command)

	// Check for empty command
	if cmd == "" {
		writeJSON(w, http.StatusForbidden, ExecResponse{
			Error: "Command not allowed: empty command",
		})
		return
	}

	// Check for shell metacharacters (defense in depth)
	if containsDangerousChars(cmd) {
		writeJSON(w, http.StatusForbidden, ExecResponse{
			Error: "Command not allowed: contains dangerous characters",
		})
		return
	}

	args := strings.Fields(cmd)

	// Extract base command - try 3 words, then 2, then 1
	// e.g., "calendar events list --days 7" -> "calendar events list"
	baseCmd := ""
	allowed := false

	// Try 3-word command first (e.g., "calendar events list")
	if len(args) >= 3 {
		baseCmd = args[0] + " " + args[1] + " " + args[2]
		allowed = allowedCommands[baseCmd]
	}

	// Try 2-word command (e.g., "email list")
	if !allowed && len(args) >= 2 {
		baseCmd = args[0] + " " + args[1]
		allowed = allowedCommands[baseCmd]
	}

	// Try 1-word command (e.g., "version")
	if !allowed && len(args) >= 1 {
		baseCmd = args[0]
		allowed = allowedCommands[baseCmd]
	}

	if !allowed {
		writeJSON(w, http.StatusForbidden, ExecResponse{
			Error: "Command not allowed: " + cmd,
		})
		return
	}

	// Execute the nylas command using the same binary that started this server
	ctx, cancel := common.CreateContext()
	defer cancel()

	// Use the current executable path instead of relying on PATH
	execPath, err := os.Executable()
	if err != nil {
		execPath = "nylas" // Fallback to PATH lookup
	}

	// #nosec G204 -- execPath is the current binary from os.Executable(), user input validated in args (not in execPath)
	execCmd := exec.CommandContext(ctx, execPath, args...)
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	err = execCmd.Run()

	output := stdout.String()
	if output == "" {
		output = stderr.String()
	}

	if err != nil {
		// Command failed but may still have useful output
		if output != "" {
			writeJSON(w, http.StatusOK, ExecResponse{
				Output: output,
			})
		} else {
			writeJSON(w, http.StatusOK, ExecResponse{
				Error: "Command failed: " + err.Error(),
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, ExecResponse{
		Output: output,
	})
}
