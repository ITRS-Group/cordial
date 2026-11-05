package includes

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

const (
	IncludePattern = "files/itrs.*.xml"
	ConfigPattern  = "files/config.*.xml"
)

func extractIncludes(h *geneos.Host, sourceFS fs.FS, version string, override bool) (err error) {
	ct := geneos.ParseComponent("gateway")
	resp := responses.New[responses.General](nil)
	fmt.Println("Extracting include files for version", version)
	dir := h.PathTo(ct, "includes", version)
	st, statErr := h.Lstat(dir)
	if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		return statErr
	}
	if st != nil && !st.IsDir() {
		return fmt.Errorf("include version path %q is not a directory", dir)
	}
	if st != nil && !override {
		return fmt.Errorf("includes directory already exists at %s and replacement is disabled", dir)
	}
	if err = h.MkdirAll(dir, 0755); err != nil {
		return err
	}
	incs, err := fs.Glob(sourceFS, IncludePattern)
	if err != nil {
		resp.Err = err
		return err
	}
	if len(incs) == 0 {
		fmt.Println("No include files found matching:", IncludePattern)
		resp.Err = fmt.Errorf("no include files found matching: %s", IncludePattern)
		return nil
	}
	for _, inc := range incs {
		efi, err := fs.Stat(sourceFS, inc)
		if err != nil {
			return fmt.Errorf("failed to inspect embedded include %q: %w", inc, err)
		}
		p := h.Join(dir, filepath.Base(inc))
		fi, err := h.Lstat(p)
		if err == nil {
			if fi.Mode().IsRegular() && efi.Size() == fi.Size() {
				f1h, err := sourceFS.Open(inc)
				if err != nil {
					return fmt.Errorf("failed to open embedded include %q: %w", inc, err)
				}
				f2h, err := h.Open(p)
				if err != nil {
					f1h.Close()
					return fmt.Errorf("failed to open existing include %q: %w", p, err)
				}
				same := compareContents(f1h, f2h)
				f1h.Close()
				f2h.Close()
				if same {
					fmt.Println("Skipping existing unchanged include file", p)
					continue
				}
			}
			ts := time.Now().Format("20060102150405")
			backupPath := fmt.Sprintf("%s.%s.bak", p, ts)
			if err := h.Rename(p, backupPath); err != nil {
				return err
			}
			fmt.Println("Backed up existing include file to", backupPath)
			resp.Completed = append(resp.Completed, fmt.Sprintf("backed up %s", p))
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to inspect include path %q: %w", p, err)
		}
		if err := copyIncludeFile(sourceFS, inc, h, p, 0444); err != nil {
			return fmt.Errorf("failed to extract include %q: %w", inc, err)
		}
		fmt.Println("Extracted include file", p)
		resp.Completed = append(resp.Completed, fmt.Sprintf("extracted %s", p))
	}
	dir = h.PathTo(ct, "includes", "config")
	if err = h.MkdirAll(dir, 0755); err != nil {
		return err
	}
	confs, err := fs.Glob(sourceFS, ConfigPattern)
	if err != nil {
		return err
	}
	for _, conf := range confs {
		p := h.Join(dir, filepath.Base(conf))
		_, err := h.Lstat(p)
		if err == nil {
			log.Debug("Config file already exists", slog.String("file", p))
		} else if errors.Is(err, fs.ErrNotExist) {
			if err := copyIncludeFile(sourceFS, conf, h, p, 0664); err != nil {
				return fmt.Errorf("failed to extract config include %q: %w", conf, err)
			}
		} else {
			return fmt.Errorf("failed to inspect config include path %q: %w", p, err)
		}
	}
	return nil
}

func copyIncludeFile(sourceFS fs.FS, source string, h *geneos.Host, destination string, mode fs.FileMode) (err error) {
	src, err := sourceFS.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := h.Create(destination, mode)
	if err != nil {
		return err
	}
	if _, err = io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}

const chunkSize = 4096

func compareContents(f1h, f2h io.Reader) bool {
	buf1 := make([]byte, chunkSize)
	buf2 := make([]byte, chunkSize)
	for {
		n1, err1 := f1h.Read(buf1)
		n2, err2 := f2h.Read(buf2)
		if n1 != n2 || err1 != nil && err1 != io.EOF || err2 != nil && err2 != io.EOF {
			return false
		}
		if n1 == 0 {
			break
		}
		if !bytes.Equal(buf1[:n1], buf2[:n2]) {
			return false
		}
	}
	return true
}
