// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2003-2005 Colin Percival
// SPDX-FileCopyrightText: 2019 Gabriel Ochsenhofer
// SPDX-FileCopyrightText: 2025 TotallyGamerJet

package bsdiff

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/dsnet/compress/bzip2"
)

// Diff takes oldBinary and newBinary and returns a patch file that converts oldBinary into newBinary.
func Diff(oldBinary, newBinary []byte) (patch []byte, err error) {
	bziprule := &bzip2.WriterConfig{
		Level: bzip2.BestCompression,
	}
	iii := make([]int, len(oldBinary)+1)
	qsufsort(iii, oldBinary)

	//var db
	var dblen, eblen int

	// create the patch file
	pf := new(bufWriter)

	// Header is
	//	0	8	 "BSDIFF40"
	//	8	8	length of bzip2ed ctrl block
	//	16	8	length of bzip2ed diff block
	//	24	8	length of pnew file */
	// File is
	//  0	32	Header
	//  32	??	Bzip2ed ctrl block
	//  ??	??	Bzip2ed diff block
	//  ??	??	Bzip2ed extra block

	newsize := len(newBinary)
	oldsize := len(oldBinary)

	header := make([]byte, 32)
	buf := make([]byte, 8)

	copy(header, []byte("BSDIFF40"))
	offtout(0, header[8:])
	offtout(0, header[16:])
	offtout(newsize, header[24:])
	if _, err := pf.Write(header); err != nil {
		return nil, err
	}
	// Compute the differences, writing ctrl as we go
	pfbz2, err := bzip2.NewWriter(pf, bziprule)
	if err != nil {
		return nil, err
	}
	var scan, ln, lastscan, lastpos, lastoffset int

	var oldscore, scsc int
	var pos int

	var s, Sf, lenf, Sb, lenb int
	var overlap, Ss, lens int

	db := make([]byte, newsize+1)
	eb := make([]byte, newsize+1)

	defer func() {
		if pfbz2 == nil {
			return
		}
		err2 := pfbz2.Close()
		if err2 != nil && err == nil {
			err = err2
		}
	}()

	for scan < newsize {
		oldscore = 0

		// scsc = scan += len
		scan += ln
		scsc = scan
		for scan < newsize {
			ln = search(iii, oldBinary, newBinary[scan:], 0, oldsize, &pos)

			for scsc < scan+ln {
				if scsc+lastoffset < oldsize && oldBinary[scsc+lastoffset] == newBinary[scsc] {
					oldscore++
				}
				scsc++
			}
			if ln == oldscore && ln != 0 {
				break
			}
			if ln > oldscore+8 {
				break
			}
			if scan+lastoffset < oldsize && oldBinary[scan+lastoffset] == newBinary[scan] {
				oldscore--
			}
			//
			scan++
		}

		if ln != oldscore || scan == newsize {
			s = 0
			Sf = 0
			lenf = 0
			i := 0
			for lastscan+i < scan && lastpos+i < oldsize {
				if oldBinary[lastpos+i] == newBinary[lastscan+i] {
					s++
				}
				i++
				if s*2-i > Sf*2-lenf {
					Sf = s
					lenf = i
				}
			}

			lenb = 0
			if scan < newsize {
				s = 0
				Sb = 0
				for i = 1; scan >= lastscan+i && pos >= i; i++ {
					if oldBinary[pos-i] == newBinary[scan-i] {
						s++
					}
					if s*2-i > Sb*2-lenb {
						Sb = s
						lenb = i
					}
				}
			}

			if lastscan+lenf > scan-lenb {
				overlap = (lastscan + lenf) - (scan - lenb)
				s = 0
				Ss = 0
				lens = 0
				for i = 0; i < overlap; i++ {
					if newBinary[lastscan+lenf-overlap+i] == oldBinary[lastpos+lenf-overlap+i] {
						s++
					}

					if newBinary[scan-lenb+i] == oldBinary[pos-lenb+i] {
						s--
					}
					if s > Ss {
						Ss = s
						lens = i + 1
					}
				}

				lenf += lens - overlap
				lenb -= lens
			}

			for i = 0; i < lenf; i++ {
				db[dblen+i] = newBinary[lastscan+i] - oldBinary[lastpos+i]
			}
			for i = 0; i < (scan-lenb)-(lastscan+lenf); i++ {
				eb[eblen+i] = newBinary[lastscan+lenf+i]
			}

			dblen += lenf
			eblen += (scan - lenb) - (lastscan + lenf)

			offtout(lenf, buf)
			if _, err = pfbz2.Write(buf); err != nil {
				return nil, err
			}

			offtout((scan-lenb)-(lastscan+lenf), buf)
			if _, err = pfbz2.Write(buf); err != nil {
				return nil, err
			}

			offtout((pos-lenb)-(lastpos+lenf), buf)
			if _, err = pfbz2.Write(buf); err != nil {
				return nil, err
			}

			lastscan = scan - lenb
			lastpos = pos - lenb
			lastoffset = pos - scan
		}
	}
	if err = pfbz2.Close(); err != nil {
		return nil, err
	}

	// Compute size of compressed ctrl data
	ln = pf.Len()
	offtout(ln-32, header[8:])

	// Write compressed diff data
	pfbz2, err = bzip2.NewWriter(pf, bziprule)
	if err != nil {
		return nil, err
	}
	if _, err = pfbz2.Write(db[:dblen]); err != nil {
		return nil, err
	}

	if err = pfbz2.Close(); err != nil {
		return nil, err
	}
	// Compute size of compressed diff data
	newsize = pf.Len()
	offtout(newsize-ln, header[16:])
	// Write compressed extra data
	pfbz2, err = bzip2.NewWriter(pf, bziprule)
	if err != nil {
		return nil, err
	}
	if _, err = pfbz2.Write(eb[:eblen]); err != nil {
		return nil, err
	}
	if err = pfbz2.Close(); err != nil {
		return nil, err
	}
	// Seek to the beginning, write the header, and close the file
	if _, err = pf.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	if _, err = pf.Write(header); err != nil {
		return nil, err
	}

	return pf.Bytes(), nil
}

func search(iii []int, oldbin []byte, newbin []byte, st, en int, pos *int) int {
	var x, y int
	oldsize := len(oldbin)
	newsize := len(newbin)

	if en-st < 2 {
		x = matchlen(oldbin[iii[st]:], newbin)
		y = matchlen(oldbin[iii[en]:], newbin)

		if x > y {
			*pos = iii[st]
			return x
		}
		*pos = iii[en]
		return y
	}

	x = st + (en-st)/2
	cmpln := min(oldsize-iii[x], newsize)
	if bytes.Compare(oldbin[iii[x]:iii[x]+cmpln], newbin[:cmpln]) < 0 {
		return search(iii, oldbin, newbin, x, en, pos)
	}
	return search(iii, oldbin, newbin, st, x, pos)
}

func matchlen(oldbin []byte, newbin []byte) int {
	var i int
	oldsize := len(oldbin)
	newsize := len(newbin)
	for (i < oldsize) && (i < newsize) {
		if oldbin[i] != newbin[i] {
			break
		}
		i++
	}
	return i
}

// offtout puts an int64 (little endian) to buf
func offtout(x int, buf []byte) {
	var y = uint64(x)
	if x < 0 {
		y = uint64(-x) | (0x80 << 56)
	}
	binary.LittleEndian.PutUint64(buf, y)
}

func qsufsort(iii []int, buf []byte) {
	buckets := make([]int, 256)
	vvv := make([]int, len(iii))
	var i, h, ln int
	bufzise := len(buf)

	for i = 0; i < bufzise; i++ {
		buckets[buf[i]]++
	}

	for i = 1; i < 256; i++ {
		buckets[i] += buckets[i-1]
	}

	for i = 255; i > 0; i-- {
		buckets[i] = buckets[i-1]
	}
	buckets[0] = 0

	for i = 0; i < bufzise; i++ {
		buckets[buf[i]]++
		iii[buckets[buf[i]]] = i
	}
	iii[0] = bufzise

	for i = 0; i < bufzise; i++ {
		vvv[i] = buckets[buf[i]]
	}
	vvv[bufzise] = 0

	for i = 1; i < 256; i++ {
		if buckets[i] == buckets[i-1]+1 {
			iii[buckets[i]] = -1
		}
	}
	iii[0] = -1

	for h = 1; iii[0] != -(bufzise + 1); h += h {
		ln = 0

		i = 0
		for i < bufzise+1 {
			if iii[i] < 0 {
				ln -= iii[i]
				i -= iii[i]
			} else {
				if ln != 0 {
					iii[i-ln] = -ln
				}
				ln = vvv[iii[i]] + 1 - i
				split(iii, vvv, i, ln, h)
				i += ln
				ln = 0
			}
		}
		if ln != 0 {
			iii[i-ln] = -ln
		}
	}

	for i = 0; i < bufzise+1; i++ {
		iii[vvv[i]] = i
	}
}

func split(iii, vvv []int, start, ln, h int) {
	var i, j, k, x int

	if ln < 16 {
		for k = start; k < start+ln; k += j {
			j = 1
			x = vvv[iii[k]+h]
			for i = 1; k+i < start+ln; i++ {
				if vvv[iii[k+i]+h] < x {
					x = vvv[iii[k+i]+h]
					j = 0
				}
				if vvv[iii[k+i]+h] == x {
					iii[k+j], iii[k+i] = iii[k+i], iii[k+j]
					j++
				}
			}
			for i = 0; i < j; i++ {
				vvv[iii[k+i]] = k + j - 1
			}
			if j == 1 {
				iii[k] = -1
			}
		}
		return
	}

	x = vvv[iii[start+(ln/2)]+h]
	var jj, kk int
	for i = start; i < start+ln; i++ {
		if vvv[iii[i]+h] < x {
			jj++
		} else if vvv[iii[i]+h] == x {
			kk++
		}
	}
	jj += start
	kk += jj

	i = start
	j = 0
	k = 0
	for i < jj {
		if vvv[iii[i]+h] < x {
			i++
		} else if vvv[iii[i]+h] == x {
			iii[i], iii[jj+j] = iii[jj+j], iii[i]
			j++
		} else {
			iii[i], iii[kk+k] = iii[kk+k], iii[i]
			k++
		}
	}
	for jj+j < kk {
		if vvv[iii[jj+j]+h] == x {
			j++
		} else {
			iii[jj+j], iii[kk+k] = iii[kk+k], iii[jj+j]
			k++
		}
	}
	if jj > start {
		split(iii, vvv, start, jj-start, h)
	}

	for i = 0; i < kk-jj; i++ {
		vvv[iii[jj+i]] = kk - 1
	}
	if jj == kk-1 {
		iii[jj] = -1
	}

	if start+ln > kk {
		split(iii, vvv, kk, start+ln-kk, h)
	}
}
