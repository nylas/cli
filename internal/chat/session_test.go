package chat

import (
	"sync"
	"testing"
)

func TestNewActiveSession(t *testing.T) {
	session := NewActiveSession()

	if session == nil {
		t.Fatal("NewActiveSession returned nil")
	}

	if session.Get() != "" {
		t.Errorf("NewActiveSession should return empty conversation ID, got: %q", session.Get())
	}
}

func TestActiveSession_GetSet(t *testing.T) {
	tests := []struct {
		name           string
		conversationID string
	}{
		{
			name:           "simple ID",
			conversationID: "conv-123",
		},
		{
			name:           "empty ID",
			conversationID: "",
		},
		{
			name:           "long ID",
			conversationID: "conversation-with-very-long-identifier-12345678901234567890",
		},
		{
			name:           "ID with special characters",
			conversationID: "conv-123_test@2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := NewActiveSession()

			// Set the conversation ID
			session.Set(tt.conversationID)

			// Get the conversation ID
			got := session.Get()

			if got != tt.conversationID {
				t.Errorf("Get() = %q, want %q", got, tt.conversationID)
			}
		})
	}
}

func TestActiveSession_Overwrite(t *testing.T) {
	session := NewActiveSession()

	// Set first value
	session.Set("first-conv")
	if got := session.Get(); got != "first-conv" {
		t.Errorf("After first Set(), Get() = %q, want %q", got, "first-conv")
	}

	// Overwrite with second value
	session.Set("second-conv")
	if got := session.Get(); got != "second-conv" {
		t.Errorf("After second Set(), Get() = %q, want %q", got, "second-conv")
	}

	// Overwrite with empty value
	session.Set("")
	if got := session.Get(); got != "" {
		t.Errorf("After Set(\"\"), Get() = %q, want \"\"", got)
	}
}

func TestActiveSession_Concurrent(t *testing.T) {
	session := NewActiveSession()
	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2) // readers and writers

	// Concurrent writers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				session.Set(string(rune(id)))
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = session.Get()
			}
		}()
	}

	wg.Wait()

	// If we got here without race detector warnings, the test passed
	// The final value doesn't matter - we just verify thread safety
	_ = session.Get()
}

func TestActiveSession_ConcurrentReadWrite(t *testing.T) {
	session := NewActiveSession()
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			session.Set("write")
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 1000; i++ {
			_ = session.Get()
		}
		done <- true
	}()

	// Wait for both to complete
	<-done
	<-done

	// Verify final state is accessible
	_ = session.Get()
}
