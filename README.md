# tf4fmt

Terraform formatter with four-space indentation.

`tf4fmt` formats native HCL in `.tf`, `.tfvars`, `.tftest.hcl`, `.tfmock.hcl`,
`.tfquery.hcl`, and `.terraform.lock.hcl` files using four spaces per indentation level.
It uses one space around `=` and before trailing comments. It changes HCL whitespace only.
It does not perform Terraform modernization or rewrite expressions into newer Terraform
syntax. JSON files, including `.tf.json` and `.tfvars.json`, are skipped.

## Usage

Run the formatter on a file or directory...

```sh
tf4fmt main.tf
tf4fmt .
```

Choose the behavior needed for a local edit or automated check...

- **Check** → Use `tf4fmt -check .` to fail when files need formatting.
- **Diff** → Use `tf4fmt -diff -write=false .` to inspect changes without writing files.
- **Recursive** → Use `tf4fmt -recursive .` to include nested directories.
- **Output Only** → Use `tf4fmt -write=false -list=false main.tf` to print formatted HCL
  without writing the file or listing its name.
- **Quiet Listing** → Use `tf4fmt -list=false .` to suppress formatted file names.
- **Standard Input** → Use `tf4fmt - < main.tf` to format stdin and write to stdout.

Place flags before file or directory arguments. Use `-` as an explicit stdin argument.

## Build

Use Go 1.25 and Bash. Clone the repository recursively so the local HCL fork is available
in `submodules/hcl/`...

```sh
git clone --recurse-submodules <repository-url>
cd tf4fmt
scripts/build.sh
./dist/tf4fmt -check .
```

For an existing clone, initialize the dependency before building...

```sh
git submodule update --init --recursive
```

The build script reads `VERSION` and injects it into
`github.com/horsenuggets/tf4fmt/src/version.Version`. Windows builds use the `.exe`
extension. Scripts resolve the repository from their own location, so they work from other
working directories as well.

A direct Go build also requires the initialized submodule...

```sh
go build -o dist/tf4fmt ./src/cmd/tf4fmt
```

Use `scripts/build.sh` when version injection is needed. Installation with
`go install ...@latest` is unsupported because module downloads do not initialize the
local HCL submodule.

## VS Code

Install the HashiCorp Terraform and Custom Local Formatters extensions, and make Go
available on `PATH`. Initialize the pinned formatter submodule recursively. In a trusted,
single-folder repository window, configure a stdin formatter command relative to the
workspace root...

```json
{
  "customLocalFormatters.formatters": [
    {
      "command": "go -C Submodules/tf4fmt run ./src/cmd/tf4fmt -",
      "languages": [
        "terraform",
        "terraform-mock",
        "terraform-search",
        "terraform-test",
        "terraform-vars"
      ]
    }
  ],
  "[terraform]": {
    "editor.defaultFormatter": "jkillian.custom-local-formatters",
    "editor.detectIndentation": false,
    "editor.formatOnSave": true,
    "editor.formatOnSaveMode": "file",
    "editor.insertSpaces": true,
    "editor.tabSize": 4
  }
}
```

Match the submodule directory's actual casing and repeat the language-specific editor
settings for the other registered language IDs. Preserve existing settings and formatter
registrations. Recommend `hashicorp.terraform` and `jkillian.custom-local-formatters` in
`.vscode/extensions.json`; recommendations do not automatically install extensions.

Explicit saves format the buffer through stdin, including files in nested directories and
workspace paths containing spaces. Timer-based Auto Save does not run VS Code's normal
format-on-save participant. Folder-specific commands are unreliable in multi-root
workspaces because Custom Local Formatters reads configuration without a document
resource.

For CLI formatting through `go -C`, pass absolute source targets such as
`"$PWD/Terraform"`. Go changes the process working directory before running the formatter.

## Development

Install golangci-lint and make ShellCheck available through your package manager...

```sh
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
```

Make sure the Go binary installation directory is on `PATH`. The scripts use the following
commands...

- **Format** → Run `scripts/fmt.sh` to apply Go formatting to `src/`.
- **Check Formatting** → Run `scripts/fmt.sh --check` to report unformatted Go files.
- **Lint** → Run `scripts/lint.sh` for formatting checks, Go vet, golangci-lint, and
  ShellCheck.
- **Test Scripts** → Run `tests/scripts.sh` for isolated tooling regression tests with
  mocked Go commands. No Go source or Go toolchain is required for these tests.
- **Test** → Run `scripts/test.sh` for script checks and Go tests with race detection.
- **Test Editor Saves** → With Node.js installed, run `npm ci --prefix tests/vscode` and
  `npm test --prefix tests/vscode`. The tests download VS Code and install both required
  extensions into an isolated profile. Linux requires a display or `xvfb-run`.
- **Cross Compile** → Run `scripts/cross.sh` for Darwin, Linux, and Windows binaries on
  both `amd64` and `arm64`, with CGO disabled.
- **Clean** → Run `scripts/clean.sh` to remove `dist/` and root coverage reports.

CI initializes submodules recursively and reads the Go version from `go.mod`. Its `build`,
`lint`, and `test` jobs build all supported targets, check source and shell scripts, run
Linux tests with race detection, and run regular tests on Windows.
