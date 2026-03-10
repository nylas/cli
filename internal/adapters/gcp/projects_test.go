package gcp

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/googleapi"
)

func TestIsConflict(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"409 conflict", &googleapi.Error{Code: http.StatusConflict}, true},
		{"404 not found", &googleapi.Error{Code: http.StatusNotFound}, false},
		{"500 server error", &googleapi.Error{Code: http.StatusInternalServerError}, false},
		{"non-googleapi error", errors.New("some error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isConflict(tt.err))
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"404 not found", &googleapi.Error{Code: http.StatusNotFound}, true},
		{"409 conflict", &googleapi.Error{Code: http.StatusConflict}, false},
		{"403 forbidden", &googleapi.Error{Code: http.StatusForbidden}, false},
		{"non-googleapi error", errors.New("some error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isNotFound(tt.err))
		})
	}
}

func TestNewClient(t *testing.T) {
	ctx := context.Background()
	client, err := NewClient(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewClientWithOptions(t *testing.T) {
	ctx := context.Background()
	client, err := NewClientWithOptions(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, client)
}
