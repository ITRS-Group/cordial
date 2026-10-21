package includes

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

const (
	IncludePattern = "itrs.*.xml"
	ConfigPattern  = "config.*.xml"
)

//go:embed _docs/extract.md
var includeExtractCmdDescription string

var includeExtractCmdForce bool

func init() {
	includesCmd.AddCommand(includesExtractCmd)

	includesExtractCmd.Flags().BoolVarP(&includeExtractCmdForce, "force", "F", false, "Overwrite existing files")
}

var includesExtractCmd = &cobra.Command{
	Use:          "extract",
	Short:        "Extract includes",
	Long:         includeExtractCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:       "false",
		cmd.CmdRequireHome:  "true",
		cmd.CmdAuditCommand: "never",
		cmd.CmdAuditActions: "never",
	},
	RunE: func(command *cobra.Command, args []string) (err error) {
		h := geneos.GetHost(cmd.Hostname)

		err = Extract(h, includeExtractCmdForce)
		if err != nil {
			return err
		}
		return nil
	},
}

// Extract extracts include files for the given host. If override is
// true, existing files will be overwritten. The files are placed in a
// version specific directory under the host's `gateway/includes` path.
// The caller should symlink these files as needed.
func Extract(h *geneos.Host, override bool) (err error) {
	ct := geneos.ParseComponent("gateway")
	version := cordial.VERSION

	// check for an existing directory and only proceed if --force is given
	dir := h.PathTo(ct, "includes", version)
	st, _ := h.Stat(dir)
	if st != nil && !override {
		log.Debug("includes directory already exists", slog.String("directory", dir))
		return nil
	}
	if err = h.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// extract includes matching pattern from Includes FS
	// backup existing includes if different from the source (implementation would go here)
	incs, err := fs.Glob(IncludesFS, IncludePattern)

	for _, inc := range incs {
		efi, err := fs.Stat(IncludesFS, inc)
		if err != nil {
			continue
		}
		p := h.Join(dir, inc)
		fi, err := h.Stat(p)
		if err == nil {
			if efi.Size() == fi.Size() {
				// compare contents
				f1h, err := IncludesFS.Open(inc)
				if err != nil {
					continue
				}
				defer f1h.Close()
				f2h, err := h.Open(p)
				if err != nil {
					continue
				}
				defer f2h.Close()
				if compareContents(f1h, f2h) {
					// contents are the same, skip
					log.Debug("Skipping identical include file", slog.String("include", inc))
					continue
				}
			}
			// backup existing file
			ts := time.Now().Format("20060102150405")
			backupPath := fmt.Sprintf("%s.%s.bak", p, ts)
			if err := h.Rename(p, backupPath); err != nil {
				return err
			}
		} else if errors.Is(err, fs.ErrNotExist) {
			// file does not exist, handle accordingly
		}
		// write the file
		src, err := IncludesFS.Open(inc)
		if err != nil {
			continue
		}
		defer src.Close()
		dst, err := h.Create(p, 0444)
		if err != nil {
			continue
		}
		defer dst.Close()
		_, err = io.Copy(dst, src)
		if err != nil {
			continue
		}
	}

	// now extract config files that do not exist
	dir = h.PathTo(ct, "includes", "config")
	// mkdirall can be called on existing directories without error
	if err = h.MkdirAll(dir, 0755); err != nil {
		return err
	}

	confs, err := fs.Glob(IncludesFS, ConfigPattern)
	if err != nil {
		return err
	}
	for _, conf := range confs {
		p := h.Join(dir, conf)
		_, err := h.Stat(p)
		if errors.Is(err, fs.ErrNotExist) {
			// write the file
			src, err := IncludesFS.Open(conf)
			if err != nil {
				continue
			}
			defer src.Close()
			dst, err := h.Create(p, 0664)
			if err != nil {
				continue
			}
			defer dst.Close()
			_, err = io.Copy(dst, src)
			if err != nil {
				continue
			}
		}
	}

	return nil
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
