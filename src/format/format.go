// Package format formats Terraform HCL with four-space structural indentation.
package format

import (
	"bytes"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// Source validates input before formatting its whitespace. Literal bytes remain intact.
func Source(source []byte, filename string) ([]byte, error) {
	_, diagnostics := hclsyntax.ParseConfig(source, filename, hcl.InitialPos)
	if diagnostics.HasErrors() {
		return nil, diagnostics
	}
	if len(source) > 0 && source[len(source)-1] != '\n' {
		source = append(bytes.Clone(source), '\n')
	}
	formatted := hclwrite.FormatWithIndent(source, 4)
	if len(formatted) == 0 {
		return formatted, nil
	}
	return append(bytes.TrimRight(formatted, "\r\n"), '\n'), nil
}
