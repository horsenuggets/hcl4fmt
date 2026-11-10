package format

import (
	"bytes"
	"strings"
	"testing"
)

func TestSourceGolden(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
	}{
		{
			name:   "nested blocks",
			source: "resource \"example\" \"test\" {\nouter {\nvalue=1\n}\n}\n",
			want:   "resource \"example\" \"test\" {\n    outer {\n        value = 1\n    }\n}\n",
		},
		{
			name:   "collections",
			source: "locals {\nitems = [\n{\nvalue=1\n},\n]\n}\n",
			want:   "locals {\n    items = [\n        {\n            value = 1\n        },\n    ]\n}\n",
		},
		{
			name:   "comments",
			source: "# Heading.\nlocals {\n// Keep this comment.\nvalue=1 # Inline.\n/* Keep this too. */\n}\n",
			want:   "# Heading.\nlocals {\n    // Keep this comment.\n    value = 1 # Inline.\n    /* Keep this too. */\n}\n",
		},
		{
			name:   "heredoc and template literals",
			source: "locals {\ntext=<<-EOT\n  literal   spaces\n    ${join(\",\", [\"a\", \"b\"])}\n%{ if true }\n  unchanged\n%{ endif }\nEOT\n\ntemplate=\"keep   ${join(\",\", [\"a\", \"b\"])}  spaces\"\n}\n",
			want:   "locals {\n    text = <<-EOT\n  literal   spaces\n    ${join(\",\", [\"a\", \"b\"])}\n%{if true}\n  unchanged\n%{endif}\nEOT\n\n    template = \"keep   ${join(\",\", [\"a\", \"b\"])}  spaces\"\n}\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := []byte(tc.source)
			original := bytes.Clone(input)
			got, err := Source(input, "example.tf")
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Fatalf("Unexpected formatting.\nGot %q\nWant %q", got, tc.want)
			}
			if !bytes.Equal(input, original) {
				t.Fatal("Source mutated its input.")
			}
			for _, next := range [][]byte{input, got} {
				again, err := Source(next, "example.tf")
				if err != nil || !bytes.Equal(again, got) {
					t.Fatalf("Formatting is not deterministic and idempotent. Got %q with error %v.", again, err)
				}
			}
		})
	}
}

func TestSourceEdgeCases(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
	}{
		{"empty", "", ""},
		{"CRLF", "locals {\r\nvalue=1\r\n}\r\n", "locals {\n    value = 1\n}"},
		{"no final newline", "locals {\nvalue=1\n}", "locals {\n    value = 1\n}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := []byte(tc.source)
			original := bytes.Clone(input)
			got, err := Source(input, "example.tf")
			if err != nil {
				t.Fatal(err)
			}
			// Compare structural whitespace without imposing a newline conversion policy.
			normalized := strings.TrimSuffix(strings.ReplaceAll(string(got), "\r\n", "\n"), "\n")
			if normalized != tc.want {
				t.Fatalf("Unexpected structural formatting %q.", got)
			}
			again, err := Source(got, "example.tf")
			if err != nil || !bytes.Equal(again, got) {
				t.Fatalf("Formatting is not idempotent. Got %q with error %v.", again, err)
			}
			if !bytes.Equal(input, original) {
				t.Fatal("Source mutated its input.")
			}
		})
	}
}

func TestSourcePreservesLiteralCRLF(t *testing.T) {
	literal := "  keep   these spaces\r\n    and these\r\n"
	input := []byte("locals {\r\ntext=<<EOT\r\n" + literal + "EOT\r\n}\r\n")
	original := bytes.Clone(input)
	got, err := Source(input, "example.tf")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte(literal)) || !bytes.Equal(input, original) {
		t.Fatalf("Literal bytes or input changed. Got %q.", got)
	}
}

func TestSourceInvalid(t *testing.T) {
	input := []byte("locals {\nvalue = [\n")
	original := bytes.Clone(input)
	got, err := Source(input, "broken.tf")
	if err == nil || got != nil {
		t.Fatalf("Expected a parse error and no output. Got %q with error %v.", got, err)
	}
	if !strings.Contains(err.Error(), "broken.tf") {
		t.Fatalf("Parse error omitted the filename. Got %v.", err)
	}
	if !bytes.Equal(input, original) {
		t.Fatal("Invalid input was mutated.")
	}
}

func FuzzSource(f *testing.F) {
	for _, seed := range []string{"", "value=1\n", "locals {\nitems=[1,2]\n}\n", "text=<<EOT\n  literal\nEOT\n", "value = \"${1 + 2}\"\n"} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		original := bytes.Clone(input)
		got, err := Source(input, "fuzz.tf")
		if !bytes.Equal(input, original) {
			t.Fatal("Source mutated its input.")
		}
		if err != nil {
			return
		}
		again, err := Source(got, "fuzz.tf")
		if err != nil || !bytes.Equal(again, got) {
			t.Fatalf("Valid input is not idempotent. Got %q then %q with error %v.", got, again, err)
		}
		repeated, err := Source(input, "fuzz.tf")
		if err != nil || !bytes.Equal(repeated, got) {
			t.Fatalf("Formatting is not deterministic. Got %q with error %v.", repeated, err)
		}
	})
}
