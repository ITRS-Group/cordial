package geneos

import (
	"net/url"
	"os"
	"path/filepath"
)

// IsFile checks if the given path is a regular file.
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		// If it's a symlink, we need to resolve it first
		resolvedPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return false
		}
		info, err = os.Stat(resolvedPath)
		if err != nil {
			return false
		}
	}
	return info.Mode().IsRegular()
}

// IsDir checks if the given path is a directory.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		// If it's a symlink, we need to resolve it first
		resolvedPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return false
		}
		info, err = os.Stat(resolvedPath)
		if err != nil {
			return false
		}
	}
	return info.IsDir()
}

// IsURL checks if the given string is a valid URL.
func IsURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	return err == nil && u.Scheme != "" && u.Host != ""
}
