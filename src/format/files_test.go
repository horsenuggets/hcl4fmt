package format

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"
)

func TestSupported(t *testing.T) {
	for _, name := range []string{"main.tf", "values.tfvars", "example.tftest.hcl", "example.tfmock.hcl", "example.tfquery.hcl", ".terraform.lock.hcl", "other.hcl", "other.lock.hcl", "packer.pkr.hcl"} {
		if !Supported(filepath.Join("nested", name)) {
			t.Errorf("Expected support for %q.", name)
		}
	}
	for _, name := range []string{"main.tf.json", "values.tfvars.json", "config.hcl.json", ".terraform.lock.hcl.bak", "main.HCL", "main.TF", "main.tf.bak", "README"} {
		if Supported(name) {
			t.Errorf("Unexpected support for %q.", name)
		}
	}
}

func TestDiscover(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"nested", ".hidden", "~backup", "backup~", "#backup#"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	names := []string{"z.tf", "a.tfvars", "b.tftest.hcl", "c.tfmock.hcl", "d.tfquery.hcl", ".terraform.lock.hcl", "nested/child.tf", ".hidden/skip.tf", ".hidden/.terraform.lock.hcl", "~backup/skip.tf", "backup~/skip.tf", "#backup#/skip.tf", ".skip.tf", "~skip.tf", "skip.tf~", "#skip.tf#", "skip.tf.json", "generic.hcl", ".skip.hcl", ".hidden/skip.hcl"}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{filepath.Join(root, ".terraform.lock.hcl"), filepath.Join(root, "a.tfvars"), filepath.Join(root, "b.tftest.hcl"), filepath.Join(root, "c.tfmock.hcl"), filepath.Join(root, "d.tfquery.hcl"), filepath.Join(root, "generic.hcl"), filepath.Join(root, "z.tf")}
	for _, recursive := range []bool{false, true} {
		t.Run(map[bool]string{false: "shallow", true: "recursive"}[recursive], func(t *testing.T) {
			expected := append([]string(nil), want...)
			if recursive {
				expected = append(expected, filepath.Join(root, "nested", "child.tf"))
			}
			sort.Strings(expected)
			// Overlapping directories, duplicate files, and cleaned aliases must deduplicate.
			paths := []string{filepath.Join(root, "z.tf"), root, root, root + string(filepath.Separator) + "." + string(filepath.Separator) + "z.tf"}
			got, err := Discover(paths, recursive)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, expected) {
				t.Fatalf("Unexpected discovery. Got %q, want %q.", got, expected)
			}
		})
	}
}

func TestDiscoverSymlinks(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "target.tf")
	if err := os.WriteFile(target, []byte("value = 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.tf")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("Symbolic links are unavailable. %v", err)
	}
	dirLink := filepath.Join(root, "linked-directory")
	if err := os.Symlink(filepath.Dir(target), dirLink); err != nil {
		t.Fatal(err)
	}
	separator := string(filepath.Separator)
	for _, path := range []string{link, dirLink, dirLink + separator, dirLink + separator + "."} {
		if _, err := Discover([]string{path}, true); err == nil {
			t.Errorf("Expected rejection of explicit symbolic link %q.", path)
		}
	}
	got, err := Discover([]string{root}, true)
	if err != nil || len(got) != 0 {
		t.Fatalf("Directory traversal followed a symbolic link. Got %q with error %v.", got, err)
	}
}

func TestDiscoverInvalidPaths(t *testing.T) {
	root := t.TempDir()
	unsupported := filepath.Join(root, "main.tf.json")
	if err := os.WriteFile(unsupported, nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(root, "missing.tf"), unsupported} {
		if _, err := Discover([]string{path}, false); err == nil {
			t.Errorf("Expected rejection of %q.", path)
		}
	}
}

func TestWrite(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.tf")
	if err := os.WriteFile(path, []byte("value=1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0640); err != nil {
		t.Fatal(err)
	}
	want := []byte("value = 1\n")
	if err := Write(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(want) {
		t.Fatalf("Unexpected saved content %q with error %v.", got, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0640 {
		t.Fatalf("Permissions changed to %v.", info.Mode().Perm())
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != "main.tf" {
		t.Fatalf("Temporary files were left behind. Got %v with error %v.", entries, err)
	}
}

func TestWriteFailureCleanup(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{root, filepath.Join(root, "missing.tf")} {
		if err := Write(path, []byte("value = 1\n")); err == nil {
			t.Errorf("Expected replacement failure for %q.", path)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("Failed writes left temporary files. Got %v with error %v.", entries, err)
	}
}

func TestWriteRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.tf")
	original := "value=1\n"
	if err := os.WriteFile(target, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.tf")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("Symbolic links are unavailable. %v", err)
	}
	if err := Write(link, []byte("value = 2\n")); err == nil {
		t.Fatal("Expected rejection of a symbolic link.")
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != original {
		t.Fatalf("Symbolic link target changed. Got %q with error %v.", got, err)
	}
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("Symbolic link was replaced. Got %v with error %v.", info, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 2 {
		t.Fatalf("Rejected write left temporary files. Got %v with error %v.", entries, err)
	}
}
