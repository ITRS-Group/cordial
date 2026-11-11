/*
Copyright © 2022 ITRS Group

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
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
)

// ImportSource copies the contents from item to the destination
// directory dir on host h. The destination filename can be given as a
// "NAME=" prefix in item. If no filename is given then it is derived
// from the base filename of item.
//
// item, without any destination prefix, can be a path to a file, a
// directory or a http/https URL.
//
// If item points to a local file and it is the same as the destination
// then an ErrExists is returned.
//
// If item points to a local directory then it is copied to the
// destination only if the destination is a directory or does not exist
// and a directory with that name can be created.
//
// If item ends with a '/' then the contents are copied without the
// directory prefix, otherwise the final directory component in source
// is also copied, e.g. 'conf/' means import the contents of the
// directory 'conf', which 'conf' means a new directory 'conf' is
// created in dest and the contents copied.
//
// TODO: add templating option - perhaps ImportFileTemplated() - to run
// through text/template with given data (e.g. instance config)
func ImportSource(h *Host, dir, item string) (filename, source string, err error) {
	// host must be valid and not `ALL`
	if h == nil || h == ALL {
		err = ErrInvalidArgs
		log.Debug("host invalid, skipping", slog.Any("error", err))
		return
	}

	// check item is not empty
	if item == "" {
		err = ErrInvalidArgs
		log.Debug("no source defined, skipping", slog.Any("error", err))
		return
	}

	// check dir is not empty and actually a directory
	if dir == "" {
		err = ErrInvalidArgs
		log.Debug("dir is empty, skipping", slog.Any("error", err))
		return
	}

	if st, err := h.Stat(dir); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.Debug("dir check failed", slog.Any("error", err))
			return "", "", err
		}

		// try and create it
		if err = h.MkdirAll(dir, 0775); err != nil {
			log.Debug("failed to create dir, skipping", slog.String("dir", dir), slog.Any("error", err))
			return "", "", err
		}
	} else {
		if !st.IsDir() {
			err = ErrInvalidArgs
			log.Debug("dir is not a directory, skipping", slog.String("dir", dir), slog.Any("error", err))
			return "", "", err
		}
	}

	// split out any "NAME=" prefix
	var dstIsDir bool
	dst, src, ok := strings.Cut(item, "=")
	if ok {
		// do some basic validation on user-supplied destination
		if dst == "" {
			err = ErrInvalidArgs
			log.Debug("dest path empty", slog.Any("error", err))
			return
		}
		// if the dest contains "://" then it is probably part of
		// the URL and not a prefix, so ignore it - restore values
		if strings.Contains(dst, "://") {
			dst = ""
		} else if strings.HasSuffix(dst, "/") {
			// create a directory
			dstIsDir = true
			if dst, err = CleanRelativePath(dst); err != nil {
				return "", "", fmt.Errorf("destination must be relative: %w", err)
			}
		} else if dst, err = CleanRelativePath(dst); err != nil {
			return "", "", fmt.Errorf("destination must be relative: %w", err)
		}
		item = src
	} else {
		dst = ""
	}

	// if the destination exists, is it a directory?
	if s, err := h.Stat(h.Join(dir, dst)); err == nil {
		if s.IsDir() {
			dir = h.Join(dir, dst)
			dst = ""
			dstIsDir = false
		}
	}

	if dstIsDir {
		dir = h.Join(dir, dst)
		dst = ""
	}

	log.Debug("importing source", slog.String("item", item), slog.String("dir", dir), slog.String("dest", dst))

	// set the Source(item) option so that any optional credentials are loaded
	from, filename, _, err := openSource(item, Source(item))
	if err != nil {
		// special case the source is a directory
		if errors.Is(err, ErrIsADirectory) {
			err = nil

			// import directory
			err = importDirectory(h, dir, item)
			return
		}
		return "", item, fmt.Errorf("open import source %q: %w", item, err)
	}
	defer from.Close()

	log.Debug("opened source", slog.String("item", item), slog.String("filename", filename))

	// only use the returned filename if no explicit destination is given
	if dst == "" {
		dst = filename
	}

	// return final basename
	dst = h.Join(dir, dst)
	filename = h.Base(dst)

	// test for same source and dest, return err. only on localhost.
	if h.IsLocalhost() && !strings.Contains(item, "://") {
		item = config.ResolveHome(item)
		sfi, err := h.Stat(item)
		if err != nil {
			return "", "", err
		}
		dfi, err := h.Stat(dst)
		if err == nil {
			if os.SameFile(sfi, dfi) {
				// same
				fmt.Printf("import skipped, source and destination are the same file: %q\n", item)
				return filename, "", ErrExists
			}
		}
	}

	source = item
	changed, err := importFile(h, dst, from, 0664)
	if err != nil {
		return filename, source, err
	}
	if changed {
		fmt.Printf("imported %q to %s:%s\n", item, h.String(), dst)
	}

	return filename, source, nil
}

var importFileLocks sync.Map

func importFile(h *Host, destination string, from io.Reader, perms fs.FileMode) (changed bool, err error) {
	if err := h.MkdirAll(h.Dir(destination), 0775); err != nil {
		return false, err
	}
	mutex, _ := importFileLocks.LoadOrStore(h.String()+":"+h.Clean(destination), &sync.Mutex{})
	lock := mutex.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	lockPath := destination + ".import.lock"
	lockFile, err := h.GetFs().OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return false, fmt.Errorf("lock import destination %q: %w", destination, err)
	}
	defer h.Remove(lockPath)
	if err := lockFile.Close(); err != nil {
		return false, err
	}

	stagedPath := destination + "." + rand.Text() + ".new"
	staged, err := h.GetFs().OpenFile(stagedPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perms)
	if err != nil {
		return false, err
	}
	defer h.Remove(stagedPath)
	digest := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(staged, digest), from)
	closeErr := staged.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return false, err
	}

	var backup string
	if info, err := h.Stat(destination); err == nil {
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("import destination %q is not a regular file", destination)
		}
		if info.Size() == written {
			current, err := h.Open(destination)
			if err != nil {
				return false, err
			}
			existingDigest := sha256.New()
			_, copyErr := io.Copy(existingDigest, current)
			closeErr := current.Close()
			if err := errors.Join(copyErr, closeErr); err != nil {
				return false, err
			}
			if bytes.Equal(digest.Sum(nil), existingDigest.Sum(nil)) {
				return false, nil
			}
		}
		backup, err = reserveImportBackup(h, destination, time.Now())
		if err != nil {
			return false, err
		}
		if err := h.Rename(destination, backup); err != nil {
			_ = h.Remove(backup)
			return false, err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	if err := h.Rename(stagedPath, destination); err != nil {
		if backup != "" {
			return false, errors.Join(err, h.Rename(backup, destination))
		}
		return false, err
	}
	return true, nil
}

func reserveImportBackup(h *Host, destination string, now time.Time) (string, error) {
	prefix := destination + "." + now.UTC().Format("20060102150405")
	for sequence := 0; ; sequence++ {
		candidate := prefix + ".old"
		if sequence > 0 {
			candidate = fmt.Sprintf("%s.%d.old", prefix, sequence)
		}
		reserved, err := h.GetFs().OpenFile(candidate, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if err := reserved.Close(); err != nil {
			_ = h.Remove(candidate)
			return "", err
		}
		return candidate, nil
	}
}

// import directory from local source directory to dir on host h
func importDirectory(h *Host, dir, source string) (err error) {
	// check dest is a directory
	if dir == "" {
		err = ErrInvalidArgs
		log.Debug("source is a directory, dir is empty, skipping", slog.String("src", source), slog.Any("error", err))
		return
	}

	// if source ends with '/' then copy contents only, not the directory named
	if !strings.HasSuffix(source, "/") {
		dir = h.Join(dir, filepath.Base(source))
	}

	return fs.WalkDir(os.DirFS(source), ".", func(file string, di fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		destfile := h.Join(dir, file)

		st, err := di.Info()
		if err != nil {
			return err
		}

		if di.IsDir() {
			return h.MkdirAll(destfile, st.Mode().Perm())
		}

		from, err := os.Open(filepath.Join(source, file))
		if err != nil {
			return err
		}
		defer from.Close()
		changed, err := importFile(h, destfile, from, st.Mode().Perm())
		if err != nil {
			return err
		}
		if changed {
			fmt.Printf("imported %q to %s:%s\n", filepath.Join(source, file), h.String(), destfile)
		}
		return nil
	})
}

// ImportCommons copies a file to an instance common directory.
func ImportCommons(h *Host, ct *Component, common string, params []string) (filenames []string, err error) {
	if ct == nil || ct == &RootComponent {
		err = ErrNotSupported
		return
	}

	if len(params) == 0 {
		log.Error("no file/url provided")
		os.Exit(1)
	}

	dir := h.PathTo(ct, common)
	for _, source := range params {
		var filename string
		if filename, _, err = ImportSource(h, dir, source); err != nil && err != ErrExists {
			return
		}
		filenames = append(filenames, filename)
	}
	err = nil // reset in case above returns ErrExists
	return
}
