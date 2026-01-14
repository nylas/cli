package contacts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchCmd(t *testing.T) {
	cmd := newSearchCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "search", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Search")
	})

	t.Run("has_company_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("company")
		assert.NotNil(t, flag)
	})

	t.Run("has_email_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("email")
		assert.NotNil(t, flag)
	})

	t.Run("has_phone_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("phone")
		assert.NotNil(t, flag)
	})

	t.Run("has_source_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("source")
		assert.NotNil(t, flag)
	})

	t.Run("has_group_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("group")
		assert.NotNil(t, flag)
	})

	t.Run("has_has_email_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("has-email")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has_limit_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("limit")
		assert.NotNil(t, flag)
		assert.Equal(t, "50", flag.DefValue)
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})
}

func TestPhotoCmd(t *testing.T) {
	cmd := newPhotoCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "photo", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "photo")
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)
	})

	t.Run("has_download_subcommand", func(t *testing.T) {
		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}
		assert.True(t, cmdMap["download"], "Missing download subcommand")
	})

	t.Run("has_info_subcommand", func(t *testing.T) {
		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}
		assert.True(t, cmdMap["info"], "Missing info subcommand")
	})
}

func TestPhotoDownloadCmd(t *testing.T) {
	cmd := newPhotoDownloadCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "download <contact-id>", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "Download")
	})

	t.Run("has_output_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("output")
		assert.NotNil(t, flag)
	})

	t.Run("has_output_shorthand", func(t *testing.T) {
		flag := cmd.Flags().ShorthandLookup("o")
		assert.NotNil(t, flag)
		assert.Equal(t, "output", flag.Name)
	})

	t.Run("has_json_flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
	})

	t.Run("requires_args", func(t *testing.T) {
		assert.NotNil(t, cmd.Args)
	})
}

func TestPhotoInfoCmd(t *testing.T) {
	cmd := newPhotoInfoCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "info", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("has_long_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
		assert.Contains(t, cmd.Long, "Profile Picture")
	})
}

func TestSyncCmd(t *testing.T) {
	cmd := newSyncCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "sync", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "sync")
	})

	t.Run("has_long_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
		assert.Contains(t, cmd.Long, "Synchronization")
	})
}

func TestContactsSearchHelp(t *testing.T) {
	cmd := NewContactsCmd()
	stdout, _, err := executeCommand(cmd, "search", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "search")
	assert.Contains(t, stdout, "--company")
	assert.Contains(t, stdout, "--email")
	assert.Contains(t, stdout, "--has-email")
}

func TestContactsPhotoHelp(t *testing.T) {
	cmd := NewContactsCmd()
	stdout, _, err := executeCommand(cmd, "photo", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "photo")
	assert.Contains(t, stdout, "download")
	assert.Contains(t, stdout, "info")
}

func TestContactsSyncHelp(t *testing.T) {
	cmd := NewContactsCmd()
	stdout, _, err := executeCommand(cmd, "sync", "--help")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "sync")
}
