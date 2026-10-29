package includes

import (
	_ "embed"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"path/filepath"
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

var includesListCmdAll bool
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
	includeListCmd.Flags().BoolVarP(&includesListCmdNoRemote, "no-remote", "n", false, "Exclude remote include files lookups")
	includeListCmd.Flags().StringVarP(&includesListCmdFormat, "format", "f", "column", "Specify the output format for the include list. Supported formats are 'column', 'csv', 'table' and 'toolkit'.")
}

var includeListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all available include files",
	Long:    includesListCmdDescription,
	Run: func(cmd *cobra.Command, args []string) {
		// validate flags
		if includesListCmdFormat != "column" && includesListCmdFormat != "csv" && includesListCmdFormat != "toolkit" && includesListCmdFormat != "table" {
			fmt.Fprintf(os.Stderr, "Invalid format: %s\n", includesListCmdFormat)
			os.Exit(1)
		}
		if !includesListCmdByGateway && !includesListCmdByVersion && !includesListCmdByInclude {
			includesListCmdByInclude = true
		}

		listIncludes(geneos.LOCAL)
	},
}

type includeFile struct {
	Path     string
	Version  string
	FileInfo os.FileInfo
	ModTime  time.Time // for remotes, this represents the modification time of the include file on the remote location; for local files, it represents the local modification time
	Size     int64     // for remotes, this represents the size of the include file on the remote location; for local files, it represents the local file size
	Remote   bool      // Location is a remote URL
	Local    bool      // Location is in or under the gateway home directory
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

// listIncludes lists all available include files for the specified host, taking
// into account the command-line flags for filtering and formatting. It
// supports listing by gateway, by version, or by shared include files,
// and can optionally exclude remote include files if the --no-remote
// flag is set.
func listIncludes(h *geneos.Host) {
	root := h.PathTo("gateway")

	versionDirs, versionedIncludes := getVersionedIncludes(h, root)

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
				fmt.Fprintf(os.Stderr, "Failed to get file info for %s: %v\n", d, err)
				os.Exit(1)
			}
			// first, list current symlinks
			resp := responses.New[responses.General](nil)
			resp.Dataview.Table = append(resp.Dataview.Table, []string{
				"current",
				fmt.Sprintf("%d", len(vinc)),
				"-",
				fi.ModTime().Format(time.RFC3339),
				"-",
				d,
			})
			resps["current"] = resp

			if includesListCmdAll {
				for _, f := range vinc {
					resp.Dataview.Table = append(resp.Dataview.Table, []string{
						"current # " + filepath.Base(f.Path),
						"-",
						f.Version,
						f.ModTime.Format(time.RFC3339),
						humanize.Bytes(uint64(f.Size)),
						f.Path,
					})
				}
			}
		}

		for _, v := range versionDirs {
			resp := responses.New[responses.General](nil)
			fi, err := v.Info()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get file info for %s: %v\n", v.Name(), err)
				os.Exit(1)
			}
			version := v.Name()
			files, err := os.ReadDir(h.Join(root, "includes", version))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read includes directory for version %s: %v\n", version, err)
				os.Exit(1)
			}
			count := 0
			for _, f := range files {
				if !f.IsDir() && filepath.Ext(f.Name()) == ".xml" {
					count++
				}
			}
			resp.Dataview.Table = append(resp.Dataview.Table, []string{
				version,
				fmt.Sprintf("%d", count),
				"-",
				fi.ModTime().Format(time.RFC3339),
				"-",
				h.Join(root, "includes", version),
			})

			if includesListCmdAll {
				for _, f := range files {
					if !f.IsDir() && filepath.Ext(f.Name()) == ".xml" {
						fi, err := f.Info()
						if err != nil {
							fmt.Fprintf(os.Stderr, "Failed to get file info for %s: %v\n", f.Name(), err)
							os.Exit(1)
						}
						resp.Dataview.Table = append(resp.Dataview.Table, []string{
							version + " # " + f.Name(),
							"-",
							"-",
							fi.ModTime().Format(time.RFC3339),
							humanize.Bytes(uint64(fi.Size())),
							h.Join(root, "includes", version, f.Name()),
						})
					}
				}
			}

			resps[v.Name()] = resp
		}

		resps.Formatted(os.Stdout, includesListCmdFormat, []string{"Version", "Files", "Version", "Modified", "Size", "Path"}, nil)
		return
	}

	sharedIncludes := getSharedIncludes(h, root)
	gatewayIncludes, includesByPath := getGatewayIncludes(h, versionedIncludes)

	// update includes with shared includes that are not local
	for _, inc := range sharedIncludes {
		if inc.Local {
			continue
		}
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
			base := filepath.Base(include.Path)
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
				if p.Local {
					continue
				}
				gatewayCount[p.Path]++
			}
		}

		// list all includes in shared / versioned directories along
		// with the count of gateways using each include, show base name
		// and flags to indicate status such as disabled, local,
		// optional, remote, versioned, and executable

		resps := responses.GeneralResponses{}
		for path, inc := range includesByPath {
			if inc.Local {
				continue
			}
			base := filepath.Base(path)
			paths := basenameMap[base]
			if len(paths) > 1 {
				total := 0
				// add sub-enries for each base name here
				for idx, p := range paths {
					var perms fs.FileMode
					var permStr string
					var modtime time.Time
					var modStr string
					var size string

					inc, ok := includesByPath[p]
					if !ok {
						fmt.Printf("Include not found for path: %s\n", p)
						continue
					}
					if inc.FileInfo != nil {
						perms = inc.FileInfo.Mode().Perm()
						permStr = perms.String()
					}

					if !inc.ModTime.IsZero() {
						modtime = inc.ModTime
						modStr = modtime.Format(time.RFC3339)
					}

					if inc.Size != 0 {
						if includesListCmdFormat == "column" {
							size = humanize.Bytes(uint64(inc.Size))
						} else {
							size = fmt.Sprintf("%d", inc.Size)
						}
					}

					count := gatewayCount[p]
					key := fmt.Sprintf("%s # %d", base, idx+1)
					resp := responses.New[responses.General](nil)
					resp.Dataview.Table = append(resp.Dataview.Table, []string{
						key,
						"",
						fmt.Sprintf("%d", count),
						renderFlags(inc),
						inc.Version,
						permStr,
						modStr,
						size,
						p,
					})
					total += count
					resps[key] = resp
				}
				// add a top-level entry

				resp := responses.New[responses.General](nil)
				resp.Dataview.Table = append(resp.Dataview.Table, []string{
					base,
					fmt.Sprintf("%d", len(paths)),
					fmt.Sprintf("%d", total),
					"-",
					"",
					"",
					"",
					"",
					"",
				})
				resps[base] = resp
			} else {
				var perms fs.FileMode
				var permStr string
				var modtime time.Time
				var modStr string
				var size string

				if inc.FileInfo != nil {
					perms = inc.FileInfo.Mode().Perm()
					permStr = perms.String()
				}

				if !inc.ModTime.IsZero() {
					modtime = inc.ModTime
					modStr = modtime.Format(time.RFC3339)
				}

				if inc.Size != 0 {
					if includesListCmdFormat == "column" {
						size = humanize.Bytes(uint64(inc.Size))
					} else {
						size = fmt.Sprintf("%d", inc.Size)
					}
				}

				count := gatewayCount[path]
				resp := responses.New[responses.General](nil)
				resp.Dataview.Table = append(resp.Dataview.Table, []string{
					base,
					"1",
					fmt.Sprintf("%d", count),
					renderFlags(inc),
					inc.Version,
					permStr,
					modStr,
					size,
					path,
				})
				resps[base] = resp
			}

		}
		resps.Formatted(os.Stdout, includesListCmdFormat, []string{"Include", "Copies", "Gateways", "Flags", "Version", "Perms", "ModTime", "Size", "Path"}, nil)
		return
	}

	if includesListCmdByGateway {
		// build responses.GeneralReponses and call Formatted to output
		resps := responses.GeneralResponses{}
		gatewaysColumns := []string{"Gateway", "Include", "Priority", "Flags", "Version", "Reload", "Perms", "ModTime", "Size", "Location"}

		for gwname, si := range gatewayIncludes {
			for _, p := range si {
				var resp *responses.General
				var flags, permStr, modStr string
				var perms fs.FileMode
				var modtime time.Time
				var size string
				var reload string

				if p.Local && !includesListCmdAll {
					continue
				}

				if resps[gwname] != nil {
					resp = resps[gwname]
				} else {
					resp = responses.New[responses.General](p.Gateway)
					resps[gwname] = resp
				}

				flags = renderGatewayFlags(p)

				if p.FileInfo != nil {
					perms = p.FileInfo.Mode().Perm()
					permStr = perms.String()
				}

				if !p.ModTime.IsZero() {
					modtime = p.ModTime
					modStr = modtime.Format(time.RFC3339)
				}

				if p.Size != 0 {
					if includesListCmdFormat == "column" {
						size = humanize.Bytes(uint64(p.Size))
					} else {
						size = fmt.Sprintf("%d", p.Size)
					}
				}

				if p.ReloadInterval != nil {
					reload = fmt.Sprintf("%ds", *p.ReloadInterval)
				}

				resp.Dataview.Table = append(resp.Dataview.Table, []string{
					gwname,
					filepath.Base(p.Location),
					fmt.Sprintf("%d", p.Priority),
					flags,
					p.Version,
					reload,
					permStr,
					modStr,
					size,
					p.Location,
				})

				// sort rows by priority (index 2 - check if changed later)
				pri := slices.Index(gatewaysColumns, "Priority")
				sort.Slice(resp.Dataview.Table, func(i, j int) bool {
					pi, _ := strconv.Atoi(resp.Dataview.Table[i][pri])
					pj, _ := strconv.Atoi(resp.Dataview.Table[j][pri])
					return pi < pj
				})

				resps[gwname] = resp
			}
		}

		resps.Formatted(os.Stdout, includesListCmdFormat, gatewaysColumns, nil)
	}
}

func renderFlags(p includeFile) (flags string) {
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
func getSharedIncludes(h *geneos.Host, gwDir string) (includes []includeFile) {
	for _, dir := range includeDirs {
		root := h.Join(gwDir, dir)
		h.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			var modtime time.Time
			var size int64
			// var mode fs.FileMode

			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			if filepath.Ext(d.Name()) != ".xml" {
				return nil
			}

			fi, err := d.Info()
			if err == nil {
				modtime = fi.ModTime()
				size = fi.Size()
				// mode = fi.Mode()
			}
			includes = append(includes, includeFile{
				Path:     h.Join(root, path),
				FileInfo: fi,
				ModTime:  modtime,
				Size:     size,
				// Mode:     mode,
			})
			return nil
		})
	}
	return includes
}

func getVersionedIncludes(h *geneos.Host, root string) (versions []fs.DirEntry, versioned []includeFile) {
	for _, dir := range versionedDirs {
		h.WalkDir(h.Join(root, dir), func(path string, ent fs.DirEntry, err error) error {
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
				if filepath.Ext(ent.Name()) != ".xml" {
					return nil
				}
				// save symlink, decompose target path to check if it points to a versioned directory
				target, err := h.Readlink(p)
				if err != nil {
					return err
				}

				v := filepath.Dir(target)
				fi, err := h.Stat(p)
				if err != nil {
					return err
				}
				versioned = append(versioned, includeFile{
					Path:     p,
					Version:  v,
					FileInfo: fi,
					ModTime:  fi.ModTime(),
					Size:     fi.Size(),
				})
			} else {
				if filepath.Ext(ent.Name()) != ".xml" {
					return nil
				}

				// un-symlinked files, record as a possible issue
				fi, err := ent.Info()
				if err != nil {
					return err
				}
				versioned = append(versioned, includeFile{
					Path:     p,
					Version:  "", // no version since it's not symlinked
					FileInfo: fi,
					ModTime:  fi.ModTime(),
					Size:     fi.Size(),
				})
			}
			return nil
		})
	}
	return
}

// getGatewayIncludes retrieves the list of includes for each gateway
// instance on the given host. It takes a host and a list of versioned
// include files, and returns a map where the keys are gateway instance
// names and the values are slices of gatewayInclude structures
// containing detailed information about each include, such as its
// location, version, and file metadata.
func getGatewayIncludes(h *geneos.Host, versioned []includeFile) (includesByGateway map[string][]gatewayInclude, includesByPath map[string]includeFile) {
	includesByGateway = make(map[string][]gatewayInclude)
	includesByPath = make(map[string]includeFile)

	for _, gw := range instance.Instances(h, geneos.ParseComponent("gateway")) {
		var remote bool
		var local bool
		var fullpath string
		var fi fs.FileInfo
		var modtime time.Time
		var size int64
		var reload *int

		g, err := instance.ReadGatewaySetup(gw)
		if err != nil || g.Includes == nil {
			continue
		}

		// handle the main setup file for the gateway, determining if it's local or remote and gathering its metadata
		setup := config.Get[string](gw.Config(), "setup")

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
							size = s
						}
					}
				}
			}
		} else {
			local = true
			if h.IsAbs(setup) {
				fullpath = setup
			} else {
				fullpath = h.Join(gw.Home(), setup)
			}
			fi, _ = h.Stat(fullpath)
			if fi != nil {
				modtime = fi.ModTime()
				size = fi.Size()
			}
		}

		if g.Includes.Reloading != nil {
			reload = g.Includes.Reloading.ReloadInterval
		}

		// add the main setup file too
		gatewaySetup := gatewayInclude{
			Gateway:        gw,
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

		incByGateway := getIncludesFromSetup(h, gw, g.Includes.Includes, g.Includes.IncludeGroups, []string{}, versioned)
		for _, gwp := range incByGateway {
			for _, inc := range gwp {
				includesByPath[inc.Path] = inc.includeFile
			}
		}

		incByGateway[gw.Name()] = append(incByGateway[gw.Name()], gatewaySetup)

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
		var size int64
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
							size = s
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
			return group.Disabled
		})

		if fi != nil {
			modtime = fi.ModTime()
			size = fi.Size()
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
			Disabled:       disabled || inc.Disabled,
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
