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
)

const (
	IncludePattern = "files/itrs.*.xml"
	ConfigPattern  = "files/config.*.xml"
)

func extractIncludes(h *geneos.Host, sourceFS fs.FS, version string, override bool) (err error) {
	return extractIncludeFiles(h, sourceFS, version, override, false)
}

func extractIncludeFiles(h *geneos.Host, sourceFS fs.FS, version string, override, missingOnly bool) (err error) {
	if err := geneos.ValidateSite(config.Get[string](config.Global(), "site")); err != nil {
		return err
	}
	site := geneos.SiteName()
	ct := geneos.ParseComponent("gateway")
	fmt.Println("Extracting include files for version", version)
	dir := h.PathTo(ct, "includes", version)
	st, statErr := h.Lstat(dir)
	if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		return statErr
	}
	if st != nil && !st.IsDir() {
		return fmt.Errorf("include version path %q is not a directory", dir)
	}
	if st != nil && !override && !missingOnly {
		return fmt.Errorf("includes directory already exists at %s and replacement is disabled", dir)
	}
	pattern, err := includePattern(version)
	if err != nil {
		return err
	}
	incs, err := fs.Glob(sourceFS, pattern)
	if err != nil {
		return err
	}
	if len(incs) == 0 {
		return fmt.Errorf("no include files found matching: %s", pattern)
	}
	if err = h.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, inc := range incs {
		efi, err := fs.Stat(sourceFS, inc)
		if err != nil {
			return fmt.Errorf("failed to inspect embedded include %q: %w", inc, err)
		}
		p := h.Join(dir, filepath.Base(inc))
		fi, err := h.Lstat(p)
		if err == nil {
			if missingOnly {
				fmt.Println("Skipping existing include file", p)
				continue
			}
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
			err := h.Rename(p, backupPath)
			auditIncludeFile(h, "backup", backupPath, err, slog.String("source", p))
			if err != nil {
				return err
			}
			fmt.Println("Backed up existing include file to", backupPath)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to inspect include path %q: %w", p, err)
		}
		if err := copyIncludeFile(sourceFS, inc, h, p, 0444); err != nil {
			return fmt.Errorf("failed to extract include %q: %w", inc, err)
		}
		fmt.Println("Extracted include file", p)
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
	legacyDir := h.PathTo(ct, "includes", "config")
	for _, source := range []string{legacyDir, dir} {
		if err = copyLegacyConfig(h, source, dir, site); err != nil {
			return err
		}
	}
	confs, err := fs.Glob(sourceFS, ConfigPattern)
	if err != nil {
		return err
	}
	for _, conf := range confs {
		p := h.Join(dir, site+"-"+filepath.Base(conf))
		_, err := h.Lstat(p)
		if err == nil {
			fmt.Println("Skipping existing site config file", p)
		} else if errors.Is(err, fs.ErrNotExist) {
			if err := copyIncludeFile(sourceFS, conf, h, p, 0664); err != nil {
				if errors.Is(err, fs.ErrExist) {
					fmt.Println("Skipping existing site config file", p)
					continue
				}
				return fmt.Errorf("failed to extract config include %q: %w", conf, err)
			}
			fmt.Println("Extracted site config file", p)
		} else {
			return fmt.Errorf("failed to inspect config include path %q: %w", p, err)
		}
	}
	return nil
}

// gatewayIncludePaths returns the set of cleaned local include file
// paths referenced by the gateway setups on the host, as reported by
// `geneos includes list --gateway`.
func gatewayIncludePaths(h *geneos.Host) (used map[string]struct{}, err error) {
	byGateway, _, err := getGatewayIncludes(h, nil)
	if err != nil {
		return nil, err
	}
	used = make(map[string]struct{})
	for _, includes := range byGateway {
		for _, inc := range includes {
			if inc.Remote || inc.Path == "" {
				continue
			}
			used[h.Join(inc.Path)] = struct{}{}
		}
	}
	return used, nil
}

// copyLegacyConfig copies referenced files from the old config directory
// or the previous site naming, keeping originals and existing destinations.
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
	var used map[string]struct{}
	err = h.WalkDir(source, func(name string, entry fs.DirEntry, walkErr error) (err error) {
		if walkErr != nil {
			return walkErr
		}
		if name == "." || entry.IsDir() {
			return nil
		}
		sourcePath := h.Join(source, name)
		targetName := h.Base(name)
		if source == destination {
			if !strings.HasPrefix(targetName, site+".") {
				return nil
			}
			targetName = site + "-config" + strings.TrimPrefix(targetName, site)
		} else if strings.HasPrefix(targetName, "config.") {
			targetName = site + "-" + targetName
		}
		if used == nil {
			used, err = gatewayIncludePaths(h)
			if err != nil {
				return fmt.Errorf("find include files in use by gateways: %w", err)
			}
		}
		if _, ok := used[sourcePath]; !ok {
			fmt.Println("Skipping unused legacy config file", sourcePath)
			return nil
		}
		target := h.Join(destination, h.Dir(name), targetName)
		if _, err := h.Lstat(target); err == nil {
			fmt.Println("Skipping legacy config file", sourcePath, "->", target, "(destination exists)")
			return nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		copied := false
		defer func() {
			if copied || err != nil {
				auditIncludeFile(h, "copy", target, err, slog.String("source", sourcePath))
			}
		}()
		if err := h.MkdirAll(h.Dir(target), 0755); err != nil {
			return err
		}
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
			fmt.Println("Skipping legacy config file", sourcePath, "->", target, "(destination exists)")
			return nil
		}
		if err != nil {
			return err
		}
		copied = true
		fmt.Println("Copied legacy config file", sourcePath, "->", target)
		return nil
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
	defer func() {
		if !errors.Is(err, fs.ErrExist) {
			auditIncludeFile(h, "extract", destination, err, slog.String("source", source))
		}
	}()
	src, err := sourceFS.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	return copyToNewIncludeFile(h, destination, src, mode)
}

const chunkSize = 4096

// compareContents returns true if both readers return the same bytes,
// regardless of how many bytes each Read returns.
func compareContents(f1h, f2h io.Reader) bool {
	buf1 := make([]byte, chunkSize)
	buf2 := make([]byte, chunkSize)
	for {
		n1, err1 := io.ReadFull(f1h, buf1)
		n2, err2 := io.ReadFull(f2h, buf2)
		// a short last chunk is reported as an unexpected EOF
		end1 := errors.Is(err1, io.EOF) || errors.Is(err1, io.ErrUnexpectedEOF)
		end2 := errors.Is(err2, io.EOF) || errors.Is(err2, io.ErrUnexpectedEOF)
		if (err1 != nil && !end1) || (err2 != nil && !end2) {
			return false
		}
		if n1 != n2 || !bytes.Equal(buf1[:n1], buf2[:n2]) {
			return false
		}
		if end1 || end2 {
			return end1 == end2
		}
	}
}
