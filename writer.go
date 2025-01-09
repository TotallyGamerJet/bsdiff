// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2019 Gabriel Ochsenhofer
// SPDX-FileCopyrightText: 2025 TotallyGamerJet

package bsdiff

import (
	"fmt"
	"io"
	"sync"
)

// BufWriter is byte slice buffer that implements io.WriteSeeker
type BufWriter struct {
	lock sync.Mutex
	buf  []byte
	pos  int
}

var _ io.Writer = (*BufWriter)(nil)

// Write the contents of p and return the bytes written
func (m *BufWriter) Write(p []byte) (n int, err error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.buf == nil {
		m.buf = make([]byte, 0)
		m.pos = 0
	}
	minCap := m.pos + len(p)
	if minCap > cap(m.buf) { // Make sure buf has enough capacity:
		buf2 := make([]byte, len(m.buf), minCap+len(p)) // add some extra
		copy(buf2, m.buf)
		m.buf = buf2
	}
	if minCap > len(m.buf) {
		m.buf = m.buf[:minCap]
	}
	copy(m.buf[m.pos:], p)
	m.pos += len(p)
	return len(p), nil
}

// Seek to a position on the byte slice
func (m *BufWriter) Seek(offset int64, whence int) (int64, error) {
	newPos, offs := 0, int(offset)
	switch whence {
	case io.SeekStart:
		newPos = offs
	case io.SeekCurrent:
		newPos = m.pos + offs
	case io.SeekEnd:
		newPos = len(m.buf) + offs
	}
	if newPos < 0 {
		return 0, fmt.Errorf("negative result pos")
	}
	m.pos = newPos
	return int64(newPos), nil
}

// Len returns the length of the internal byte slice
func (m *BufWriter) Len() int {
	return len(m.buf)
}

// Bytes return a copy of the internal byte slice
func (m *BufWriter) Bytes() []byte {
	b2 := make([]byte, len(m.buf))
	copy(b2, m.buf)
	return b2
}
