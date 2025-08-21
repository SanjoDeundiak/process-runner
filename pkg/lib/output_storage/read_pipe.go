package output_storage

import (
	"errors"

	"golang.org/x/sys/unix"
)

// ReadvPipe fills the provided slices from a pipe FD using a single readv per loop.
// It returns total bytes read until EOF or an error.
// Buffers are advanced in-place; they are NOT reallocated.
// TODO: This is actually not tested and not used, requires additional work to wire up to Command struct.
func ReadvPipe(pipeFD int) ([]byte, error) {
	n, err := unix.IoctlGetInt(pipeFD, FIONREAD)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, n)

	n, err = unix.Read(pipeFD, buf)
	if err != nil {
		if errors.Is(err, unix.EINTR) {
			// retry
			return ReadvPipe(pipeFD)
		}
		if errors.Is(err, unix.EAGAIN) {
			// nonblocking pipe and no data available
			return nil, err
		}

		return nil, err
	}

	if n == 0 {
		// EOF
		return nil, nil
	}

	// TODO: Trim to n?
	return buf, nil
}
