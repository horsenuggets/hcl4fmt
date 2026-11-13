package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/horsenuggets/tf4fmt/src/version"
)

func TestRunFileModes(t *testing.T) {
	// All sixteen combinations pin the interaction between these four flags.
	for mask := 0; mask < 16; mask++ {
		check, write, list, diff := mask&1 != 0, mask&2 != 0, mask&4 != 0, mask&8 != 0
		name := fmt.Sprintf("check=%t/write=%t/list=%t/diff=%t", check, write, list, diff)
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "main.tf")
			before, after := "value=1\n", "value = 1\n"
			if err := os.WriteFile(path, []byte(before), 0600); err != nil {
				t.Fatal(err)
			}
			args := []string{fmt.Sprintf("-check=%t", check), fmt.Sprintf("-write=%t", write), fmt.Sprintf("-list=%t", list), fmt.Sprintf("-diff=%t", diff), path}
			var stdout, stderr bytes.Buffer
			code := run(args, strings.NewReader(""), &stdout, &stderr)
			wantCode := 0
			if check {
				wantCode = 3
			}
			if code != wantCode || stderr.Len() != 0 {
				t.Fatalf("Unexpected exit %d and stderr %q.", code, stderr.String())
			}
			wantOutput := ""
			if list {
				wantOutput += path + "\n"
			}
			if diff {
				wantOutput += "--- old/" + path + "\n+++ new/" + path + "\n@@ -1 +1 @@\n-value=1\n+value = 1\n"
			}
			if !check && !diff && !list && !write {
				wantOutput += after
			}
			if stdout.String() != wantOutput {
				t.Fatalf("Unexpected stdout. Got %q, want %q.", stdout.String(), wantOutput)
			}
			wantFile := before
			if write && !check {
				wantFile = after
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != wantFile {
				t.Fatalf("Unexpected saved content %q with error %v.", got, err)
			}
		})
	}
}

func TestRunDefaultsAndCleanFiles(t *testing.T) {
	for _, tc := range []struct {
		name, source, wantFile string
		args                   []string
		listed                 bool
	}{
		{"default writes and lists", "value=1\n", "value = 1\n", nil, true},
		{"clean check", "value = 1\n", "value = 1\n", []string{"-check", "-diff"}, false},
		{"clean default", "value = 1\n", "value = 1\n", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "main.tf")
			if err := os.WriteFile(path, []byte(tc.source), 0600); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			args := append(append([]string(nil), tc.args...), path)
			if code := run(args, strings.NewReader(""), &stdout, &stderr); code != 0 || stderr.Len() != 0 {
				t.Fatalf("Unexpected exit %d and stderr %q.", code, stderr.String())
			}
			wantOutput := ""
			if tc.listed {
				wantOutput = path + "\n"
			}
			if stdout.String() != wantOutput {
				t.Fatalf("Unexpected stdout %q.", stdout.String())
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != tc.wantFile {
				t.Fatalf("Unexpected saved content %q with error %v.", got, err)
			}
		})
	}
}

func TestRunStdin(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		args               []string
		code               int
	}{
		{"default", "value=1\n", "value = 1\n", nil, 0},
		{"write and list disabled", "value=1\n", "value = 1\n", []string{"-write=false", "-list=false"}, 0},
		{"changed check", "value=1\n", "", []string{"-check", "-write=true"}, 3},
		{"clean check", "value = 1\n", "", []string{"-check"}, 0},
		{"diff", "value=1\n", "--- old/stdin.tf\n+++ new/stdin.tf\n@@ -1 +1 @@\n-value=1\n+value = 1\n", []string{"-diff"}, 0},
		{"empty", "", "", nil, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			args := append(append([]string(nil), tc.args...), "-")
			code := run(args, strings.NewReader(tc.source), &stdout, &stderr)
			if code != tc.code || stdout.String() != tc.want || stderr.Len() != 0 {
				t.Fatalf("Unexpected exit %d, stdout %q, and stderr %q.", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunInvalidInputDoesNotOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.tf")
	source := "locals {\nvalue = [\n"
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{path, "-"} {
		t.Run(target, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run([]string{"-diff", target}, strings.NewReader(source), &stdout, &stderr)
			if code != 2 || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("Unexpected exit %d, stdout %q, and stderr %q.", code, stdout.String(), stderr.String())
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != source {
				t.Fatalf("Invalid file was overwritten. Got %q with error %v.", got, err)
			}
		})
	}
}

func TestRunRecursion(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0755); err != nil {
		t.Fatal(err)
	}
	parent, child := filepath.Join(root, "main.tf"), filepath.Join(nested, "child.tf")
	for _, path := range []string{parent, child} {
		if err := os.WriteFile(path, []byte("value=1\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, recursive := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		code := run([]string{"-check", fmt.Sprintf("-recursive=%t", recursive), root}, strings.NewReader(""), &stdout, &stderr)
		want := parent + "\n"
		if recursive {
			want += child + "\n"
		}
		if code != 3 || stdout.String() != want || stderr.Len() != 0 {
			t.Fatalf("Unexpected recursion result. Exit %d, stdout %q, stderr %q.", code, stdout.String(), stderr.String())
		}
	}
}

func TestRunDependencyLockFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".terraform.lock.hcl")
	before := "provider \"registry.terraform.io/hashicorp/null\" {\nversion    = \"3.2.4\"\nconstraints = \">= 3.0.0\"\nhashes=[\n\"h1:abc\",\n]\n}\n"
	after := "provider \"registry.terraform.io/hashicorp/null\" {\n    version = \"3.2.4\"\n    constraints = \">= 3.0.0\"\n    hashes = [\n        \"h1:abc\",\n    ]\n}\n"
	if err := os.WriteFile(path, []byte(before), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{path}, strings.NewReader(""), &stdout, &stderr); code != 0 || stdout.String() != path+"\n" || stderr.Len() != 0 {
		t.Fatalf("Unexpected lockfile result. Exit %d, stdout %q, stderr %q.", code, stdout.String(), stderr.String())
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != after {
		t.Fatalf("Unexpected lockfile content %q with error %v.", got, err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"-check", "-recursive", root}, strings.NewReader(""), &stdout, &stderr); code != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("Lockfile was not clean during discovery. Exit %d, stdout %q, stderr %q.", code, stdout.String(), stderr.String())
	}
}

func TestRunBadArgumentsAndPaths(t *testing.T) {
	root := t.TempDir()
	unsupported := filepath.Join(root, "main.tf.json")
	if err := os.WriteFile(unsupported, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
		code int
	}{
		{"unknown flag", []string{"-unknown"}, 1},
		{"invalid boolean", []string{"-write=maybe"}, 1},
		{"mixed stdin", []string{"-", root}, 1},
		{"missing path", []string{filepath.Join(root, "missing.tf")}, 2},
		{"unsupported path", []string{unsupported}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tc.args, strings.NewReader(""), &stdout, &stderr)
			if code != tc.code || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("Unexpected exit %d, stdout %q, and stderr %q.", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunHelpAndVersion(t *testing.T) {
	original := version.Version
	t.Cleanup(func() { version.Version = original })
	for _, tc := range []struct {
		name, buildVersion, want string
		args                     []string
		code                     int
	}{
		{"help", "", usage, []string{"-help"}, 0},
		{"short help", "", usage, []string{"-h"}, 0},
		{"version", "test-build", "test-build\n", []string{"-version"}, 0},
		{"missing version", "", "", []string{"-version"}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			version.Version = tc.buildVersion
			var stdout, stderr bytes.Buffer
			code := run(tc.args, strings.NewReader(""), &stdout, &stderr)
			if code != tc.code || stdout.String() != tc.want || (stderr.Len() != 0) != (tc.code != 0) {
				t.Fatalf("Unexpected exit %d, stdout %q, and stderr %q.", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunRejectsDirectorySymlinkAliases(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.tf")
	original := "value=1\n"
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "linked-directory")
	if err := os.Symlink(root, link); err != nil {
		t.Skipf("Symbolic links are unavailable. %v", err)
	}
	separator := string(filepath.Separator)
	for _, target := range []string{link, link + separator, link + separator + "."} {
		var stdout, stderr bytes.Buffer
		if code := run([]string{target}, strings.NewReader(""), &stdout, &stderr); code != 2 {
			t.Fatalf("Expected rejection of %q, got exit %d.", target, code)
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != original {
			t.Fatalf("Symlink target changed. Got %q with error %v.", got, err)
		}
	}
}

type failingStream struct{}

func (failingStream) Read([]byte) (int, error) {
	return 0, errors.New("injected stream failure")
}

func (failingStream) Write([]byte) (int, error) {
	return 0, errors.New("injected stream failure")
}

func TestRunStreamErrors(t *testing.T) {
	original := version.Version
	version.Version = "test-build"
	t.Cleanup(func() { version.Version = original })
	path := filepath.Join(t.TempDir(), "main.tf")
	before := "value=1\n"
	if err := os.WriteFile(path, []byte(before), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
		in   io.Reader
		out  io.Writer
	}{
		{"stdin read", []string{"-"}, failingStream{}, io.Discard},
		{"stdin output", []string{"-"}, strings.NewReader(before), failingStream{}},
		{"diff output", []string{"-diff", "-list=false", path}, strings.NewReader(""), failingStream{}},
		{"list output", []string{path}, strings.NewReader(""), failingStream{}},
		{"formatted file output", []string{"-write=false", "-list=false", path}, strings.NewReader(""), failingStream{}},
		{"help output", []string{"-help"}, strings.NewReader(""), failingStream{}},
		{"short help output", []string{"-h"}, strings.NewReader(""), failingStream{}},
		{"version output", []string{"-version"}, strings.NewReader(""), failingStream{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stderr bytes.Buffer
			code := run(tc.args, tc.in, tc.out, &stderr)
			if code != 2 || !strings.Contains(stderr.String(), "injected stream failure") {
				t.Fatalf("Unexpected exit %d and stderr %q.", code, stderr.String())
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != before {
				t.Fatalf("Output failure overwrote the file. Got %q with error %v.", got, err)
			}
		})
	}
}
