package update

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyChecksum(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "artifact.bin")
	content := []byte("hello, updater")
	require.NoError(t, os.WriteFile(filePath, content, 0o600))

	sum := sha256.Sum256(content)
	expected := hex.EncodeToString(sum[:])

	ok, err := verifyChecksum(filePath, expected)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = verifyChecksum(filePath, strings.Repeat("0", len(expected)))
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestExtractFromTarGz(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "nylas.tar.gz")

	f, err := os.Create(archivePath)
	require.NoError(t, err)

	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)

	content := []byte("binary-content")
	header := &tar.Header{
		Name: binaryName,
		Mode: 0o755,
		Size: int64(len(content)),
	}
	require.NoError(t, tw.WriteHeader(header))
	_, err = tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())
	require.NoError(t, f.Close())

	extractedPath, err := extractFromTarGz(archivePath)
	require.NoError(t, err)
	defer func() { _ = os.Remove(extractedPath) }()

	extracted, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	assert.Equal(t, content, extracted)
}

func TestInstallBinaryForOS_WindowsStagesDeferredReplacement(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "nylas.exe")
	newBinaryPath := filepath.Join(dir, "nylas-new.exe")

	require.NoError(t, os.WriteFile(targetPath, []byte("old-binary"), 0o755))
	require.NoError(t, os.WriteFile(newBinaryPath, []byte("new-binary"), 0o755))

	var commandName string
	var commandArgs []string
	oldStarter := startDetachedUpdateCommand
	startDetachedUpdateCommand = func(name string, args ...string) error {
		commandName = name
		commandArgs = append([]string(nil), args...)
		return nil
	}
	defer func() {
		startDetachedUpdateCommand = oldStarter
	}()

	err := installBinaryForOS(newBinaryPath, targetPath, "windows")
	require.NoError(t, err)

	stagedPath := targetPath + ".new"
	staged, err := os.ReadFile(stagedPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("new-binary"), staged)

	assert.Equal(t, "cmd", commandName)
	require.Len(t, commandArgs, 2)
	assert.Equal(t, "/C", commandArgs[0])

	scriptBody, err := os.ReadFile(commandArgs[1])
	require.NoError(t, err)
	assert.Contains(t, string(scriptBody), stagedPath)
	assert.Contains(t, string(scriptBody), targetPath)
	assert.Contains(t, string(scriptBody), `if "%REPLACED%"=="1" goto cleanup`)
	assert.Contains(t, string(scriptBody), `move /y "%BACKUP%" "%TARGET%" >nul 2>&1`)
}

func TestBuildWindowsUpdateScript_PreservesBackupUntilReplacementSucceeds(t *testing.T) {
	t.Parallel()

	script := buildWindowsUpdateScript(
		`C:\Tools\100%\nylas.exe`,
		`C:\Tools\100%\nylas.exe.new`,
		`C:\Tools\100%\nylas.exe.bak`,
	)

	assert.Contains(t, script, `set "TARGET=C:\Tools\100%%\nylas.exe"`)
	assert.Contains(t, script, `set "STAGED=C:\Tools\100%%\nylas.exe.new"`)
	assert.Contains(t, script, `set "BACKUP=C:\Tools\100%%\nylas.exe.bak"`)
	assert.Contains(t, script, `if "%REPLACED%"=="1" goto cleanup`)
	assert.Contains(t, script, `if not exist "%BACKUP%" goto end`)
	assert.Contains(t, script, `move /y "%BACKUP%" "%TARGET%" >nul 2>&1`)
}
