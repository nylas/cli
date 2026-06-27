package rpc

import "testing"

// The token command is a thin wrapper over rpcserver.ResolveToken (covered in
// internal/adapters/rpcserver/auth_test.go). This verifies it's wired into the
// rpc command with the documented --copy flag so scripts can rely on it.
func TestTokenCmd_Wiring(t *testing.T) {
	root := NewRPCCmd()

	cmd, _, err := root.Find([]string{"token"})
	if err != nil {
		t.Fatalf("find token subcommand: %v", err)
	}
	if cmd.Use != "token" {
		t.Fatalf("Use = %q, want %q", cmd.Use, "token")
	}
	if cmd.Flags().Lookup("copy") == nil {
		t.Fatal("token command missing --copy flag")
	}
}
