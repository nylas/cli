//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// FOLDER COMMAND TESTS
// =============================================================================

func TestCLI_FoldersList(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "folders", "list", testGrantID)
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("folders list failed: %v\nstderr: %s", err, stderr)
	}

	// Should show folders header
	if !strings.Contains(stdout, "Folders") || !strings.Contains(stdout, "NAME") {
		t.Errorf("Expected folders list header, got: %s", stdout)
	}

	// Should contain common folders like Inbox
	if !strings.Contains(stdout, "Inbox") && !strings.Contains(stdout, "INBOX") {
		t.Errorf("Expected 'Inbox' folder in output, got: %s", stdout)
	}

	t.Logf("folders list output:\n%s", stdout)
}

func TestCLI_FoldersListWithID(t *testing.T) {
	skipIfMissingCreds(t)

	stdout, stderr, err := runCLI("email", "folders", "list", testGrantID, "--id")
	skipIfProviderNotSupported(t, stderr)

	if err != nil {
		t.Fatalf("folders list --id failed: %v\nstderr: %s", err, stderr)
	}

	// Should show folders with ID column
	if !strings.Contains(stdout, "Folders") {
		t.Errorf("Expected folders header, got: %s", stdout)
	}

	// Should contain ID column in header
	if !strings.Contains(stdout, "ID") {
		t.Errorf("Expected 'ID' column header with --id flag, got: %s", stdout)
	}

	// Should NOT show the hint to use --id (since we already used it)
	if strings.Contains(stdout, "Use --id to see folder IDs") {
		t.Errorf("Should not show --id hint when flag is already used, got: %s", stdout)
	}

	t.Logf("folders list --id output:\n%s", stdout)
}

func TestCLI_FoldersCreateAndDelete(t *testing.T) {
	skipIfMissingCreds(t)

	if os.Getenv("NYLAS_TEST_DELETE") != "true" {
		t.Skip("NYLAS_TEST_DELETE not set to 'true'")
	}

	folderName := fmt.Sprintf("CLI-Test-%d", time.Now().Unix())

	// Create folder
	t.Run("create", func(t *testing.T) {
		stdout, stderr, err := runCLI("email", "folders", "create", folderName, testGrantID)
		if err != nil {
			t.Fatalf("folders create failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "Created folder") {
			t.Errorf("Expected 'Created folder' in output, got: %s", stdout)
		}

		t.Logf("folders create output: %s", stdout)
	})

	// Wait for folder to be created
	time.Sleep(2 * time.Second)

	// Get folder ID
	client := getTestClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	folders, err := client.GetFolders(ctx, testGrantID)
	if err != nil {
		t.Fatalf("Failed to get folders: %v", err)
	}

	var folderID string
	for _, f := range folders {
		if f.Name == folderName {
			folderID = f.ID
			break
		}
	}

	if folderID == "" {
		t.Skip("Created folder not found - may need more time to sync")
	}

	// Delete folder
	t.Run("delete", func(t *testing.T) {
		stdout, stderr, err := runCLIWithInput("y\n", "email", "folders", "delete", folderID, testGrantID)
		if err != nil {
			t.Fatalf("folders delete failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "deleted") {
			t.Errorf("Expected 'deleted' in output, got: %s", stdout)
		}

		t.Logf("folders delete output: %s", stdout)
	})
}
