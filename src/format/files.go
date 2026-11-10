package format

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Supported reports whether a file has a native Terraform HCL suffix.
func Supported(path string) bool {
	for _, suffix := range []string{".tf", ".tfvars", ".tftest.hcl", ".tfmock.hcl", ".tfquery.hcl"} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

func ignored(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "~") ||
		strings.HasSuffix(name, "~") || (strings.HasPrefix(name, "#") && strings.HasSuffix(name, "#"))
}

// Discover returns sorted, unique files without following symbolic links.
func Discover(paths []string, recursive bool) ([]string, error) {
	found := make(map[string]string)
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("cannot format symbolic link %q", path)
		}
		if !info.IsDir() {
			if !info.Mode().IsRegular() || !Supported(path) {
				return nil, fmt.Errorf("unsupported file %q", path)
			}
			absolute, err := filepath.Abs(path)
			if err != nil {
				return nil, err
			}
			found[absolute] = filepath.Clean(path)
			continue
		}
		err = filepath.WalkDir(path, func(current string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if current == path {
				return nil
			}
			if entry.IsDir() {
				if !recursive || ignored(entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if ignored(entry.Name()) || !entry.Type().IsRegular() || !Supported(current) {
				return nil
			}
			absolute, err := filepath.Abs(current)
			if err != nil {
				return err
			}
			if _, exists := found[absolute]; !exists {
				found[absolute] = current
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	files := make([]string, 0, len(found))
	for _, path := range found {
		files = append(files, path)
	}
	sort.Strings(files)
	return files, nil
}

// Write atomically replaces a regular file, retaining its permission bits.
func Write(path string, source []byte) (err error) {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("cannot replace non-regular file %q", path)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".tf4fmt-*")
	if err != nil {
		return err
	}
	defer func() {
		removeErr := os.Remove(temporary.Name())
		if err == nil && removeErr != nil && !os.IsNotExist(removeErr) {
			err = removeErr
		}
	}()
	if _, err = temporary.Write(source); err != nil {
		_ = temporary.Close()
		return err
	}
	if err = temporary.Chmod(info.Mode().Perm()); err != nil {
		_ = temporary.Close()
		return err
	}
	if err = temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), path)
}
