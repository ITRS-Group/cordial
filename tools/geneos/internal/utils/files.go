package utils

import "path/filepath"

// JoinSlash returns a file path joined but with slashes (to support Windows
// looking for remote paths on Linux)
func JoinSlash(elem ...string) string {
	return filepath.ToSlash(filepath.Join(elem...))
}

func Dir(path string) string {
	return filepath.ToSlash(filepath.Dir(path))
}
