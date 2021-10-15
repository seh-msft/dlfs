// Copyright (c) 2020 Microsoft Corporation, Sean Hinchee.
// Licensed under the MIT License.

// Misc utility functions
package main

import (
	"fmt"
	"os"
)

// Maximum of two integrals
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Return Missing = { x | x ∈ Remote, x ∉ Local}
func missingLocally(local, remote []string) []string {
	lCounts := make(map[string]uint)
	rCounts := make(map[string]uint)
	missing := make([]string, 0, max(len(local), len(remote)))

	for _, s := range local {
		lCounts[s]++
	}

	for _, s := range remote {
		rCounts[s]++
	}

	for key, _ := range rCounts {
		if _, ok := lCounts[key]; !ok {
			missing = append(missing, key)
		}
	}

	return missing
}

// Return Missing = { x | x ∉ Remote, x ∈ Local}
func missingRemotely(local, remote []string) []string {
	lCounts := make(map[string]uint)
	rCounts := make(map[string]uint)
	missing := make([]string, 0, max(len(local), len(remote)))

	for _, s := range local {
		lCounts[s]++
	}

	for _, s := range remote {
		rCounts[s]++
	}

	for key, _ := range lCounts {
		if _, ok := rCounts[key]; !ok {
			missing = append(missing, key)
		}
	}

	return missing
}

// Return A ∩ B
func intersect(a, b []string) []string {
	counts := make(map[string]uint)
	intersect := make([]string, 0, max(len(a), len(b)))

	scan := func(set []string) {
		for _, s := range set {
			counts[s]++
		}
	}

	scan(a)
	scan(b)

	// Take out everything that only occurs once
	for key, value := range counts {
		if value < 2 {
			intersect = append(intersect, key)
		}
	}

	return intersect
}

// Print a tree, nicely
func (t *File) String() string {
	var descend func(depth uint64, t *File) string

	descend = func(depth uint64, t *File) string {
		s := ""

		for i := uint64(0); i < depth; i++ {
			s += "\t"
		}

		s += t.name

		if t.name != "/" && t.isDir {
			s += "/"
		}

		s += "\n"

		depth++

		for _, child := range t.Children {
			s += descend(depth, child)
		}

		return s
	}

	return descend(0, t)
}

// Prompt the user for a y/n response - only accepts `y`
func prompt(s string) bool {
	var response string
	fmt.Print(s + " [y/n]: ")
	fmt.Scanln(&response)
	if response == "y" {
		return true
	}

	return false
}

// Fatal - end program with an error message and newline
func fatal(s ...interface{}) {
	fmt.Fprintln(os.Stderr, s...)
	os.Exit(1)
}
