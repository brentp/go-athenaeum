// Packge unsplit is a faster alternative to bytes.Split when a [][]byte is not needed.
package unsplit

import "bytes"

// Split has the Next method.
type Split struct {
	line []byte
	sep  []byte
	off  int
}

// Next returns the next item in the line or nil when done.
func (u *Split) Next() []byte {
	if u.off > len(u.line) {
		return nil
	}
	p := u.off + bytes.Index(u.line[u.off:], u.sep)
	if p == u.off-1 {
		p = len(u.line)
	}
	v := u.line[u.off:p]
	u.off = p + 1
	return v
}

// New returns an iterable of each item in the line.
func New(line []byte, sep []byte) *Split {
	return &Split{line: line, sep: sep}
}
