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

package instance

import (
	"fmt"
	"io/fs"
	"log/slog"
	"strings"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// ImportTemplate imports the template file or URL into the templates
// directory for the type of instance i and returns the resulting
// filename. An error is returned if nothing could be imported, so that
// callers never have to deal with an empty list of filenames.
func ImportTemplate(i geneos.Instance, template string) (filename string, err error) {
	filenames, err := geneos.ImportCommons(i.Host(), i.Type(), "templates", []string{template})
	if err != nil {
		return "", fmt.Errorf("import template %q: %w", template, err)
	}
	if len(filenames) == 0 || filenames[0] == "" {
		return "", fmt.Errorf("import template %q: %w", template, geneos.ErrInvalidArgs)
	}
	return filenames[0], nil
}

// ImportPackageFiles copies files from the directory srcDir, which is
// on the same host as instance i, into the home directory of i. This is
// used to seed new instances from their (read-only) installed package
// and, unlike [ImportFiles], does not depend on the process working
// directory and works for remote hosts.
//
// Each item is in the form `[DEST=]SOURCE` where SOURCE is relative to
// srcDir and can be a file or a directory, which is copied recursively.
// DEST is relative to the instance home and defaults to the base name
// of SOURCE. Existing destination files are overwritten.
func ImportPackageFiles(i geneos.Instance, srcDir string, items ...string) (err error) {
	h := i.Host()
	home := i.Home()

	for _, item := range items {
		dst, src, found := strings.Cut(item, "=")
		if !found {
			src, dst = dst, ""
		}
		src = h.Join(srcDir, src)
		if dst == "" {
			dst = h.Base(src)
		}
		if dst, err = geneos.CleanRelativePath(dst); err != nil {
			return fmt.Errorf("import %q: destination must be relative: %w", item, err)
		}
		if err = copyPackagePath(h, src, h.Join(home, dst)); err != nil {
			return fmt.Errorf("import %q: %w", item, err)
		}
	}
	return nil
}

func copyPackagePath(h *geneos.Host, src, dst string) error {
	st, err := h.Stat(src)
	if err != nil {
		return err
	}

	if !st.IsDir() {
		return copyPackageFile(h, src, dst, st.Mode().Perm())
	}

	if err = h.MkdirAll(dst, 0775); err != nil {
		return err
	}
	entries, err := h.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err = copyPackagePath(h, h.Join(src, entry.Name()), h.Join(dst, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyPackageFile(h *geneos.Host, src, dst string, perm fs.FileMode) error {
	data, err := h.ReadFile(src)
	if err != nil {
		return err
	}
	if err = h.MkdirAll(h.Dir(dst), 0775); err != nil {
		return err
	}
	log.Debug("importing package file", slog.String("source", src), slog.String("destination", dst))
	return h.WriteFile(dst, data, perm|0200)
}
