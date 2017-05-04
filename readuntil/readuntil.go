package readuntil

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

var errFound = errors.New("readuntil: pattern found")
var errNotFound = errors.New("readuntil: pattern not found")

// Index will read the io.Reader until it finds the first instance of pattern
// and return the number of bytes read. If the patter was not found it will
// return io.EOF.
// If the reader is a readseeker then it will seek to that position.
func Index(r io.Reader, pat []byte) (int, error) {
	var n int
	var found bool
	var err error

	sf := func(data []byte, atEOF bool) (int, []byte, error) {
		v := bytes.Index(data, pat)
		if v != -1 {
			n += v
			found = true
			return v, data, errFound
		}
		if atEOF {
			err = errNotFound
			return n + len(data), nil, err
		}
		if len(data) < len(pat) {
			return 0, nil, nil
		}
		off := len(data) - len(pat) + 1
		if off <= 0 {
			off = 1
		}
		n += off

		return off, data[off:], err
	}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 4096), 16384)
	sc.Split(sf)

	for sc.Scan() {
	}
	err = sc.Err()
	if err == errFound {
		err = nil
		if s, ok := r.(io.ReadSeeker); ok {
			s.Seek(int64(n), 0)
		}
	} else if err == errNotFound {
		err = io.EOF
	}

	return n, err
}
