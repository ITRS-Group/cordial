package includes

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

const (
	IncludePattern = "files/itrs.*.xml"
	ConfigPattern  = "files/config.*.xml"
)

func extractIncludes(h *geneos.Host, sourceFS fs.FS, version string, override bool) (err error) {
	if err := geneos.ValidateSite(config.Get[string](config.Global(), "site")); err != nil {
		return err
	}
	site := geneos.SiteName()
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
	dir = h.PathTo(ct, site)
	st, statErr = h.Lstat(dir)
	if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		return fmt.Errorf("inspect site directory %q: %w", dir, statErr)
	}
	if st != nil && !st.IsDir() {
		return fmt.Errorf("site path %q is not a directory", dir)
	}
	if err = h.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err = copyLegacyConfig(h, h.PathTo(ct, "includes", "config"), dir, site); err != nil {
		return err
	}
	confs, err := fs.Glob(sourceFS, ConfigPattern)
	if err != nil {
		return err
	}
	for _, conf := range confs {
		p := h.Join(dir, site+strings.TrimPrefix(filepath.Base(conf), "config"))
		_, err := h.Lstat(p)
		if err == nil {
			log.Debug("Config file already exists", slog.String("file", p))
		} else if errors.Is(err, fs.ErrNotExist) {
			if err := copyIncludeFile(sourceFS, conf, h, p, 0664); err != nil {
				if errors.Is(err, fs.ErrExist) {
					continue
				}
				return fmt.Errorf("failed to extract config include %q: %w", conf, err)
			}
		} else {
			return fmt.Errorf("failed to inspect config include path %q: %w", p, err)
		}
	}
	return nil
}

func copyLegacyConfig(h *geneos.Host, source, destination, site string) error {
	info, err := h.Lstat(source)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect old config directory %q: %w", source, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("old config path %q is not a directory", source)
	}
	err = h.WalkDir(source, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		targetName := h.Base(name)
		if !entry.IsDir() && strings.HasPrefix(targetName, "config.") {
			targetName = site + strings.TrimPrefix(targetName, "config")
		}
		target := h.Join(destination, h.Dir(name), targetName)
		existing, err := h.Lstat(target)
		if err == nil {
			if entry.IsDir() && !existing.IsDir() {
				return fmt.Errorf("config destination %q is not a directory", target)
			}
			return nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			return h.MkdirAll(target, info.Mode().Perm())
		}
		sourcePath := h.Join(source, name)
		info, err := h.Stat(sourcePath)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("config source %q is not a regular file", sourcePath)
		}
		input, err := h.Open(sourcePath)
		if err != nil {
			return err
		}
		defer input.Close()
		err = copyToNewIncludeFile(h, target, input, info.Mode().Perm())
		if errors.Is(err, fs.ErrExist) {
			return nil
		}
		return err
	})
	if err != nil {
		return fmt.Errorf("copy old config directory %q: %w", source, err)
	}
	return nil
}

func copyToNewIncludeFile(h *geneos.Host, name string, source io.Reader, mode fs.FileMode) error {
	filesystem := h.GetFs()
	if filesystem == nil {
		return fmt.Errorf("cannot access filesystem on %s", h)
	}
	output, err := filesystem.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, source)
	closeErr := output.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return errors.Join(err, h.Remove(name))
	}
	return nil
}

func copyIncludeFile(sourceFS fs.FS, source string, h *geneos.Host, destination string, mode fs.FileMode) (err error) {
	src, err := sourceFS.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	return copyToNewIncludeFile(h, destination, src, mode)
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
