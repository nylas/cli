package mcp

import (
	"bytes"
	"strings"
	"testing"
)

func TestSuccessResponse_MarshalFailure(t *testing.T) {
	t.Parallel()

	// channels cannot be marshaled to JSON, so this triggers the error branch.
	data := successResponse(float64(1), make(chan int))
	if !bytes.Equal(data, fallbackErrorResponse) {
		t.Errorf("got %s, want fallback error response", data)
	}
}

func TestErrorResponse_MarshalFailure(t *testing.T) {
	t.Parallel()

	// An unmarshalable ID triggers the error branch in errorResponse.
	data := errorResponse(make(chan int), codeInternalError, "test")
	if !bytes.Equal(data, fallbackErrorResponse) {
		t.Errorf("got %s, want fallback error response", data)
	}
}

func TestToolSuccess_MarshalFailure(t *testing.T) {
	t.Parallel()

	resp := toolSuccess(make(chan int))
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if !resp.IsError {
		t.Error("expected IsError=true for marshal failure")
	}
	if len(resp.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
	if !strings.Contains(resp.Content[0].Text, "failed to marshal") {
		t.Errorf("text = %q, want it to contain 'failed to marshal'", resp.Content[0].Text)
	}
}
