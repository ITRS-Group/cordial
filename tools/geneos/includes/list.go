package includes

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/config"
	geneosxml "github.com/itrs-group/cordial/pkg/geneos"

	genecmd "github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

//go:embed _docs/list.md
var includesListCmdDescription string

// versionedDirs specifies the include directories that should be
// checked for version-named sub-directories (semantic versioning
// format)
var versionedDirs = []string{
	"includes",
}

var includesListCmdAll, includesListCmdLong bool
var includesListCmdNoRemote bool
var includesListCmdFormat string

var includesListCmdByGateway bool
var includesListCmdByVersion bool
var includesListCmdByInclude bool

func init() {
	includesCmd.AddCommand(includeListCmd)

	includeListCmd.Flags().BoolVarP(&includesListCmdByGateway, "gateway", "g", false, "List include files by gateway usage")
	includeListCmd.Flags().BoolVarP(&includesListCmdByVersion, "version", "v", false, "List include files by version")
	includeListCmd.Flags().BoolVarP(&includesListCmdByInclude, "include", "i", false, "List include files by shared include files")
	includeListCmd.MarkFlagsMutuallyExclusive("gateway", "version", "include")

	includeListCmd.Flags().BoolVarP(&includesListCmdAll, "all", "a", false, "List all include files, including those in versioned directories")
	includeListCmd.Flags().BoolVarP(&includesListCmdLong, "long", "l", false, "Long format with extra information")
	includeListCmd.Flags().BoolVarP(&includesListCmdNoRemote, "no-remote", "n", false, "Exclude remote include files lookups")
	includeListCmd.Flags().StringVarP(&includesListCmdFormat, "format", "f", "column", "Specify the output format for the include list. Supported formats are 'column', 'csv', 'json', 'pretty', 'table' and 'toolkit'.")
}

var includeListCmd = &cobra.Command{
	Use:          "list [NAMES...]",
	Aliases:      []string{"ls"},
	Short:        "List all available include files",
	Long:         includesListCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		geneos.CmdGlobal:               "true",
		geneos.CmdRequireHome:          "true",
		geneos.CmdWildcardNames:        "true", // only for filtering --gateway view
		geneos.CmdAllowRoot:            "true",
		geneos.CmdNonInstanceArgsError: "true",
		geneos.CmdAuditCommand:         "never",
		geneos.CmdAuditActions:         "never",
	},
	RunE: func(command *cobra.Command, _ []string) error {
		_, names, _, err := genecmd.FetchArgs(command)
		if err != nil {
			return err
		}
		if err := validateIncludesListFormat(includesListCmdFormat); err != nil {
			return err
		}
		if !includesListCmdByGateway && !includesListCmdByVersion && !includesListCmdByInclude {
			includesListCmdByInclude = true
		}

		return listIncludes(geneos.GetHost(genecmd.Hostname), names...)
	},
}

func validateIncludesListFormat(format string) error {
	switch format {
	case "column", "csv", "json", "pretty", "table", "toolkit":
		return nil
	default:
		return fmt.Errorf("invalid format %q: must be one of column, csv, json, pretty, toolkit or table", format)
	}
}

type includeFile struct {
	Path     string
	Version  string
	FileInfo os.FileInfo
	ModTime  time.Time // for remotes, this represents the modification time of the include file on the remote location; for local files, it represents the local modification time
	Size     *int64    // nil when size metadata is unavailable; a pointer to zero is a known empty file
	Remote   bool      // Location is a remote URL
	Local    bool      // Location is in or under the gateway home directory
	Shared   bool      // Location is in a shared include directory
}

type gatewayInclude struct {
	includeFile

	Gateway        geneos.Instance // gateway instance this include belongs to
	Groups         []string        // list of groups for this include
	Priority       int             // priority of the include
	Required       *bool
	ReloadInterval *int   // reload interval for the include
	Location       string // original location
	Disabled       bool   // list of disables associated with this include
}

var versionColumns = []string{"Version", "Files", "Modified", "Size", "Path"}
var versionColumnsLong = []string{"Version", "Files", "Version", "Perms", "Owner", "LastModified", "Size", "Path"}

var includeColumns = []string{"File", "Copies", "Gateways", "Flags", "LastModified", "Size", "Location"}
var includeColumnsLong = []string{"File", "Copies", "Gateways", "Flags", "Version", "Perms", "Owner", "LastModified", "Size", "Location"}

var gatewayColumns = []string{"Gateway", "File", "Priority", "Flags", "Reload", "LastModified", "Size", "Location"}
var gatewayColumnsLong = []string{"Gateway", "File", "Priority", "Flags", "Version", "Reload", "Perms", "Owner", "LastModified", "Size", "Location"}

// listIncludes lists all available include files for the specified host, taking
// into account the command-line flags for filtering and formatting. It
// supports listing by gateway, by version, or by shared include files,
// and can optionally exclude remote include files if the --no-remote
// flag is set.
func listIncludes(h *geneos.Host, names ...string) (err error) {
	root := h.PathTo("gateway")
	longFormat := includeListLongFormat()

	versionDirs, versionedIncludes, err := getVersionedIncludes(h, root)
	if err != nil {
		return err
	}

	if includesListCmdByVersion {
		resps := responses.GeneralResponses{}

		for _, v := range versionedDirs {
			vinc := versionedIncludes
			vinc = slices.DeleteFunc(vinc, func(e includeFile) bool {
				return e.Version == ""
			})
			d := h.Join(root, v)
			fi, err := h.Stat(d)
			if err != nil {
				return fmt.Errorf("failed to get file info for %s: %w", d, err)
			}
			// first, list current symlinks
			resp := responses.New[responses.General](nil)
			if longFormat {
				resp.Dataview.Table = append(resp.Dataview.Table, []string{
					"current",
					fmt.Sprintf("%d", len(vinc)),
					"-",
					"-", // perms
					"-", // owner
					fi.ModTime().Format(time.RFC3339),
					"-",
					d,
				})
			} else {
				resp.Dataview.Table = append(resp.Dataview.Table, []string{
					"current",
					fmt.Sprintf("%d", len(vinc)),
					fi.ModTime().Format(time.RFC3339),
					"-",
					d,
				})
			}
			resps["current"] = resp

			if includesListCmdAll {
				for _, f := range vinc {
					if longFormat {
						resp.Dataview.Table = append(resp.Dataview.Table, []string{
							"current # " + h.Base(f.Path),
							"-",
							f.Version,
							includePermissions(f.FileInfo.Mode(), true), // perms
							"-", // owner
							f.ModTime.Format(time.RFC3339),
							includeSizeString(f),
							f.Path,
						})
					} else {
						resp.Dataview.Table = append(resp.Dataview.Table, []string{
							"current # " + h.Base(f.Path),
							"-",
							f.ModTime.Format(time.RFC3339),
							includeSizeString(f),
							f.Path,
						})
					}
				}
			}
		}

		for _, v := range versionDirs {
			resp := responses.New[responses.General](nil)
			fi, err := v.Info()
			if err != nil {
				return fmt.Errorf("failed to get file info for %s: %w", v.Name(), err)
			}
			version := v.Name()
			files, err := h.ReadDir(h.Join(root, "includes", version))
			if err != nil {
				return fmt.Errorf("failed to read includes directory for version %s: %w", version, err)
			}
			count := 0
			for _, f := range files {
				if !f.IsDir() && h.Ext(f.Name()) == ".xml" {
					count++
				}
			}
			if longFormat {
				resp.Dataview.Table = append(resp.Dataview.Table, []string{
					version,
					fmt.Sprintf("%d", count),
					"-",
					includePermissions(fi.Mode(), true), // perms
					"-",                                 // owner
					fi.ModTime().Format(time.RFC3339),
					"-",
					h.Join(root, "includes", version),
				})
			} else {
				resp.Dataview.Table = append(resp.Dataview.Table, []string{
					version,
					fmt.Sprintf("%d", count),
					fi.ModTime().Format(time.RFC3339),
					"-",
					h.Join(root, "includes", version),
				})
			}

			if includesListCmdAll {
				for _, f := range files {
					if !f.IsDir() && h.Ext(f.Name()) == ".xml" {
						fi, err := f.Info()
						if err != nil {
							return fmt.Errorf("failed to get file info for %s: %w", f.Name(), err)
						}
						size := fi.Size()
						if longFormat {
							resp.Dataview.Table = append(resp.Dataview.Table, []string{
								version + " # " + f.Name(),
								"-",
								"-",
								includePermissions(fi.Mode(), true), // perms
								"-",                                 // owner
								fi.ModTime().Format(time.RFC3339),
								includeSizeString(includeFile{FileInfo: fi, Size: &size}),
								h.Join(root, "includes", version, f.Name()),
							})
						} else {
							resp.Dataview.Table = append(resp.Dataview.Table, []string{
								version + " # " + f.Name(),
								"-",
								fi.ModTime().Format(time.RFC3339),
								includeSizeString(includeFile{FileInfo: fi, Size: &size}),
								h.Join(root, "includes", version, f.Name()),
							})
						}
					}
				}
			}

			resps[v.Name()] = resp
		}

		columns := versionColumns
		if longFormat {
			columns = versionColumnsLong
		}
		return formatIncludeResponses(os.Stdout, resps, columns)
	}

	sharedIncludes, err := getSharedIncludes(h, root)
	if err != nil {
		return err
	}
	var gatewayNames []string
	if includesListCmdByGateway {
		gatewayNames = names
	}
	gatewayIncludes, includesByPath := getGatewayIncludes(h, versionedIncludes, gatewayNames...)

	// update includesByPath with shared and versioned includes
	sharedPaths := make(map[string]struct{}, len(sharedIncludes))
	for _, inc := range sharedIncludes {
		includesByPath[inc.Path] = inc
		sharedPaths[inc.Path] = struct{}{}
	}
	for name, gatewayList := range gatewayIncludes {
		for i := range gatewayList {
			if _, ok := sharedPaths[gatewayList[i].Path]; ok {
				gatewayList[i].Shared = true
			}
		}
		gatewayIncludes[name] = gatewayList
	}

	for _, inc := range versionedIncludes {
		includesByPath[inc.Path] = inc
	}

	// check all the versioned and shared includes for how many gateways
	// reference them, check for duplicate base names (inc versus
	// versioned files) etc.
	if includesListCmdByInclude {
		// build a map of all basenames of includes to a slice of full
		// paths that have that basename
		basenameMap := make(map[string][]string)
		for _, include := range includesByPath {
			base := h.Base(include.Path)
			if _, ok := basenameMap[base]; !ok {
				basenameMap[base] = []string{}
			}
			basenameMap[base] = append(basenameMap[base], include.Path)
		}

		// now remove duplicate paths from each base
		for base, paths := range basenameMap {
			slices.Sort(paths)
			basenameMap[base] = slices.Compact(paths)
		}

		// count gateways using each by fullpath (or url)
		gatewayCount := make(map[string]int)
		for _, si := range gatewayIncludes {
			for _, p := range si {
				gatewayCount[p.Path]++
			}
		}

		// list all includes in shared / versioned directories along
		// with the count of gateways using each include, show base name
		// and flags to indicate status such as disabled, local,
		// optional, remote, versioned, and executable

		resps := responses.GeneralResponses{}
		for path, inc := range includesByPath {
			if inc.Local && !includesListCmdAll {
				continue
			}
			base := h.Base(path)
			paths := basenameMap[base]
			if len(paths) > 1 {
				total := 0
				// Add a row for each include that shares this basename.
				for idx, p := range paths {
					inc, ok := includesByPath[p]
					if !ok {
						return fmt.Errorf("include not found for path %q", p)
					}
					count := gatewayCount[p]
					key := fmt.Sprintf("%s # %d", base, idx+1)
					resp := responses.New(nil)
					resp.Dataview.Table = append(resp.Dataview.Table, includeFileRow(key, "", count, inc, p, longFormat))
					total += count
					resps[key] = resp
				}

				// Add a summary row for the basename shared by these files.
				resp := responses.New(nil)
				resp.Dataview.Table = append(resp.Dataview.Table, includeSummaryRow(base, len(paths), total, longFormat))
				resps[base] = resp
			} else {
				count := gatewayCount[path]
				resp := responses.New(nil)
				resp.Dataview.Table = append(resp.Dataview.Table, includeFileRow(base, "1", count, inc, path, longFormat))
				resps[base] = resp
			}

		}
		columns := includeColumns
		if longFormat {
			columns = includeColumnsLong
		}
		return formatIncludeResponses(os.Stdout, resps, columns)
	}

	if includesListCmdByGateway {
		// build responses.GeneralReponses and call Formatted to output
		resps := responses.GeneralResponses{}
		columns := gatewayColumns
		if longFormat {
			columns = gatewayColumnsLong
		}

		for gwname, si := range gatewayIncludes {
			for _, p := range si {
				var resp *responses.General
				if p.Local && !includesListCmdAll {
					continue
				}

				if resps[gwname] != nil {
					resp = resps[gwname]
				} else {
					resp = responses.New(nil)
					resps[gwname] = resp
				}

				resp.Dataview.Table = append(resp.Dataview.Table, gatewayIncludeRow(gwname, p, longFormat))

				// Priority is the third column in both output schemas.
				pri := slices.Index(columns, "Priority")
				sort.Slice(resp.Dataview.Table, func(i, j int) bool {
					pi, _ := strconv.Atoi(resp.Dataview.Table[i][pri])
					pj, _ := strconv.Atoi(resp.Dataview.Table[j][pri])
					return pi < pj
				})

				resps[gwname] = resp
			}
		}

		return formatIncludeResponses(os.Stdout, resps, columns)
	}
	return nil
}

func formatIncludeResponses(w io.Writer, result responses.GeneralResponses, columns []string) error {
	if !includesListJSON() {
		return result.Formatted(w, includesListCmdFormat, columns, nil)
	}

	rows := make([]map[string]any, 0)
	for _, key := range slices.Sorted(maps.Keys(result)) {
		response := result[key]
		if response == nil || response.Err != nil || response.Dataview == nil {
			continue
		}
		for _, row := range response.Dataview.Table {
			item := make(map[string]any, len(columns))
			for index, column := range columns {
				if index >= len(row) {
					break
				}
				if column == "Owner" {
					continue
				}
				if column == "Flags" {
					addIncludeFlagFields(item, row[index])
					continue
				}
				value := includeJSONValue(column, row[index])
				if column == "Copies" && value == 0 {
					continue
				}
				if text, ok := value.(string); ok && text == "" {
					continue
				}
				name := strings.ToLower(column[:1]) + column[1:]
				item[name] = value
			}
			rows = append(rows, item)
		}
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if includesListCmdFormat == "pretty" {
		encoder.SetIndent("", "    ")
	}
	return encoder.Encode(rows)
}

func includesListJSON() bool {
	return includesListCmdFormat == "json" || includesListCmdFormat == "pretty"
}

func includeListLongFormat() bool {
	return includesListCmdLong || includesListJSON()
}

func includePermissions(mode fs.FileMode, includeType bool) string {
	if includesListJSON() {
		return fmt.Sprintf("%#o", mode.Perm())
	}
	if includeType {
		return mode.String()
	}
	return mode.Perm().String()
}

func addIncludeFlagFields(fields map[string]any, flags string) {
	flagNames := map[rune]string{
		'D': "disabled",
		'L': "local",
		'R': "readOnly",
		'S': "shared",
		'U': "remote",
		'X': "missing",
		'o': "optional",
	}
	for _, flag := range flags {
		if name, ok := flagNames[flag]; ok {
			fields[name] = true
		}
	}
}

func includeJSONValue(column, value string) any {
	switch column {
	case "Copies", "Files", "Gateways", "Priority":
		if column == "Copies" && value == "" {
			return 0
		}
		if number, err := strconv.Atoi(value); err == nil {
			return number
		}
	case "Size":
		if value == "" || value == "-" {
			return int64(-1)
		}
		if size, err := strconv.ParseInt(value, 10, 64); err == nil {
			return size
		}
		return int64(-1)
	}
	return value
}

func includeSizeString(include includeFile) string {
	if include.Size == nil {
		if includesListCmdFormat == "json" || includesListCmdFormat == "pretty" {
			return "-1"
		}
		return ""
	}
	if includesListCmdFormat == "column" {
		return humanize.Bytes(uint64(*include.Size))
	}
	return fmt.Sprintf("%d", *include.Size)
}

func includeFileRow(name, copies string, gatewayCount int, include includeFile, location string, long bool) []string {
	modified := ""
	if !include.ModTime.IsZero() {
		modified = include.ModTime.Format(time.RFC3339)
	}
	size := includeSizeString(include)
	if !long {
		return []string{name, copies, fmt.Sprintf("%d", gatewayCount), renderFlags(include), modified, size, location}
	}

	permissions := ""
	if include.FileInfo != nil {
		permissions = includePermissions(include.FileInfo.Mode(), false)
	}
	return []string{name, copies, fmt.Sprintf("%d", gatewayCount), renderFlags(include), include.Version, permissions, "", modified, size, location}
}

func includeSummaryRow(name string, copies, gatewayCount int, long bool) []string {
	row := []string{name, fmt.Sprintf("%d", copies), fmt.Sprintf("%d", gatewayCount), "", "", "", ""}
	if long {
		row = []string{name, fmt.Sprintf("%d", copies), fmt.Sprintf("%d", gatewayCount), "", "", "", "", "", "", ""}
	}
	return row
}

func gatewayIncludeRow(gateway string, include gatewayInclude, long bool) []string {
	name := path.Base(include.Location)
	if include.Gateway != nil {
		name = include.Gateway.Host().Base(include.Location)
	}
	modified := ""
	if !include.ModTime.IsZero() {
		modified = include.ModTime.Format(time.RFC3339)
	}
	size := includeSizeString(include.includeFile)
	reload := ""
	if include.ReloadInterval != nil {
		reload = fmt.Sprintf("%ds", *include.ReloadInterval)
	}
	row := []string{gateway, name, fmt.Sprintf("%d", include.Priority), renderGatewayFlags(include), reload, modified, size, include.Location}
	if !long {
		return row
	}
	permissions := ""
	if include.FileInfo != nil {
		permissions = includePermissions(include.FileInfo.Mode(), false)
	}
	return []string{gateway, name, fmt.Sprintf("%d", include.Priority), renderGatewayFlags(include), include.Version, reload, permissions, "", modified, size, include.Location}
}

func renderFlags(p includeFile) (flags string) {
	if p.Shared {
		flags += "S"
	}

	// gateway local include (not shared)
	if p.Local {
		flags += "L"
	}

	// URL / remote
	if p.Remote {
		flags += "U"
	}

	// read-only file
	if !p.Remote && p.FileInfo != nil && p.FileInfo.Mode()&0222 == 0 {
		flags += "R"
	}

	// versioned include file
	if p.Version != "" {
		flags += "V"
	}

	// file not found
	if !p.Remote && p.FileInfo == nil {
		flags += "X"
	}

	if flags == "" {
		flags = "-"
	}

	return sortString(flags)
}

func renderGatewayFlags(p gatewayInclude) (flags string) {
	flags = renderFlags(p.includeFile)

	if flags == "-" {
		flags = ""
	}

	// disabled
	if p.Disabled {
		flags += "D"
	}

	// optional
	if p.Required != nil && !*p.Required {
		flags += "o"
	}

	if flags == "" {
		flags = "-"
	}

	return sortString(flags)
}

func sortString(s string) string {
	r := []rune(s)
	sort.Slice(r, func(i, j int) bool { return r[i] < r[j] })
	return string(r)
}

// includeDirs specifies the shared include directories to search for
// include files relative to the top-level `gateway` directory.
// Individual instance references are resolved separately.
var includeDirs = []string{
	"includes/config",
	"shared",
	"gateway_shared",
}

// getSharedIncludes searches for shared include files within the
// specified gateway directory on the given host. It traverses the
// predefined include directories and collects all XML files, returning
// them as a slice of directory entries.
func getSharedIncludes(h *geneos.Host, gwDir string) (includes []includeFile, err error) {
	for _, dir := range includeDirs {
		root := h.Join(gwDir, dir)
		err = h.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			var modtime time.Time

			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			if h.Ext(d.Name()) != ".xml" {
				return nil
			}

			fi, err := d.Info()
			if err != nil {
				return err
			}
			modtime = fi.ModTime()
			size := fi.Size()
			includes = append(includes, includeFile{
				Path:     h.Join(root, path),
				FileInfo: fi,
				ModTime:  modtime,
				Size:     &size,
				Shared:   true,
			})
			return nil
		})
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
	}
	return includes, nil
}

func getVersionedIncludes(h *geneos.Host, root string) (versions []fs.DirEntry, versioned []includeFile, err error) {
	for _, dir := range versionedDirs {
		err = h.WalkDir(h.Join(root, dir), func(path string, ent fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if ent.Name() == "." {
				// continue descending
				return nil
			}

			p := h.Join(root, dir, ent.Name())

			if ent.IsDir() {
				// Check if the directory name matches a semantic
				// version pattern, and save. do not descend into
				// directories, so return skipdir
				if _, err := version.NewVersion(ent.Name()); err == nil {
					versions = append(versions, ent)
				}
				return fs.SkipDir
			} else if info, err := ent.Info(); err == nil && info.Mode()&fs.ModeSymlink != 0 {
				if h.Ext(ent.Name()) != ".xml" {
					return nil
				}
				// save symlink, decompose target path to check if it points to a versioned directory
				target, err := h.Readlink(p)
				if err != nil {
					return err
				}

				v := h.Dir(target)
				fi, err := h.Stat(p)
				if err != nil {
					return err
				}
				size := fi.Size()
				versioned = append(versioned, includeFile{
					Path:     p,
					Version:  v,
					FileInfo: fi,
					ModTime:  fi.ModTime(),
					Size:     &size,
				})
			} else {
				if h.Ext(ent.Name()) != ".xml" {
					return nil
				}

				// un-symlinked files, record as a possible issue
				fi, err := ent.Info()
				if err != nil {
					return err
				}
				size := fi.Size()
				versioned = append(versioned, includeFile{
					Path:     p,
					Version:  "", // no version since it's not symlinked
					FileInfo: fi,
					ModTime:  fi.ModTime(),
					Size:     &size,
				})
			}
			return nil
		})
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
	}
	return versions, versioned, nil
}

// getGatewayIncludes retrieves the list of includes for each gateway
// instance on the given host. It takes a host and a list of versioned
// include files, and returns a map where the keys are gateway instance
// names and the values are slices of gatewayInclude structures
// containing detailed information about each include, such as its
// location, version, and file metadata.
func getGatewayIncludes(h *geneos.Host, versioned []includeFile, names ...string) (includesByGateway map[string][]gatewayInclude, includesByPath map[string]includeFile) {
	includesByGateway = make(map[string][]gatewayInclude)
	includesByPath = make(map[string]includeFile)

	for _, i := range instance.Instances(h, nil, instance.MatchNames(names...)) {
		if !instance.IsA(i, "gateway") {
			continue
		}
		var remote bool
		var local bool
		var fullpath string
		var fi fs.FileInfo
		var modtime time.Time
		var size *int64
		var reload *int

		g, err := instance.ReadGatewaySetup(i)
		if err != nil || g.Includes == nil {
			continue
		}

		// handle the main setup file for the gateway, determining if it's local or remote and gathering its metadata
		cf := i.Config()
		setup := config.Get[string](cf, "setup")

		if strings.HasPrefix(setup, "https://") || strings.HasPrefix(setup, "http://") {
			remote = true
			if !includesListCmdNoRemote {
				// perform a HEAD request and fill in what we can
				resp, err := http.Head(setup)
				if err == nil {
					defer resp.Body.Close()
					if lm := resp.Header.Get("Last-Modified"); lm != "" {
						if t, err := http.ParseTime(lm); err == nil {
							modtime = t.Local()
						}
					}
					if cl := resp.Header.Get("Content-Length"); cl != "" {
						if s, err := strconv.ParseInt(cl, 10, 64); err == nil {
							size = &s
						}
					}
				}
			}
		} else {
			local = true
			if h.IsAbs(setup) {
				fullpath = setup
			} else {
				fullpath = h.Join(i.Home(), setup)
			}
			fi, _ = h.Stat(fullpath)
			if fi != nil {
				modtime = fi.ModTime()
				sizeValue := fi.Size()
				size = &sizeValue
			}
		}

		if g.Includes.Reloading != nil {
			reload = g.Includes.Reloading.ReloadInterval
		}

		// add the main setup file too
		gatewaySetup := gatewayInclude{
			Gateway:        i,
			Priority:       g.Includes.Priority,
			Required:       &[]bool{true}[0],
			ReloadInterval: reload,
			Location:       setup,
			Path:           fullpath,
			Local:          local,
			Remote:         remote,
			FileInfo:       fi,
			ModTime:        modtime,
			Size:           size,
		}

		includesByPath[gatewaySetup.Path] = gatewaySetup.includeFile

		incByGateway := getIncludesFromSetup(h, i, g.Includes.Includes, g.Includes.IncludeGroups, []string{}, versioned)
		for _, gwp := range incByGateway {
			for _, inc := range gwp {
				includesByPath[inc.Path] = inc.includeFile
			}
		}

		incByGateway[i.Name()] = append(incByGateway[i.Name()], gatewaySetup)

		// merge them in
		maps.Copy(includesByGateway, incByGateway)
	}
	return
}

// getIncludesFromSetup recursively extracts all include file
// locations from a given gateway setup, including those in include
// groups. It returns a slice of IncludePath with Groups needing to be
// prefixed with the parent group hierarchy.
func getIncludesFromSetup(h *geneos.Host, gw geneos.Instance, includes []geneosxml.Include, groups []geneosxml.IncludeGroup, groupPath []string, versioned []includeFile) (result map[string][]gatewayInclude) {
	result = make(map[string][]gatewayInclude)

	// Iterate over each include in the provided list and determine its
	// full path, whether it is local or remote, and gather metadata
	// such as modification time, size, and file mode. Also, check for
	// version information if available.
	for _, inc := range includes {
		var local, remote bool
		var fi fs.FileInfo
		var modtime time.Time
		var size *int64
		var version string

		// leave inc.Location unchanged, for inclusion later
		abspath := inc.Location

		if strings.HasPrefix(inc.Location, "https://") || strings.HasPrefix(inc.Location, "http://") {
			remote = true
			if !includesListCmdNoRemote {
				// perform a HEAD request and fill in what we can
				resp, err := http.Head(inc.Location)
				if err == nil {
					defer resp.Body.Close()
					if lm := resp.Header.Get("Last-Modified"); lm != "" {
						if t, err := http.ParseTime(lm); err == nil {
							modtime = t.Local()
						}
					}
					if cl := resp.Header.Get("Content-Length"); cl != "" {
						if s, err := strconv.ParseInt(cl, 10, 64); err == nil {
							size = &s
						}
					}
				}
			}
		} else {
			if !h.IsAbs(inc.Location) {
				abspath = h.Join(gw.Home(), inc.Location)
			}
			if strings.HasPrefix(abspath, gw.Home()) {
				local = true
			}
			if i, err := h.Stat(abspath); err == nil {
				fi = i
			}
		}

		if v := slices.IndexFunc(versioned, func(v includeFile) bool {
			return abspath == v.Path
		}); v != -1 {
			version = versioned[v].Version
		}

		disabled := slices.ContainsFunc(groups, func(group geneosxml.IncludeGroup) bool {
			return group.Disabled != nil && *group.Disabled
		})

		if fi != nil {
			modtime = fi.ModTime()
			sizeValue := fi.Size()
			size = &sizeValue
		}

		result[gw.Name()] = append(result[gw.Name()], gatewayInclude{
			Path:     abspath,
			FileInfo: fi,
			ModTime:  modtime,
			Size:     size,
			Version:  version,
			Local:    local,
			Remote:   remote,

			Gateway:        gw,
			Groups:         groupPath,
			Priority:       inc.Priority,
			Required:       inc.Required,
			ReloadInterval: inc.ReloadInterval,
			Location:       inc.Location,
			Disabled:       disabled || inc.Disabled != nil && *inc.Disabled,
		})
	}

	// Recursively process each include group to extract nested
	// includes. This ensures that all sub-groups are traversed and
	// their includes are added to the result map. The sub-group path is
	// constructed by appending the group name to the current group path.
	for _, inc := range groups {
		// recursively extract includes from sub-groups
		subPath := append(append([]string{}, groupPath...), inc.Name)
		for gwName, si := range getIncludesFromSetup(h, gw, inc.Includes, inc.IncludeGroups, subPath, versioned) {
			result[gwName] = append(result[gwName], si...)
		}
	}

	return result
}
