package output_storage

// Write implements io.Writer for OutputStorage.
// It appends a copy of p to the storage to satisfy io.Writer semantics
// (callers may reuse or mutate p after Write returns).
//
// Behavior:
// - nil receiver: no-op, returns len(p), nil (consistent with other methods).
// - empty input: returns 0, nil.
func (s *OutputStorage) Write(p []byte) (int, error) {
	if s == nil {
		return len(p), nil
	}
	if len(p) == 0 {
		return 0, nil
	}

	// FIXME: The only approach that should allow to avoid this copy is to create pipes manually and read them
	//  manually instead of using the standard library's StdoutPipe(). But some extra work is required to wire up
	//  that approach to Command struct.
	//  See read_pipe.go (untested) for an example of how to read from a pipe without the extra copy.

	//Copy the input to avoid retaining caller's buffer.
	cp := append([]byte(nil), p...)

	s.Append(cp)

	return len(p), nil
}
