package geneos

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"time"

	"github.com/rs/zerolog/log"
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
		fullpath := path.Join(dir, name)
		switch hdr.Typeflag {
		case tar.TypeReg:
			// check (and created) containing directories - account for munged tar files
			dir := path.Dir(fullpath)
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
				log.Error().Msgf("lengths different: %d %d", hdr.Size, n)
			}
			out.Close()
			if err := h.Chtimes(fullpath, hdr.AccessTime, hdr.ModTime); err != nil {
				log.Warn().Err(err).Msg("cannot update mtime")
			}
		case tar.TypeDir:
			if err = h.MkdirAll(fullpath, hdr.FileInfo().Mode()); err != nil {
				return
			}
			dirtimes = append(dirtimes, dirtime{Name: fullpath, Modified: hdr.ModTime})

		case tar.TypeSymlink, tar.TypeGNULongLink:
			if path.IsAbs(hdr.Linkname) {
				err = fmt.Errorf("archive contains absolute symlink target")
				return
			}
			if _, err = h.Stat(fullpath); err != nil {
				if err = h.Symlink(hdr.Linkname, fullpath); err != nil {
					return
				}

			}
			if err := h.Lchtimes(fullpath, hdr.AccessTime, hdr.ModTime); h == LOCAL && err != nil {
				// ignore on remotes, sftp cannot set times on symlinks
				log.Debug().Err(err).Msg("cannot update mtime (symlink?)")
			}

		default:
			log.Warn().Msgf("unsupported file type %c\n", hdr.Typeflag)
		}

	}

	slices.Reverse(dirtimes)
	for _, d := range dirtimes {
		if err := h.Chtimes(d.Name, time.Time{}, d.Modified); err != nil {
			log.Warn().Err(err).Msgf("cannot update mtime on %q", d.Name)
		}
	}
	return
}
