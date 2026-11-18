package geneos

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
)

// defaultMaxArchiveSize is the default limit, in bytes, on the size of
// a zip archive that is copied to a temporary file to be unpacked. It
// is set by the setting `download.max-archive-size`, or the environment
// variable `ITRS_DOWNLOAD_MAX_ARCHIVE_SIZE`, where zero means no limit.
const defaultMaxArchiveSize int64 = 8 << 30

func maxArchiveSize() int64 {
	cf := config.Global()
	return config.Get[int64](cf, cf.Join("download", "max-archive-size"), config.DefaultValue(defaultMaxArchiveSize))
}

// zipSource returns a [io.ReaderAt] and the size for the zip archive
// read from archive, as zip files cannot be read from a stream. If
// archive can already be read at any offset, such as a file, it is used
// as it is and the cleanup function does nothing. Otherwise the archive
// is copied to a temporary file, which is created in dir, or if that
// fails in the default directory for temporary files, and the cleanup
// function closes and removes it. cleanup must always be called, even
// if there is an error. If limit is greater than zero and the archive
// is larger than that then an error is returned.
func zipSource(archive io.Reader, filesize int64, dir string, limit int64) (ra io.ReaderAt, size int64, cleanup func(), err error) {
	cleanup = func() {}

	switch a := archive.(type) {
	case *os.File:
		if st, err := a.Stat(); err == nil && st.Mode().IsRegular() {
			return a, st.Size(), cleanup, nil
		}
	case interface {
		io.ReaderAt
		io.Seeker
	}:
		cur, err := a.Seek(0, io.SeekCurrent)
		if err == nil {
			size, err = a.Seek(0, io.SeekEnd)
			if _, serr := a.Seek(cur, io.SeekStart); err == nil {
				err = serr
			}
		}
		if err == nil {
			return a, size, cleanup, nil
		}
	}

	if limit > 0 && filesize > limit {
		return nil, 0, cleanup, fmt.Errorf("archive is %d bytes, larger than the limit of %d; increase download.max-archive-size (ITRS_DOWNLOAD_MAX_ARCHIVE_SIZE)", filesize, limit)
	}

	tmp, err := os.CreateTemp(dir, ".unzip-*.tmp")
	if err != nil {
		if tmp, err = os.CreateTemp("", "geneos-unzip-*.tmp"); err != nil {
			return nil, 0, cleanup, err
		}
	}
	cleanup = func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}

	src := archive
	if limit > 0 {
		// read one byte more than is allowed to detect an archive that is too big
		src = io.LimitReader(archive, limit+1)
	}
	if size, err = io.Copy(tmp, src); err != nil {
		return nil, 0, cleanup, err
	}
	if limit > 0 && size > limit {
		return nil, 0, cleanup, fmt.Errorf("archive is larger than the limit of %d bytes; increase download.max-archive-size (ITRS_DOWNLOAD_MAX_ARCHIVE_SIZE)", limit)
	}
	return tmp, size, cleanup, nil
}

// unzip unpacks the zip archive from the reader to basedir on host h.
// An archive that is not a file is first copied to a temporary file, on
// the same volume as basedir if h is the local host, and removed
// afterwards, so that the whole archive is never held in memory.
func unzip(h *Host, basedir string, archive io.Reader, filesize int64, stripPrefix func(string) string) (err error) {
	return unzipLimit(h, basedir, archive, filesize, stripPrefix, maxArchiveSize())
}

func unzipLimit(h *Host, basedir string, archive io.Reader, filesize int64, stripPrefix func(string) string, limit int64) (err error) {
	var spoolDir string
	if h.IsLocalhost() {
		spoolDir = basedir
	}
	ra, size, cleanup, err := zipSource(archive, filesize, spoolDir, limit)
	defer cleanup()
	if err != nil {
		return err
	}
	z, err := zip.NewReader(ra, size)
	if err != nil {
		return err
	}

	label := "installing"
	if h != LOCAL {
		label += " on " + h.String()
	}
	bar, isterm := getbar(os.Stdout, label, size)

	defer func() {
		if !isterm {
			fmt.Println(bar.String())
		} else {
			bar.Finish()
			bar.Close()
		}
	}()

	var dirtimes []dirtime

	for _, f := range z.File {
		name := f.Name
		if stripPrefix != nil {
			if name = stripPrefix(f.Name); name == "" {
				continue
			}
		}
		name, err = CleanRelativePath(name)
		if err != nil {
			return err
		}
		if name == "." {
			continue
		}

		fullpath := h.Join(basedir, name)

		// dir
		if strings.HasSuffix(f.Name, "/") {
			if err = h.MkdirAll(fullpath, f.Mode()); err != nil {
				return err
			}
			// save the dir to update modified times later
			dirtimes = append(dirtimes, dirtime{Name: fullpath, Modified: f.Modified})
			continue
		}

		// symlinks are not created, and one that points outside of
		// basedir is an error
		if f.Mode()&fs.ModeSymlink != 0 {
			if err = checkZipSymlink(f, name); err != nil {
				return err
			}
			log.Warn("skipping symbolic link in zip archive", slog.String("name", f.Name))
			continue
		}

		var c io.ReadCloser
		c, err = f.Open()
		if err != nil {
			return err
		}

		dir := h.Dir(fullpath)
		var st fs.FileInfo
		if st, err = h.Stat(dir); errors.Is(err, fs.ErrNotExist) {
			if err = h.MkdirAll(dir, 0775); err != nil {
				c.Close()
				return
			}
		} else if err != nil {
			c.Close()
			return
		} else if !st.IsDir() {
			c.Close()
			return fmt.Errorf("%q is not a directory", dir)
		}

		var out io.WriteCloser
		if out, err = h.Create(fullpath, f.Mode()); err != nil {
			c.Close()
			return
		}

		var n int64
		n, err = io.CopyN(out, c, int64(f.UncompressedSize64))
		if err != nil {
			out.Close()
			c.Close()
			return
		}
		bar.Add64(int64(f.CompressedSize64))
		if n != int64(f.UncompressedSize64) {
			log.Error("lengths different", slog.Int64("expected", int64(f.UncompressedSize64)), slog.Int64("actual", n))
		}
		c.Close()
		if err = out.Close(); err != nil {
			return
		}

		if err := h.Chtimes(fullpath, time.Time{}, f.Modified); err != nil {
			log.Debug("cannot update mtime (symlink?)", slog.Any("error", err))
		}
	}

	slices.Reverse(dirtimes)
	for _, d := range dirtimes {
		log.Debug("updating mtime", slog.String("file", d.Name), slog.Any("modified", d.Modified))
		if err := h.Chtimes(d.Name, time.Time{}, d.Modified); err != nil {
			log.Warn("cannot update mtime", slog.String("file", d.Name), slog.Any("error", err))
		}
	}
	return
}

// checkZipSymlink returns an error if the symbolic link in the zip
// archive entry f, which is at name relative to the destination, is
// absolute or points outside of the destination.
func checkZipSymlink(f *zip.File, name string) error {
	c, err := f.Open()
	if err != nil {
		return err
	}
	defer c.Close()
	target, err := io.ReadAll(io.LimitReader(c, 4096))
	if err != nil {
		return err
	}
	link := string(target)
	if path.IsAbs(link) || strings.HasPrefix(link, "/") || filepath.VolumeName(link) != "" {
		return fmt.Errorf("symlink %q in archive points outside of the destination: %w", f.Name, ErrInvalidArgs)
	}
	if _, err := CleanRelativePath(path.Join(path.Dir(name), filepath.ToSlash(link))); err != nil {
		return fmt.Errorf("symlink %q in archive points outside of the destination: %w", f.Name, ErrInvalidArgs)
	}
	return nil
}
