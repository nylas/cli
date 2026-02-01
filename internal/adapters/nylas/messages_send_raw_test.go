package nylas

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendRawMessage_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/v3/grants/test-grant/messages/send") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("type") != "mime" {
			t.Errorf("Expected type=mime query parameter")
		}

		// Validate Content-Type is multipart/form-data
		contentType := r.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "multipart/form-data") {
			t.Errorf("Expected multipart/form-data, got %s", contentType)
		}

		// Parse multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("Failed to parse multipart form: %v", err)
		}

		// Validate MIME field exists
		mimeData := r.FormValue("mime")
		if mimeData == "" {
			t.Error("MIME field is empty")
		}
		if !strings.Contains(mimeData, "MIME-Version: 1.0") {
			t.Error("MIME data missing MIME-Version header")
		}

		// Return success response
		resp := struct {
			Data messageResponse `json:"data"`
		}{
			Data: messageResponse{
				ID:      "msg-123",
				GrantID: "test-grant",
				Object:  "message",
				Subject: "Test",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client
	client := NewHTTPClient()
	client.SetBaseURL(server.URL)
	client.SetCredentials("", "", "test-api-key")

	// Test raw MIME message
	rawMIME := []byte("MIME-Version: 1.0\r\nFrom: test@example.com\r\nTo: recipient@example.com\r\nSubject: Test\r\n\r\nTest body")

	msg, err := client.SendRawMessage(context.Background(), "test-grant", rawMIME)
	if err != nil {
		t.Fatalf("SendRawMessage() error = %v", err)
	}

	if msg == nil {
		t.Fatal("SendRawMessage() returned nil message")
	}
	if msg.ID != "msg-123" {
		t.Errorf("Expected message ID msg-123, got %s", msg.ID)
	}
	if msg.GrantID != "test-grant" {
		t.Errorf("Expected grant ID test-grant, got %s", msg.GrantID)
	}
}

func TestSendRawMessage_EmptyMIME(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("Failed to parse multipart form: %v", err)
		}

		// Return success even with empty MIME (server accepts it)
		resp := struct {
			Data messageResponse `json:"data"`
		}{
			Data: messageResponse{
				ID:      "msg-empty",
				GrantID: "test-grant",
				Object:  "message",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.SetBaseURL(server.URL)
	client.SetCredentials("", "", "test-api-key")

	// Empty MIME should still work (server validates)
	msg, err := client.SendRawMessage(context.Background(), "test-grant", []byte(""))
	if err != nil {
		t.Fatalf("SendRawMessage() with empty MIME error = %v", err)
	}
	if msg.ID != "msg-empty" {
		t.Errorf("Expected message ID msg-empty, got %s", msg.ID)
	}
}

func TestSendRawMessage_APIError(t *testing.T) {
	// Mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		resp := map[string]any{
			"error": map[string]any{
				"type":    "invalid_request_error",
				"message": "Invalid MIME format",
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.SetBaseURL(server.URL)
	client.SetCredentials("", "", "test-api-key")

	rawMIME := []byte("invalid mime data")

	_, err := client.SendRawMessage(context.Background(), "test-grant", rawMIME)
	if err == nil {
		t.Fatal("Expected error for invalid MIME, got nil")
	}
	if !strings.Contains(err.Error(), "Invalid MIME format") {
		t.Errorf("Expected 'Invalid MIME format' error, got: %v", err)
	}
}

func TestSendRawMessage_NetworkError(t *testing.T) {
	// Use invalid URL to trigger network error
	client := NewHTTPClient()
	client.SetBaseURL("http://invalid-host-12345.example.com")
	client.SetCredentials("", "", "test-api-key")

	rawMIME := []byte("MIME-Version: 1.0\r\n\r\nTest")

	_, err := client.SendRawMessage(context.Background(), "test-grant", rawMIME)
	if err == nil {
		t.Fatal("Expected network error, got nil")
	}
	// Should return domain.ErrNetworkError
	if !strings.Contains(err.Error(), "network error") {
		t.Errorf("Expected network error, got: %v", err)
	}
}

func TestSendRawMessage_InvalidJSON(t *testing.T) {
	// Mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.SetBaseURL(server.URL)
	client.SetCredentials("", "", "test-api-key")

	rawMIME := []byte("MIME-Version: 1.0\r\n\r\nTest")

	_, err := client.SendRawMessage(context.Background(), "test-grant", rawMIME)
	if err == nil {
		t.Fatal("Expected JSON decode error, got nil")
	}
}

func TestSendRawMessage_MultipartFormConstruction(t *testing.T) {
	// Test that multipart form is correctly constructed
	var capturedBody []byte
	var capturedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture body and content type
		capturedContentType = r.Header.Get("Content-Type")
		var err error
		capturedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read body: %v", err)
		}

		// Return success
		resp := struct {
			Data messageResponse `json:"data"`
		}{
			Data: messageResponse{
				ID:      "msg-test",
				GrantID: "test-grant",
				Object:  "message",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.SetBaseURL(server.URL)
	client.SetCredentials("", "", "test-api-key")

	testMIME := []byte("MIME-Version: 1.0\r\nSubject: Test\r\n\r\nBody")

	_, err := client.SendRawMessage(context.Background(), "test-grant", testMIME)
	if err != nil {
		t.Fatalf("SendRawMessage() error = %v", err)
	}

	// Validate Content-Type has boundary
	if !strings.HasPrefix(capturedContentType, "multipart/form-data; boundary=") {
		t.Errorf("Invalid Content-Type: %s", capturedContentType)
	}

	// Parse captured multipart body
	parts := strings.Split(capturedContentType, "boundary=")
	if len(parts) != 2 {
		t.Fatal("Could not extract boundary from Content-Type")
	}
	boundary := parts[1]

	reader := multipart.NewReader(bytes.NewReader(capturedBody), boundary)

	// Read first part (should be "mime" field)
	part, err := reader.NextPart()
	if err != nil {
		t.Fatalf("Failed to read first part: %v", err)
	}

	if part.FormName() != "mime" {
		t.Errorf("Expected form field 'mime', got '%s'", part.FormName())
	}

	mimeData, err := io.ReadAll(part)
	if err != nil {
		t.Fatalf("Failed to read mime data: %v", err)
	}

	if !bytes.Equal(mimeData, testMIME) {
		t.Errorf("MIME data mismatch.\nExpected: %s\nGot: %s", testMIME, mimeData)
	}

	// Should be no more parts
	_, err = reader.NextPart()
	if err != io.EOF {
		t.Error("Expected only one part in multipart form")
	}
}

func TestSendRawMessage_StatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "200 OK",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "201 Created",
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:       "202 Accepted",
			statusCode: http.StatusAccepted,
			wantErr:    false,
		},
		{
			name:       "400 Bad Request",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "401 Unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)

				if tt.wantErr {
					// Return error response
					resp := map[string]any{
						"error": map[string]any{
							"type":    "api_error",
							"message": "Test error",
						},
					}
					_ = json.NewEncoder(w).Encode(resp)
				} else {
					// Return success response
					resp := struct {
						Data messageResponse `json:"data"`
					}{
						Data: messageResponse{
							ID:      "msg-test",
							GrantID: "test-grant",
							Object:  "message",
						},
					}
					_ = json.NewEncoder(w).Encode(resp)
				}
			}))
			defer server.Close()

			client := NewHTTPClient()
			client.SetBaseURL(server.URL)
			client.SetCredentials("", "", "test-api-key")

			rawMIME := []byte("MIME-Version: 1.0\r\n\r\nTest")

			_, err := client.SendRawMessage(context.Background(), "test-grant", rawMIME)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendRawMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
