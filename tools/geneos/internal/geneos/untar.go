package geneos

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"slices"
	"strings"
	"time"
)

// untar the archive from an io.Reader onto host h in directory dir.
// Call stripPrefix for each file to remove configurable prefix
func untar(h *Host, dir string, tarfile io.Reader, filelen int64, stripPrefix func(string) string) (err error) {
	var name string

	tr := tar.NewReader(tarfile)
	label := "installing"
	if h != LOCAL {
		label += " on " + h.String()
	}
	bar, isterm := getbar(os.Stdout, label, filelen)

	defer func() {
		if !isterm {
			fmt.Println(bar.String())
		} else {
			bar.Finish()
			bar.Close()
		}
	}()

	var dirtimes []dirtime

	for {
		var hdr *tar.Header
		hdr, err = tr.Next()
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			return
		}

		// do not trust tar archives to contain safe paths

		name = hdr.Name
		if stripPrefix != nil {
			if name = stripPrefix(hdr.Name); name == "" {
				continue
			}
		}
		if name, err = CleanRelativePath(name); err != nil {
			return
		}
		fullpath := h.Join(dir, name)
		if err = rejectSymlinkPath(h, dir, name); err != nil {
			return
		}
		switch hdr.Typeflag {
		case tar.TypeReg:
			// check (and created) containing directories - account for munged tar files
			dir := h.Dir(fullpath)
			if err = h.MkdirAll(dir, 0775); err != nil {
				return
			}

			var out io.WriteCloser
			if out, err = h.Create(fullpath, hdr.FileInfo().Mode()); err != nil {
				return
			}
			var n int64
			n, err = io.Copy(out, tr)
			if err != nil {
				out.Close()
				return
			}
			bar.Add64(n)
			if n != hdr.Size {
				log.Error("lengths different", slog.Int64("expected", hdr.Size), slog.Int64("actual", n))
			}
			out.Close()
			if err := h.Chtimes(fullpath, hdr.AccessTime, hdr.ModTime); err != nil {
				log.Warn("cannot update mtime", slog.Any("error", err))
			}
		case tar.TypeDir:
			if err = h.MkdirAll(fullpath, hdr.FileInfo().Mode()); err != nil {
				return
			}
			dirtimes = append(dirtimes, dirtime{Name: fullpath, Modified: hdr.ModTime})

		case tar.TypeSymlink, tar.TypeGNULongLink:
			target := h.ToSlash(hdr.Linkname)
			if !h.IsAbs(hdr.Linkname) {
				target = strings.TrimSuffix(h.ToSlash(h.Dir(fullpath)), "/") + "/" + target
			}
			if err = rejectSymlinkPath(h, dir, target); err != nil {
				return
			}
			if _, err = h.Stat(fullpath); err != nil {
				if err = h.Symlink(hdr.Linkname, fullpath); err != nil {
					return
				}

			}
			if err := h.Lchtimes(fullpath, hdr.AccessTime, hdr.ModTime); h == LOCAL && err != nil {
				// ignore on remotes, sftp cannot set times on symlinks
				log.Debug("cannot update mtime (symlink?)", slog.Any("error", err))
			}

		default:
			log.Warn("unsupported file type", slog.String("type", string(hdr.Typeflag)))
		}

	}

	slices.Reverse(dirtimes)
	for _, d := range dirtimes {
		if err := h.Chtimes(d.Name, time.Time{}, d.Modified); err != nil {
			log.Warn("cannot update mtime", slog.String("file", d.Name), slog.Any("error", err))
		}
	}
	return
}

func rejectSymlinkPath(h *Host, root, relative string) error {
	root, err := h.Abs(root)
	if err != nil {
		return err
	}
	resolvedRoot, err := resolveArchivePath(h, root)
	if err != nil {
		return err
	}
	candidate := relative
	if !h.IsAbs(candidate) {
		candidate = h.Join(root, candidate)
	}
	resolvedCandidate, err := resolveArchivePath(h, candidate)
	if err != nil {
		return err
	}
	if !pathWithinRoot(h, resolvedRoot, resolvedCandidate) {
		return fmt.Errorf("path %q resolves outside extraction root %q", relative, resolvedRoot)
	}
	return nil
}

func resolveArchivePath(h *Host, name string) (string, error) {
	if !h.IsAbs(name) {
		var err error
		name, err = h.Abs(name)
		if err != nil {
			return "", err
		}
	}

	volume := h.VolumeName(name)
	filesystemRoot := "/"
	if volume != "" {
		filesystemRoot = strings.TrimRight(h.ToSlash(volume), "/") + "/"
	}
	remaining := h.ToSlash(name)
	if volume != "" {
		remaining = strings.TrimPrefix(remaining, h.ToSlash(volume))
	}
	components := strings.Split(strings.TrimLeft(remaining, "/"), "/")
	current := filesystemRoot
	symlinkCount := 0

	for len(components) > 0 {
		component := components[0]
		components = components[1:]
		switch component {
		case "", ".":
			continue
		case "..":
			if current != filesystemRoot {
				current = h.ToSlash(h.Dir(current))
			}
			continue
		}

		next := path.Join(current, component)
		info, err := h.Lstat(next)
		if errors.Is(err, fs.ErrNotExist) {
			current = next
			continue
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			current = next
			continue
		}

		symlinkCount++
		if symlinkCount > 255 {
			return "", fmt.Errorf("too many symlinks while resolving %q", name)
		}
		target, err := h.Readlink(next)
		if err != nil {
			return "", err
		}
		target = h.ToSlash(target)
		if h.IsAbs(target) {
			targetVolume := h.VolumeName(target)
			current = "/"
			if targetVolume != "" {
				current = strings.TrimRight(h.ToSlash(targetVolume), "/") + "/"
				target = strings.TrimPrefix(target, h.ToSlash(targetVolume))
			}
			target = strings.TrimLeft(target, "/")
		} else {
			current = h.ToSlash(h.Dir(next))
		}
		components = append(strings.Split(target, "/"), components...)
	}

	return path.Clean(current), nil
}

func pathWithinRoot(h *Host, root, candidate string) bool {
	root = path.Clean(h.ToSlash(root))
	candidate = path.Clean(h.ToSlash(candidate))
	if h.OS() == "windows" {
		root = strings.ToLower(root)
		candidate = strings.ToLower(candidate)
	}
	if candidate == root {
		return true
	}
	if root == "/" {
		return strings.HasPrefix(candidate, "/")
	}
	return strings.HasPrefix(candidate, strings.TrimSuffix(root, "/")+"/")
}
