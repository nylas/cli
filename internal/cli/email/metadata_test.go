package email

import (
	"testing"
)

func TestMetadataCommands(t *testing.T) {
	t.Run("metadata command exists", func(t *testing.T) {
		cmd := newMetadataCmd()
		if cmd == nil {
			t.Fatal("expected metadata command to exist")
		}
		if cmd.Use != "metadata" {
			t.Errorf("expected Use to be 'metadata', got %s", cmd.Use)
		}
	})

	t.Run("metadata show command exists", func(t *testing.T) {
		cmd := newMetadataShowCmd()
		if cmd == nil {
			t.Fatal("expected metadata show command to exist")
		}
		if cmd.Use != "show <message-id> [grant-id]" {
			t.Errorf("expected Use to be 'show <message-id> [grant-id]', got %s", cmd.Use)
		}
	})

	t.Run("metadata info command exists", func(t *testing.T) {
		cmd := newMetadataInfoCmd()
		if cmd == nil {
			t.Fatal("expected metadata info command to exist")
		}
		if cmd.Use != "info" {
			t.Errorf("expected Use to be 'info', got %s", cmd.Use)
		}
	})
}

func TestMetadataShowCommand(t *testing.T) {
	t.Run("accepts message-id argument", func(t *testing.T) {
		cmd := newMetadataShowCmd()
		// Test that the command accepts at least 1 argument
		if cmd.Args == nil {
			t.Error("expected Args validator to be set")
		}
	})

	t.Run("has json flag", func(t *testing.T) {
		cmd := newMetadataShowCmd()
		flag := cmd.Flags().Lookup("json")
		if flag == nil {
			t.Error("expected --json flag to exist")
		}
	})
}

func TestMetadataInfoCommand(t *testing.T) {
	t.Run("has no required arguments", func(t *testing.T) {
		cmd := newMetadataInfoCmd()
		// Info command should accept no arguments
		if cmd.Args != nil {
			t.Error("expected info command to have no Args validator")
		}
	})
}
