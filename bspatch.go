// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2003-2005 Colin Percival
// SPDX-FileCopyrightText: 2019 Gabriel Ochsenhofer
// SPDX-FileCopyrightText: 2025 TotallyGamerJet

package bsdiff

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/dsnet/compress/bzip2"
)

// Patch takes the oldBinary and a patch file and produces the new binary or an error.
func Patch(oldBinary, patch []byte) (newBinary []byte, err error) {
	oldsize := len(oldBinary)
	var newsize int
	header := make([]byte, 32)
	buf := make([]byte, 8)
	var lenread int
	var i int
	ctrl := make([]int, 3)

	f := bytes.NewReader(patch)

	//	File format:
	//		0	8	"BSDIFF40"
	//		8	8	X
	//		16	8	Y
	//		24	8	sizeof(newfile)
	//		32	X	bzip2(control block)
	//		32+X	Y	bzip2(diff block)
	//		32+X+Y	???	bzip2(extra block)
	//	with control block a set of triples (x,y,z) meaning "add x bytes
	//	from oldBinary to x bytes from the diff block; copy y bytes from the
	//	extra block; seek forwards in oldBinary by z bytes".

	// Read header
	var n int
	if n, err = f.Read(header); err != nil || n < 32 {
		if err != nil {
			return nil, fmt.Errorf("corrupt patch %w", err)
		}
		return nil, fmt.Errorf("corrupt patch (n %v < 32)", n)
	}
	// Check for appropriate magic
	if bytes.Compare(header[:8], []byte("BSDIFF40")) != 0 {
		return nil, fmt.Errorf("corrupt patch (header BSDIFF40)")
	}

	// Read lengths from header
	bzctrllen := offtin(header[8:])
	bzdatalen := offtin(header[16:])
	newsize = offtin(header[24:])

	if bzctrllen < 0 || bzdatalen < 0 || newsize < 0 {
		return nil, fmt.Errorf("corrupt patch (bzctrllen %v bzdatalen %v newsize %v)", bzctrllen, bzdatalen, newsize)
	}

	// Close patch file and re-open it via libbzip2 at the right places
	f = nil
	cpf := bytes.NewReader(patch)
	if _, err := cpf.Seek(32, io.SeekStart); err != nil {
		return nil, err
	}
	cpfbz2, err := bzip2.NewReader(cpf, nil)
	if err != nil {
		return nil, err
	}
	dpf := bytes.NewReader(patch)
	if _, err = dpf.Seek(int64(32+bzctrllen), io.SeekStart); err != nil {
		return nil, err
	}
	dpfbz2, err := bzip2.NewReader(dpf, nil)
	if err != nil {
		return nil, err
	}
	epf := bytes.NewReader(patch)
	if _, err = epf.Seek(int64(32+bzctrllen+bzdatalen), io.SeekStart); err != nil {
		return nil, err
	}
	epfbz2, err := bzip2.NewReader(epf, nil)
	if err != nil {
		return nil, err
	}

	pnew := make([]byte, newsize)

	oldpos := 0
	newpos := 0

	for newpos < newsize {
		// Read control data
		for i = 0; i <= 2; i++ {
			lenread, err = zreadall(cpfbz2, buf, 8)
			if lenread != 8 || (err != nil && err != io.EOF) {
				return nil, fmt.Errorf("corrupt patch or bzstream ended: %w (read: %v/8)", err, lenread)
			}
			ctrl[i] = offtin(buf)
		}
		// Sanity-check
		if newpos+ctrl[0] > newsize {
			return nil, fmt.Errorf("corrupt patch (sanity check)")
		}

		// Read diff string
		// lenread, err = dpfbz2.Read(pnew[newpos : newpos+ctrl[0]])
		lenread, err = zreadall(dpfbz2, pnew[newpos:newpos+ctrl[0]], ctrl[0])
		if lenread < ctrl[0] || (err != nil && err != io.EOF) {
			return nil, fmt.Errorf("corrupt patch or bzstream ended (2): %w", err)
		}
		// Add pold data to diff string
		for i = 0; i < ctrl[0]; i++ {
			if oldpos+i >= 0 && oldpos+i < oldsize {
				pnew[newpos+i] += oldBinary[oldpos+i]
			}
		}

		// Adjust pointers
		newpos += ctrl[0]
		oldpos += ctrl[0]

		// Sanity-check
		if newpos+ctrl[1] > newsize {
			return nil, fmt.Errorf("corrupt patch newpos+ctrl[1] newsize")
		}

		// Read extra string
		// epfbz2.Read was not reading all the requested bytes, probably an internal buffer limitation ?
		// it was encapsulated by zreadall to work around the issue
		lenread, err = zreadall(epfbz2, pnew[newpos:newpos+ctrl[1]], ctrl[1])
		if lenread < ctrl[1] || (err != nil && err != io.EOF) {
			return nil, fmt.Errorf("corrupt patch or bzstream ended (3): %w", err)
		}
		// Adjust pointers
		newpos += ctrl[1]
		oldpos += ctrl[2]
	}

	// Clean up the bzip2 reads
	if err = cpfbz2.Close(); err != nil {
		return nil, err
	}
	if err = dpfbz2.Close(); err != nil {
		return nil, err
	}
	if err = epfbz2.Close(); err != nil {
		return nil, err
	}
	cpfbz2 = nil
	dpfbz2 = nil
	epfbz2 = nil
	cpf = nil
	dpf = nil
	epf = nil

	return pnew, nil
}

// offtin reads an int64 (little endian)
func offtin(buf []byte) int {
	y := binary.LittleEndian.Uint64(buf)
	if (y>>56)&0x80 != 0 {
		return -int(y & 0x7FFFFFFF)
	}
	return int(y & 0x7FFFFFFF)
}

func zreadall(r io.Reader, b []byte, expected int) (int, error) {
	var allread int
	var offset int
	for {
		nread, err := r.Read(b[offset:])
		if nread == expected {
			return nread, err
		}
		if err != nil {
			return allread + nread, err
		}
		allread += nread
		if allread >= expected {
			return allread, nil
		}
		offset += nread
	}
}
