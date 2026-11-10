package format

import "testing"

func TestDiff(t *testing.T) {
	before := []byte("value=1\n")
	after := []byte("value = 1\n")
	got, err := Diff("main.tf", before, after)
	if err != nil {
		t.Fatal(err)
	}
	want := "--- old/main.tf\n+++ new/main.tf\n@@ -1 +1 @@\n-value=1\n+value = 1\n"
	if got != want {
		t.Fatalf("Unexpected unified diff.\nGot %q\nWant %q", got, want)
	}
}

func TestDiffUnchanged(t *testing.T) {
	for _, source := range [][]byte{nil, []byte("value = 1\n")} {
		got, err := Diff("main.tf", source, source)
		if err != nil || got != "" {
			t.Fatalf("Expected no diff. Got %q with error %v.", got, err)
		}
	}
}
