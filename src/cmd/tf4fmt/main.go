// Command tf4fmt formats native Terraform HCL with four-space indentation.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/horsenuggets/tf4fmt/src/format"
	"github.com/horsenuggets/tf4fmt/src/version"
)

const usage = `Terraform formatter with four-space indentation.

[Usage]
    tf4fmt [options] [file or directory ...]
    tf4fmt [options] -

Options...
    -check          Check formatting without writing files.
    -diff           Show unified diffs for changed files.
    -help           Show this help message.
    -list=false     Suppress changed file names.
    -no-color       Produce uncolored output (always enabled).
    -recursive      Visit subdirectories.
    -version        Show the build version.
    -write=false    Do not write files.

With no target, format the current directory. Use - for standard input.
JSON files are not supported. Only HCL whitespace is formatted.
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("tf4fmt", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	check := flags.Bool("check", false, "Check formatting.")
	diff := flags.Bool("diff", false, "Show differences.")
	help := flags.Bool("help", false, "Show help.")
	list := flags.Bool("list", true, "List changed files.")
	flags.Bool("no-color", false, "Disable color.")
	recursive := flags.Bool("recursive", false, "Visit subdirectories.")
	showVersion := flags.Bool("version", false, "Show version.")
	write := flags.Bool("write", true, "Write formatted files.")
	report := func(err error) int {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			if _, err = io.WriteString(stdout, usage); err != nil {
				return report(err)
			}
			return 0
		}
		_, _ = fmt.Fprintf(stderr, "Invalid arguments. %s\n%s", err, usage)
		return 1
	}
	if *help {
		if _, err := io.WriteString(stdout, usage); err != nil {
			return report(err)
		}
		return 0
	}
	if *showVersion {
		if version.Version == "" {
			_, _ = fmt.Fprintln(stderr, "Build version is missing. Build with scripts/build.sh.")
			return 2
		}
		if _, err := fmt.Fprintln(stdout, version.Version); err != nil {
			return report(err)
		}
		return 0
	}
	targets := flags.Args()
	for _, target := range targets {
		if target == "-" && len(targets) != 1 {
			_, _ = fmt.Fprintln(stderr, "Standard input must be the only target.")
			return 1
		}
	}
	if *check {
		*write = false
	}
	changed := false
	process := func(path string, source []byte, isStdin bool) error {
		formatted, err := format.Source(source, path)
		if err != nil {
			return err
		}
		different := !bytes.Equal(source, formatted)
		changed = changed || different
		if different {
			if *list && !isStdin {
				if _, err := fmt.Fprintln(stdout, path); err != nil {
					return err
				}
			}
			if *diff {
				text, err := format.Diff(path, source, formatted)
				if err != nil {
					return err
				}
				if _, err := io.WriteString(stdout, text); err != nil {
					return err
				}
			}
			if *write && !isStdin {
				if err := format.Write(path, formatted); err != nil {
					return err
				}
			}
		}
		if !*check && !*diff && (isStdin || (!*list && !*write)) {
			_, err = stdout.Write(formatted)
		}
		return err
	}
	if len(targets) == 1 && targets[0] == "-" {
		source, err := io.ReadAll(stdin)
		if err != nil {
			return report(err)
		}
		if err := process("stdin.tf", source, true); err != nil {
			return report(err)
		}
	} else {
		if len(targets) == 0 {
			targets = []string{"."}
		}
		paths, err := format.Discover(targets, *recursive)
		if err != nil {
			return report(err)
		}
		failed := false
		for _, path := range paths {
			source, err := os.ReadFile(path)
			if err == nil {
				err = process(path, source, false)
			}
			if err != nil {
				report(err)
				failed = true
			}
		}
		if failed {
			return 2
		}
	}
	if *check && changed {
		return 3
	}
	return 0
}
