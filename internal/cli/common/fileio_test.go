//go:build !integration

package common

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeWriteCloser records writes and returns configurable errors.
type fakeWriteCloser struct {
	buf      bytes.Buffer
	writeErr error
	closeErr error
	closed   bool
}

func (f *fakeWriteCloser) Write(p []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return f.buf.Write(p)
}

func (f *fakeWriteCloser) Close() error {
	f.closed = true
	return f.closeErr
}

func TestCopyAndClose(t *testing.T) {
	t.Run("copies and closes on success", func(t *testing.T) {
		dst := &fakeWriteCloser{}

		written, err := CopyAndClose(dst, strings.NewReader("hello"))

		require.NoError(t, err)
		assert.Equal(t, int64(5), written)
		assert.Equal(t, "hello", dst.buf.String())
		assert.True(t, dst.closed)
	})

	t.Run("propagates close error", func(t *testing.T) {
		// Some filesystems only surface write failures at Close; success must
		// not be reported when Close fails.
		closeErr := errors.New("close failed: disk full")
		dst := &fakeWriteCloser{closeErr: closeErr}

		_, err := CopyAndClose(dst, strings.NewReader("hello"))

		assert.ErrorIs(t, err, closeErr)
		assert.True(t, dst.closed)
	})

	t.Run("propagates write error and still closes", func(t *testing.T) {
		writeErr := errors.New("write failed")
		dst := &fakeWriteCloser{writeErr: writeErr}

		_, err := CopyAndClose(dst, strings.NewReader("hello"))

		assert.ErrorIs(t, err, writeErr)
		assert.True(t, dst.closed)
	})

	t.Run("write error takes precedence over close error", func(t *testing.T) {
		writeErr := errors.New("write failed")
		dst := &fakeWriteCloser{writeErr: writeErr, closeErr: errors.New("close failed")}

		_, err := CopyAndClose(dst, strings.NewReader("hello"))

		assert.ErrorIs(t, err, writeErr)
	})
}
