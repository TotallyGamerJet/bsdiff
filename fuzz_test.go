// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 TotallyGamerJet

package bsdiff

import (
	"bytes"
	"testing"
)

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
