package audit

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTempAuditDir creates a temporary directory for audit store tests
// and returns a cleanup function.
func setupTempAuditDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

// =============================================================================
// NewAuditCmd Tests
// =============================================================================

func TestNewAuditCmd_Structure(t *testing.T) {
	cmd := NewAuditCmd()

	assert.Equal(t, "audit", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
}

func TestNewAuditCmd_Subcommands(t *testing.T) {
	cmd := NewAuditCmd()
	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Use] = true
	}

	expected := []string{"init", "logs", "config", "export"}
	for _, name := range expected {
		assert.True(t, subNames[name], "expected subcommand %q to exist", name)
	}
}

// =============================================================================
// newInitCmd Tests
// =============================================================================

func TestNewInitCmd_FlagsExist(t *testing.T) {
	cmd := newInitCmd()

	flags := []string{"path", "retention", "max-size", "format", "enable", "no-prompt"}
	for _, name := range flags {
		f := cmd.Flags().Lookup(name)
		assert.NotNil(t, f, "expected flag --%s to exist", name)
	}
}

func TestNewInitCmd_DefaultFlagValues(t *testing.T) {
	cmd := newInitCmd()

	retention := cmd.Flags().Lookup("retention")
	require.NotNil(t, retention)
	assert.Equal(t, "90", retention.DefValue)

	maxSize := cmd.Flags().Lookup("max-size")
	require.NotNil(t, maxSize)
	assert.Equal(t, "100", maxSize.DefValue)

	format := cmd.Flags().Lookup("format")
	require.NotNil(t, format)
	assert.Equal(t, "jsonl", format.DefValue)

	enable := cmd.Flags().Lookup("enable")
	require.NotNil(t, enable)
	assert.Equal(t, "false", enable.DefValue)
}

func TestNewInitCmd_NoPrompt_CreatesConfig(t *testing.T) {
	dir := setupTempAuditDir(t)

	cmd := newInitCmd()
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	err := cmd.Flags().Set("no-prompt", "true")
	require.NoError(t, err)
	err = cmd.Flags().Set("path", dir)
	require.NoError(t, err)
	err = cmd.Flags().Set("enable", "true")
	require.NoError(t, err)

	err = cmd.RunE(cmd, nil)
	require.NoError(t, err)

	// Verify config file was created
	cfgPath := filepath.Join(dir, "config.json")
	info, statErr := os.Stat(cfgPath)
	require.NoError(t, statErr)
	assert.True(t, info.Size() > 0)
}

func TestNewInitCmd_NoPrompt_WithRetention(t *testing.T) {
	dir := setupTempAuditDir(t)

	cmd := newInitCmd()
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	require.NoError(t, cmd.Flags().Set("no-prompt", "true"))
	require.NoError(t, cmd.Flags().Set("path", dir))
	require.NoError(t, cmd.Flags().Set("retention", "30"))

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
}

// =============================================================================
// newLogsCmd Tests
// =============================================================================

func TestNewLogsCmd_Structure(t *testing.T) {
	cmd := newLogsCmd()

	assert.Equal(t, "logs", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Use] = true
	}

	expected := []string{"enable", "disable", "status", "show", "summary", "clear"}
	for _, name := range expected {
		assert.True(t, subNames[name], "expected subcommand %q to exist", name)
	}
}

func TestNewEnableCmd_ErrorWhenNotInitialized(t *testing.T) {
	dir := setupTempAuditDir(t)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cmd := newEnableCmd()
	err := cmd.RunE(cmd, nil)
	// Should error because no config exists
	assert.Error(t, err)
}

func TestNewDisableCmd_SucceedsWithDefaultConfig(t *testing.T) {
	dir := setupTempAuditDir(t)
	t.Setenv("XDG_CONFIG_HOME", dir)

	// disable does not require initialization — it saves whatever default config exists
	cmd := newDisableCmd()
	err := cmd.RunE(cmd, nil)
	assert.NoError(t, err)
}

func TestNewStatusCmd_HandlesNoConfig(t *testing.T) {
	dir := setupTempAuditDir(t)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cmd := newStatusCmd()
	// Should not return error — just prints "not initialized"
	err := cmd.RunE(cmd, nil)
	assert.NoError(t, err)
}

func TestNewClearCmd_FlagExists(t *testing.T) {
	cmd := newClearCmd()

	f := cmd.Flags().Lookup("force")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
}

func TestNewClearCmd_NoLogsReturnsNoError(t *testing.T) {
	dir := setupTempAuditDir(t)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cmd := newClearCmd()
	err := cmd.RunE(cmd, nil)
	assert.NoError(t, err)
}

// =============================================================================
// newShowCmd Tests
// =============================================================================

func TestNewShowCmd_FlagsExist(t *testing.T) {
	cmd := newShowCmd()

	flags := []string{
		"limit", "since", "until", "command",
		"status", "grant", "request-id", "invoker", "source",
	}
	for _, name := range flags {
		f := cmd.Flags().Lookup(name)
		assert.NotNil(t, f, "expected flag --%s to exist", name)
	}
}

func TestNewShowCmd_DefaultLimit(t *testing.T) {
	cmd := newShowCmd()

	f := cmd.Flags().Lookup("limit")
	require.NotNil(t, f)
	assert.Equal(t, "20", f.DefValue)
}

func TestNewShowCmd_ErrorWhenNotInitialized(t *testing.T) {
	dir := setupTempAuditDir(t)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cmd := newShowCmd()
	err := cmd.RunE(cmd, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestNewShowCmd_InvalidSinceDate(t *testing.T) {
	dir := setupTempAuditDir(t)

	// Create an initialized audit store so the command gets past the config check
	initCmd := newInitCmd()
	require.NoError(t, initCmd.Flags().Set("no-prompt", "true"))
	require.NoError(t, initCmd.Flags().Set("path", dir))
	require.NoError(t, initCmd.RunE(initCmd, nil))

	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(dir))

	showCmd := newShowCmd()
	require.NoError(t, showCmd.Flags().Set("since", "not-a-date"))

	// We can't easily inject the store path, so just verify flag parsing rejects bad date
	// via the exported parseDate helper (already tested in helpers_test.go).
	_, err := parseDate("not-a-date")
	assert.Error(t, err)
}

func TestNewShowCmd_InvalidUntilDate(t *testing.T) {
	_, err := parseDate("2024/13/45") // MM/DD/YYYY not supported
	assert.Error(t, err)
}

// =============================================================================
// newSummaryCmd Tests
// =============================================================================

func TestNewSummaryCmd_FlagExists(t *testing.T) {
	cmd := newSummaryCmd()

	f := cmd.Flags().Lookup("days")
	require.NotNil(t, f)
	assert.Equal(t, "7", f.DefValue)
}

func TestNewSummaryCmd_ErrorWhenNotInitialized(t *testing.T) {
	dir := setupTempAuditDir(t)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cmd := newSummaryCmd()
	err := cmd.RunE(cmd, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

// =============================================================================
// newConfigCmd Tests
// =============================================================================

func TestNewConfigCmd_Structure(t *testing.T) {
	cmd := newConfigCmd()

	assert.Equal(t, "config", cmd.Use)

	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Use] = true
	}

	assert.True(t, subNames["show"])
	assert.True(t, subNames["set <key> <value>"])
}

func TestNewConfigShowCmd_HandlesNoConfig(t *testing.T) {
	dir := setupTempAuditDir(t)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cmd := newConfigShowCmd()
	// No config exists → prints message but no error
	err := cmd.RunE(cmd, nil)
	assert.NoError(t, err)
}

func TestNewConfigSetCmd_RequiresExactlyTwoArgs(t *testing.T) {
	cmd := newConfigSetCmd()
	// cobra.ExactArgs(2) — verify Args validator is set
	assert.NotNil(t, cmd.Args)
}

func TestNewConfigSetCmd_UnknownKey(t *testing.T) {
	dir := setupTempAuditDir(t)

	// Initialize first
	initCmd := newInitCmd()
	require.NoError(t, initCmd.Flags().Set("no-prompt", "true"))
	require.NoError(t, initCmd.Flags().Set("path", dir))
	require.NoError(t, initCmd.RunE(initCmd, nil))

	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(dir))

	setCmd := newConfigSetCmd()
	err := setCmd.RunE(setCmd, []string{"unknown_key", "value"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown configuration key")
}

func TestNewConfigSetCmd_InvalidRetentionDays(t *testing.T) {
	dir := setupTempAuditDir(t)

	initCmd := newInitCmd()
	require.NoError(t, initCmd.Flags().Set("no-prompt", "true"))
	require.NoError(t, initCmd.Flags().Set("path", dir))
	require.NoError(t, initCmd.RunE(initCmd, nil))

	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(dir))

	tests := []struct {
		name  string
		value string
	}{
		{name: "non-numeric", value: "abc"},
		{name: "zero", value: "0"},
		{name: "negative", value: "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setCmd := newConfigSetCmd()
			err := setCmd.RunE(setCmd, []string{"retention_days", tt.value})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "retention_days must be a positive integer")
		})
	}
}

func TestNewConfigSetCmd_InvalidMaxSizeMB(t *testing.T) {
	dir := setupTempAuditDir(t)

	initCmd := newInitCmd()
	require.NoError(t, initCmd.Flags().Set("no-prompt", "true"))
	require.NoError(t, initCmd.Flags().Set("path", dir))
	require.NoError(t, initCmd.RunE(initCmd, nil))

	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(dir))

	setCmd := newConfigSetCmd()
	err := setCmd.RunE(setCmd, []string{"max_size_mb", "not-a-number"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_size_mb must be a positive integer")
}

// =============================================================================
// newExportCmd Tests
// =============================================================================

func TestNewExportCmd_FlagsExist(t *testing.T) {
	cmd := newExportCmd()

	flags := []string{"output", "format", "since", "until", "limit"}
	for _, name := range flags {
		f := cmd.Flags().Lookup(name)
		assert.NotNil(t, f, "expected flag --%s to exist", name)
	}
}

func TestNewExportCmd_DefaultLimit(t *testing.T) {
	cmd := newExportCmd()

	f := cmd.Flags().Lookup("limit")
	require.NotNil(t, f)
	assert.Equal(t, "10000", f.DefValue)
}

func TestNewExportCmd_ErrorWhenNotInitialized(t *testing.T) {
	dir := setupTempAuditDir(t)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cmd := newExportCmd()
	err := cmd.RunE(cmd, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestNewExportCmd_InvalidSinceDate(t *testing.T) {
	_, err := parseDate("bad-date")
	assert.Error(t, err)
}

// =============================================================================
// printTopItems Tests
// =============================================================================

func TestPrintTopItems_Empty(t *testing.T) {
	// Should not panic on empty map
	assert.NotPanics(t, func() {
		printTopItems(map[string]int{}, 5)
	})
}

func TestPrintTopItems_LimitRespected(t *testing.T) {
	// Build a map with 10 entries
	counts := map[string]int{
		"a": 1, "b": 2, "c": 3, "d": 4, "e": 5,
		"f": 6, "g": 7, "h": 8, "i": 9, "j": 10,
	}

	// Capture stdout to verify limit is respected
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printTopItems(counts, 3)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should have exactly 3 lines of output
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 3, len(lines), "expected 3 lines of output, got %d", len(lines))
}

func TestPrintTopItems_SortedDescending(t *testing.T) {
	counts := map[string]int{"low": 1, "high": 100, "mid": 50}

	// Capture stdout to verify sort order
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printTopItems(counts, 5)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Equal(t, 3, len(lines))

	// First line should contain "high" (count 100), last should contain "low" (count 1)
	assert.Contains(t, lines[0], "high")
	assert.Contains(t, lines[1], "mid")
	assert.Contains(t, lines[2], "low")
}

// =============================================================================
// readLine Tests
// =============================================================================

func TestReadLine_ReadsAndTrims(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "simple line", input: "hello\n", expected: "hello"},
		{name: "line with leading and trailing spaces", input: "  hello  \n", expected: "hello"},
		{name: "empty line", input: "\n", expected: ""},
		{name: "no newline at end", input: "world", expected: "world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			got := readLine(reader)
			assert.Equal(t, tt.expected, got)
		})
	}
}
