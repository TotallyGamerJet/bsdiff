// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 TotallyGamerJet

package main

import (
	"fmt"
	"os"

	"github.com/totallygamerjet/bsdiff"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("usage: " + os.Args[0] + " oldfile newfile patchfile")
		os.Exit(1)
	}
	if err := file(os.Args[1], os.Args[2], os.Args[3]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func file(oldfile, newfile, patchfile string) (err error) {
	var (
		oldbs, patchbs, newbytes []byte
	)
	if oldbs, err = os.ReadFile(oldfile); err != nil {
		return fmt.Errorf("could not read oldfile '%v': %w", oldfile, err)
	}
	if patchbs, err = os.ReadFile(patchfile); err != nil {
		return fmt.Errorf("could not read patchfile '%v': %w", patchfile, err)
	}
	if newbytes, err = bsdiff.Patch(oldbs, patchbs); err != nil {
		return fmt.Errorf("bspatch: %w", err)
	}
	if err = os.WriteFile(newfile, newbytes, 0o644); err != nil {
		return fmt.Errorf("could not create newfile '%v': %w", newfile, err)
	}
	return nil
}
