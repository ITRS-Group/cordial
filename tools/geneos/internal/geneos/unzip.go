package geneos

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	zlog "github.com/rs/zerolog/log"
)

func unzip(h *Host, basedir string, archive io.Reader, filesize int64, stripPrefix func(string) string) (err error) {
	// zip files are read into memory for now
	var z *zip.Reader
	var r []byte
	r, err = io.ReadAll(archive)
	if err != nil {
		panic(err)
	}
	b := bytes.NewReader(r)
	z, err = zip.NewReader(b, int64(len(r)))

	label := "installing"
	if h != LOCAL {
		label += " on " + h.String()
	}
	bar, isterm := getbar(os.Stdout, label, int64(len(r)))

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
			panic(err)
		}
		if name == "." {
			continue
		}

		fullpath := path.Join(basedir, name)

		// dir
		if strings.HasSuffix(f.Name, "/") {
			if err = h.MkdirAll(fullpath, f.Mode()); err != nil {
				panic(err)
			}
			// save the dir to update modified times later
			dirtimes = append(dirtimes, dirtime{Name: fullpath, Modified: f.Modified})
			continue
		}

		var c io.ReadCloser
		c, err = f.Open()
		if err != nil {
			panic(err)
		}

		dir := path.Dir(fullpath)
		var st fs.FileInfo
		if st, err = h.Stat(dir); errors.Is(err, fs.ErrNotExist) {
			if err = h.MkdirAll(dir, 0775); err != nil {
				c.Close()
				return
			}
		} else if !st.IsDir() {
			panic(err)
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
			zlog.Error().Msgf("lengths different: %d %d", f.UncompressedSize64, n)
		}
		out.Close()
		c.Close()

		if err := h.Chtimes(fullpath, time.Time{}, f.Modified); err != nil {
			zlog.Debug().Err(err).Msg("cannot update mtime (symlink?)")
		}
	}

	slices.Reverse(dirtimes)
	for _, d := range dirtimes {
		zlog.Debug().Msgf("updating %q to %v", d.Name, d.Modified)
		if err := h.Chtimes(d.Name, time.Time{}, d.Modified); err != nil {
			zlog.Warn().Err(err).Msgf("cannot update mtime on %q", d.Name)
		}
	}
	return
}
