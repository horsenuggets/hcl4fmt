package format

import (
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

// Diff produces a unified diff without invoking external programs.
func Diff(path string, before, after []byte) (string, error) {
	return difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        diffLines(string(before)),
		B:        diffLines(string(after)),
		FromFile: "old/" + path,
		ToFile:   "new/" + path,
		Context:  3,
	})
}

func diffLines(source string) []string {
	if source == "" {
		return nil
	}
	lines := strings.SplitAfter(source, "\n")
	last := len(lines) - 1
	if lines[last] == "" {
		return lines[:last]
	}
	lines[last] += "\n\\ No newline at end of file\n"
	return lines
}
