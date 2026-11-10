package format

import "testing"

func TestDiffFinalNewlineChange(t *testing.T) {
	got, err := Diff("main.tf", []byte("a = 1"), []byte("a = 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := "--- old/main.tf\n+++ new/main.tf\n@@ -1 +1 @@\n-a = 1\n\\ No newline at end of file\n+a = 1\n"
	if got != want {
		t.Fatalf("Unexpected newline diff.\nGot %q\nWant %q", got, want)
	}
}

func TestSourceFinalNewlineContract(t *testing.T) {
	for _, input := range []string{"a=1", "a=1 ", "a=1\n", "a=1\n\n"} {
		got, err := Source([]byte(input), "main.tf")
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "a = 1\n" {
			t.Fatalf("Expected one final newline, got %q.", got)
		}
	}
}
