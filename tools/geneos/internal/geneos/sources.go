/*
Copyright © 2026 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
