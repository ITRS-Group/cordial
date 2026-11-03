package includes

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

const (
	IncludePattern = "files/itrs.*.xml"
	ConfigPattern  = "files/config.*.xml"
)

//go:embed _docs/extract.md
var includeExtractCmdDescription string

var includeExtractCmdUpdate, includeExtractCmdForce bool

func init() {
	includesCmd.AddCommand(includesExtractCmd)

	includesExtractCmd.Flags().BoolVarP(&includeExtractCmdUpdate, "update", "U", false, "Update version links for existing include files")
	includesExtractCmd.Flags().BoolVarP(&includeExtractCmdForce, "force", "F", false, "Overwrite existing files")
}

var includesExtractCmd = &cobra.Command{
	Use:          "extract",
	Args:         cobra.NoArgs,
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
		if includeExtractCmdUpdate {
			err = Update(h, "", includeExtractCmdForce)
			if err != nil {
				return err
			}
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

	resp := responses.New[responses.General](nil)

	fmt.Println("Extracting include files for version", version)

	// check for an existing directory and only proceed if --force is given
	dir := h.PathTo(ct, "includes", version)
	st, _ := h.Stat(dir)
	if st != nil && !override {
		fmt.Println("includes directory already exists at", dir, "and --force not specified, skipping extraction")
		return fmt.Errorf("includes directory already exists at %s and --force not specified", dir)
	}
	if err = h.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// extract includes matching pattern from Includes FS
	// backup existing includes if different from the source (implementation would go here)
	incs, err := fs.Glob(IncludesFS, IncludePattern)
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
		efi, err := fs.Stat(IncludesFS, inc)
		if err != nil {
			return fmt.Errorf("failed to inspect embedded include %q: %w", inc, err)
		}
		p := h.Join(dir, filepath.Base(inc))
		fi, err := h.Stat(p)
		if err == nil {
			if efi.Size() == fi.Size() {
				// compare contents
				f1h, err := IncludesFS.Open(inc)
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
					// contents are the same, skip
					fmt.Println("Skipping existing unchanged include file", p)
					continue
				}
			}
			// backup existing file
			ts := time.Now().Format("20060102150405")
			backupPath := fmt.Sprintf("%s.%s.bak", p, ts)
			if err := h.Rename(p, backupPath); err != nil {
				return err
			}
			fmt.Println("Backed up existing include file to", backupPath)
			resp.Completed = append(resp.Completed, fmt.Sprintf("backed up %s", p))
		} else if errors.Is(err, fs.ErrNotExist) {
			// file does not exist, handle accordingly
		} else {
			return fmt.Errorf("failed to inspect include path %q: %w", p, err)
		}
		if err := copyIncludeFile(IncludesFS, inc, h, p, 0444); err != nil {
			return fmt.Errorf("failed to extract include %q: %w", inc, err)
		}
		fmt.Println("Extracted include file", p)
		resp.Completed = append(resp.Completed, fmt.Sprintf("extracted %s", p))

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
		p := h.Join(dir, filepath.Base(conf))
		_, err := h.Stat(p)
		if err == nil {
			log.Debug("Config file already exists", slog.String("file", p))
		} else if errors.Is(err, fs.ErrNotExist) {
			if err := copyIncludeFile(IncludesFS, conf, h, p, 0664); err != nil {
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
