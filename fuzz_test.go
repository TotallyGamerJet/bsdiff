// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 TotallyGamerJet

package bsdiff

import (
	"bytes"
	"testing"
)

func Fuzz_offt(f *testing.F) {
	f.Add(9001)
	f.Add(-9001)
	f.Fuzz(func(t *testing.T, x int) {
		buf := make([]byte, 8)
		offtout(x, buf)

		y := offtin(buf)

		if x != y {
			t.Errorf("x != y: %d != %d", x, y)
		}
	})
}

func FuzzDiffPatch(f *testing.F) {
	f.Add([]byte("Here is text"), []byte("Here are some texts that I have"))
	f.Fuzz(func(t *testing.T, old, new []byte) {
		patch, err := Diff(old, new)
		if err != nil {
			t.Errorf("diff failed: %s", err)
		}
		new2, err := Patch(old, patch)
		if err != nil {
			t.Errorf("patch failed: %s", err)
		}
		if !bytes.Equal(new, new2) {
			t.Errorf("patch did not recreate the file")
		}
	})
}
