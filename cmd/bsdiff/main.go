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
		oldbs, newbs, diffbytes []byte
	)
	if oldbs, err = os.ReadFile(oldfile); err != nil {
		return fmt.Errorf("could not read oldfile '%v': %w", oldfile, err)
	}
	if newbs, err = os.ReadFile(newfile); err != nil {
		return fmt.Errorf("could not read newfile '%v': %w", newfile, err)
	}
	if diffbytes, err = bsdiff.Diff(oldbs, newbs); err != nil {
		return fmt.Errorf("bsdiff: %w", err)
	}
	if err = os.WriteFile(patchfile, diffbytes, 0o644); err != nil {
		return fmt.Errorf("could not create patchfile '%v': %w", patchfile, err)
	}
	return nil
}
