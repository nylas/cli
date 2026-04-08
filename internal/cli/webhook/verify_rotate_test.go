package webhook

import (
	"os"
	"testing"

	"github.com/nylas/cli/internal/adapters/webhookserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRotateSecretCommand(t *testing.T) {
	cmd := newRotateSecretCmd()

	assert.Equal(t, "rotate-secret <webhook-id>", cmd.Use)
	flag := cmd.Flags().Lookup("yes")
	assert.NotNil(t, flag)
}

func TestVerifyCommand(t *testing.T) {
	cmd := newVerifyCmd()

	assert.Equal(t, "verify", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("payload"))
	assert.NotNil(t, cmd.Flags().Lookup("payload-file"))
	assert.NotNil(t, cmd.Flags().Lookup("signature"))
	assert.NotNil(t, cmd.Flags().Lookup("secret"))
}

func TestRotateSecretCommand_RequiresYes(t *testing.T) {
	_, _, err := executeCommand(newRotateSecretCmd(), "webhook-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret rotation requires confirmation")
}

func TestVerifyCommand_Validation(t *testing.T) {
	payload := `{"type":"message.created"}`
	secret := "test-secret"
	signature := webhookserver.ComputeSignature([]byte(payload), secret)

	_, _, err := executeCommand(newVerifyCmd(), "--payload", payload, "--signature", signature, "--secret", secret)
	require.NoError(t, err)

	_, _, err = executeCommand(newVerifyCmd(), "--payload", payload, "--signature", "invalid", "--secret", secret)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature verification failed")

	_, _, err = executeCommand(newVerifyCmd(), "--payload", payload, "--secret", secret)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--signature")
}

func TestVerifyCommand_PayloadFileValidation(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "verify-payload-*.json")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpFile.Name()))
	}()

	_, err = tmpFile.WriteString(`{"type":"message.created"}`)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	secret := "test-secret"
	signature := webhookserver.ComputeSignature([]byte(`{"type":"message.created"}`), secret)

	_, _, err = executeCommand(newVerifyCmd(),
		"--payload", `{"type":"message.created"}`,
		"--payload-file", tmpFile.Name(),
		"--signature", signature,
		"--secret", secret,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only one of --payload or --payload-file")
}
