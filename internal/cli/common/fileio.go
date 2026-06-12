package common

import "io"

// CopyAndClose copies src into dst, then closes dst, returning the number of
// bytes written and the first error encountered. The Close error is checked
// because some filesystems only surface write failures at Close time, so
// callers must not report success until both the copy and the close succeed.
func CopyAndClose(dst io.WriteCloser, src io.Reader) (int64, error) {
	written, err := io.Copy(dst, src)
	closeErr := dst.Close()
	if err != nil {
		return written, err
	}
	return written, closeErr
}
