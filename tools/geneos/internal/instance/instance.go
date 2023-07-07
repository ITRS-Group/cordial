/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package instance

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/pkg/process"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// The Instance type is the common data shared by all instance / component types
type Instance struct {
	geneos.Instance `json:"-"`
	Conf            *config.Config    `json:"-"`
	InstanceHost    *geneos.Host      `json:"-"`
	Component       *geneos.Component `json:"-"`
	ConfigLoaded    bool              `json:"-"`
}

// IsA returns true if instance c has a type that is component of the
// type name. If name is not a known component type then false is
// returned without checking the instance.
func IsA(c geneos.Instance, name string) bool {
	ct := geneos.FindComponent(name)
	if ct == nil {
		return false
	}
	return ct.IsA(c.Type().String())
}

// DisplayName returns the type, name and non-local host as a string
// suitable for display.
func DisplayName(c geneos.Instance) string {
	if c.Host().IsLocal() {
		return fmt.Sprintf("%s %q", c.Type(), c.Name())
	}
	return fmt.Sprintf("%s \"%s@%s\"", c.Type(), c.Name(), c.Host())
}

// ReservedName returns true if in name a reserved word. Reserved names
// are checked against all the values registered by components at
// start-up.
func ReservedName(name string) (ok bool) {
	log.Debug().Msgf("checking %q", name)
	if geneos.FindComponent(name) != nil {
		log.Debug().Msg("matches a reserved word")
		return true
	}
	if config.GetString("reservednames") != "" {
		list := strings.Split(name, ",")
		for _, n := range list {
			if strings.EqualFold(name, strings.TrimSpace(n)) {
				log.Debug().Msg("matches a user defined reserved name")
				return true
			}
		}
	}
	return
}

// spaces are valid - dumb, but valid - for now. If the name starts with
// number then the next character cannot be a number or '.' to help
// distinguish from versions
var validStringRE = regexp.MustCompile(`^\w[\w-]?[:@\.\w -]*$`)

// ValidInstanceName returns true if name is considered a valid instance
// name. It is not checked against the list of reserved names.
//
// XXX used to consume instance names until parameters are then passed
// down
func ValidInstanceName(name string) (ok bool) {
	ok = validStringRE.MatchString(name)
	if !ok {
		log.Debug().Msgf("no rexexp match: %s", name)
	}
	return
}

// LogFile returns the full path to the log file for the instance.
//
// XXX logdir = LogD relative to Home or absolute
func LogFile(c geneos.Instance) (logfile string) {
	logdir := path.Clean(c.Config().GetString("logdir"))
	switch {
	case logdir == "":
		logfile = c.Home()
	case filepath.IsAbs(logdir):
		logfile = logdir
	default:
		logfile = path.Join(c.Home(), logdir)
	}
	logfile = path.Join(logfile, c.Config().GetString("logfile"))
	return
}

// Signal sends the signal to the instance
func Signal(c geneos.Instance, signal syscall.Signal) (err error) {
	pid, err := GetPID(c)
	if err != nil {
		return os.ErrProcessDone
	}

	return c.Host().Signal(pid, signal)
}

// Get return an instance of component ct, and loads the config. It is
// an error if the config cannot be loaded.
func Get(ct *geneos.Component, name string) (c geneos.Instance, err error) {
	if ct == nil {
		return nil, geneos.ErrInvalidArgs
	}

	c = ct.New(name)
	if c == nil {
		// if no instance is created, check why
		_, _, h := SplitName(name, geneos.LOCAL)
		if h == geneos.LOCAL && geneos.Root() == "" {
			err = geneos.ErrRootNotSet
			return
		}
		err = geneos.ErrInvalidArgs
		return
	}
	err = c.Load()
	return
}

// GetAll returns a slice of instances for a given component type on remote r
func GetAll(r *geneos.Host, ct *geneos.Component) (confs []geneos.Instance) {
	if ct == nil {
		for _, c := range geneos.RealComponents() {
			confs = append(confs, GetAll(r, c)...)
		}
		return
	}
	for _, name := range AllNames(r, ct) {
		i, err := Get(ct, name)
		if err != nil {
			continue
		}
		confs = append(confs, i)
	}

	return
}

// Match looks for exactly one matching instance across types and hosts
// returns Invalid Args if zero of more than 1 match
func Match(ct *geneos.Component, h *geneos.Host, name string) (c geneos.Instance, err error) {
	list := MatchAll(ct, h, name)
	if len(list) == 0 {
		err = os.ErrNotExist
		return
	}
	if len(list) == 1 {
		c = list[0]
		return
	}
	err = geneos.ErrInvalidArgs
	return
}

// MatchAll constructs and returns a slice of instances that have a
// matching name
func MatchAll(ct *geneos.Component, h *geneos.Host, name string) (c []geneos.Instance) {
	_, local, r := SplitName(name, h)
	if !r.IsAvailable() {
		log.Debug().Err(host.ErrNotAvailable).Msgf("host %s", r)
		return
	}

	if h != geneos.ALL && r.String() != h.String() {
		return
	}

	if ct == nil {
		for _, ct := range geneos.RealComponents() {
			c = append(c, MatchAll(ct, h, name)...)
		}
		return
	}

	for _, name := range AllNames(r, ct) {
		// for case insensitive match change to EqualFold here
		_, ldir, _ := SplitName(name, geneos.ALL)
		if path.Base(ldir) == local {
			i, err := Get(ct, name)
			if err != nil {
				continue
			}
			c = append(c, i)
		}
	}

	return
}

// MatchKeyValue returns a slice of instances where the instance
// configuration key matches the value given.
func MatchKeyValue(h *geneos.Host, ct *geneos.Component, key, value string) (confs []geneos.Instance) {
	if ct == nil {
		for _, c := range geneos.RealComponents() {
			confs = append(confs, MatchKeyValue(h, c, key, value)...)
		}
		return
	}

	for _, name := range AllNames(h, ct) {
		i, err := Get(ct, name)
		if err != nil {
			continue
		}
		confs = append(confs, i)
	}

	// filter in place
	n := 0
	for _, c := range confs {
		if c.Config().GetString(key) == value {
			confs[n] = c
			n++
		}
	}
	confs = confs[:n]

	return
}

// GetPorts gets all used ports in config files on a specific remote
// this will not work for ports assigned in component config files, such
// as gateway setup or netprobe collection agent
//
// returns a map
func GetPorts(r *geneos.Host) (ports map[uint16]*geneos.Component) {
	if r == geneos.ALL {
		log.Fatal().Msg("getports() call with all hosts")
	}
	ports = make(map[uint16]*geneos.Component)
	for _, c := range GetAll(r, nil) {
		if !c.Loaded() {
			log.Error().Msgf("cannot load configuration for %s", c)
			continue
		}
		if port := c.Config().GetInt("port"); port != 0 {
			ports[uint16(port)] = c.Type()
		}
	}
	return
}

// syntax of ranges of ints:
// x,y,a-b,c..d m n o-p
// also open ended A,N-,B
// command or space seperated?
// - or .. = inclusive range
//
// how to represent
// split, for range, check min-max -> max > min
// repeats ignored
// special ports? - nah
//

// given a range, find the first unused port
//
// range is comma or two-dot separated list of
// single number, e.g. "7036"
// min-max inclusive range, e.g. "7036-8036"
// start- open ended range, e.g. "7041-"
//
// some limits based on https://en.wikipedia.org/wiki/List_of_TCP_and_UDP_port_numbers
//
// not concurrency safe at this time
func NextPort(r *geneos.Host, ct *geneos.Component) uint16 {
	from := config.GetString(ct.PortRange)
	used := GetPorts(r)
	ps := strings.Split(from, ",")
	for _, p := range ps {
		// split on comma or ".."
		m := strings.SplitN(p, "-", 2)
		if len(m) == 1 {
			m = strings.SplitN(p, "..", 2)
		}

		if len(m) > 1 {
			var min uint16
			mn, err := strconv.Atoi(m[0])
			if err != nil {
				continue
			}
			if mn < 0 || mn > 65534 {
				min = 65535
			} else {
				min = uint16(mn)
			}
			if m[1] == "" {
				m[1] = "49151"
			}
			max, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}
			if int(min) >= max {
				continue
			}
			for i := min; int(i) <= max; i++ {
				if _, ok := used[i]; !ok {
					// found an unused port
					return i
				}
			}
		} else {
			var p1 uint16
			p, err := strconv.Atoi(m[0])
			if err != nil {
				continue
			}
			if p < 0 || p > 65534 {
				p1 = 65535
			} else {
				p1 = uint16(p)
			}
			if _, ok := used[p1]; !ok {
				return p1
			}
		}
	}
	return 0
}

// ForAll calls the supplied function for each matching instance. It
// prints any returned error on STDOUT and the only error returned is
// os.ErrNotExist if there are no matching instances.
func ForAll(ct *geneos.Component, hostname string, fn func(geneos.Instance, []string) error, args []string, params []string) (err error) {
	n := 0
	// if args is empty, get all matching instances. this allows internal
	// calls with an empty arg list without having to do the parseArgs()
	// dance
	h := geneos.GetHost(hostname)
	if h == nil {
		h = geneos.ALL
	}
	if len(args) == 0 {
		args = AllNames(h, ct)
	}

	allcs := []geneos.Instance{}

	for _, name := range args {
		cs := MatchAll(ct, h, name)
		if len(cs) == 0 {
			continue
		}
		n++
		allcs = append(allcs, cs...)
	}

	if n == 0 {
		return os.ErrNotExist
	}

	for _, c := range allcs {
		if err = fn(c, params); err != nil && !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, geneos.ErrNotSupported) {
			fmt.Printf("%s: %s\n", c, err)
		}
	}

	return nil
}

// ForAllWithResults calls the given function for each matching instance
// and gather the return values into a slice of interfaces for handling
// upstream. Errors are printed on STDOUT for each call and the only
// error returned ErrNotExist if there are no matches.
func ForAllWithResults(ct *geneos.Component, hostname string, fn func(geneos.Instance, []string) (interface{}, error), args []string, params []string) (results []interface{}, err error) {
	n := 0
	// if args is empty, get all matching instances. this allows internal
	// calls with an empty arg list without having to do the parseArgs()
	// dance
	h := geneos.GetHost(hostname)
	if h == nil {
		h = geneos.ALL
	}
	if len(args) == 0 {
		args = AllNames(h, ct)
	}
	allcs := []geneos.Instance{}

	for _, name := range args {
		cs := MatchAll(ct, h, name)
		if len(cs) == 0 {
			continue
		}
		n++
		allcs = append(allcs, cs...)
	}

	for _, c := range allcs {
		var res interface{}
		if res, err = fn(c, params); err != nil && !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, geneos.ErrNotSupported) {
			fmt.Printf("%s: %s\n", c, err)
		}
		if res != nil {
			results = append(results, res)
		}
	}
	if n == 0 {
		return nil, os.ErrNotExist
	}
	return results, nil
}

// AllNames returns a slice of all instance names for a given component.
// No checking is done to validate that the directory is a populated
// instance.
//
// To support the move to parent types we do a little more, looking for
// legacy locations in here
func AllNames(h *geneos.Host, ct *geneos.Component) (names []string) {
	var files []fs.DirEntry

	if h == nil {
		h = geneos.ALL
	}

	if h == geneos.ALL {
		for _, h := range geneos.AllHosts() {
			names = append(names, AllNames(h, ct)...)
		}
		return
	}

	if ct == nil {
		for _, ct := range geneos.RealComponents() {
			// ignore errors, we only care about any files found
			for _, dir := range ct.InstancesDirs(h) {
				// log.Debug().Msgf("ct, dirs: %s %s", ct, dir)
				d, _ := h.ReadDir(dir)
				files = append(files, d...)
			}
		}
	} else {
		// ignore errors, we only care about any files found
		for _, dir := range ct.InstancesDirs(h) {
			d, _ := h.ReadDir(dir)
			files = append(files, d...)
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})
	for i, file := range files {
		// skip for values with the same name as previous
		if i > 0 && i < len(files) && file.Name() == files[i-1].Name() {
			continue
		}
		if file.IsDir() {
			names = append(names, file.Name()+"@"+h.String())
		}
	}
	return
}

// SplitName returns the parts of an instance name given an instance
// name in the format [TYPE:]NAME[@HOST] and a default host, return a
// *geneos.Component for the TYPE if given, a string for the NAME and a
// *geneos.Host - the latter being either from the name or the default
// provided
func SplitName(in string, defaultHost *geneos.Host) (ct *geneos.Component, name string, h *geneos.Host) {
	if defaultHost == nil {
		h = geneos.ALL
	} else {
		h = defaultHost
	}
	parts := strings.SplitN(in, "@", 2)
	name = parts[0]
	if len(parts) > 1 {
		h = geneos.GetHost(parts[1])
	}
	parts = strings.SplitN(name, ":", 2)
	if len(parts) > 1 {
		ct = geneos.FindComponent(parts[0])
		name = parts[1]
	}
	return
}

// BuildCmd gathers the path to the binary, arguments and any environment variables
// for an instance and returns an exec.Cmd, almost ready for execution. Callers
// will add more details such as working directories, user and group etc.
//
// If nodecode is set then any secure environment vars are not decoded, so OK for display
func BuildCmd(c geneos.Instance, nodecode bool) (cmd *exec.Cmd, env []string, home string) {
	binary := PathOf(c, "program")

	args, env, home := c.Command()

	opts := strings.Fields(c.Config().GetString("options"))
	args = append(args, opts...)

	envs := c.Config().GetStringSlice("Env", config.NoDecode(nodecode))
	libs := []string{}
	if c.Config().GetString("libpaths") != "" {
		libs = append(libs, c.Config().GetString("libpaths"))
	}

	for _, e := range envs {
		switch {
		case strings.HasPrefix(e, "LD_LIBRARY_PATH="):
			libs = append(libs, strings.TrimPrefix(e, "LD_LIBRARY_PATH="))
		default:
			env = append(env, e)
		}
	}
	if len(libs) > 0 {
		env = append(env, "LD_LIBRARY_PATH="+strings.Join(libs, ":"))
	}
	cmd = exec.Command(binary, args...)

	return
}

// Disable the instance c. Does not try to stop a running instance and
// returns an error if it is running.
func Disable(c geneos.Instance) (err error) {
	if IsRunning(c) {
		return fmt.Errorf("instance %s running", c)
	}

	disablePath := ComponentFilepath(c, geneos.DisableExtension)

	h := c.Host()

	f, err := h.Create(disablePath, 0664)
	if err != nil {
		return err
	}
	f.Close()
	return
}

// Enable removes the disabled flag, if any,m from instance c.
func Enable(c geneos.Instance) (err error) {
	disableFile := ComponentFilepath(c, geneos.DisableExtension)
	if _, err = c.Host().Stat(disableFile); err != nil {
		return nil
	}
	return c.Host().Remove(disableFile)
}

// GetPID returns the PID of the process running for the instance. If
// not found then an err of os.ErrProcessDone is returned.
//
// The process is identified by checking the conventions used to start
// Geneos processes.
//
// the component type must be part of the basename of the executable and
// the component name must be on the command line as an exact and
// standalone args
//
// walk the /proc directory (local or remote) and find the matching pid.
// This is subject to races, but not much we can do
func GetPID(c geneos.Instance) (pid int, err error) {
	// if fn := c.Type().GetPID; fn != nil {
	// 	return fn(c)
	// }

	return process.GetPID(c.Host(), c.Config().GetString("binary"), c.Type().GetPID, c, c.Name())
}

func GetPIDInfo(c geneos.Instance) (pid int, uid int, gid int, mtime time.Time, err error) {
	if pid, err = GetPID(c); err != nil {
		return
	}

	var st os.FileInfo
	st, err = c.Host().Stat(fmt.Sprintf("/proc/%d", pid))
	s := c.Host().GetFileOwner(st)
	return pid, s.Uid, s.Gid, st.ModTime(), err
}

var tcpfiles = []string{
	"/proc/net/tcp",
	"/proc/net/tcp6",
}

// allTCPListenPorts returns a map of inodes to ports for all listening
// TCP ports from the source (typically /proc/net/tcp or /proc/net/tcp6)
// on host h. Will only work on Linux hosts.
func allTCPListenPorts(h *geneos.Host, ports map[int]int) (err error) {
	for _, source := range tcpfiles {
		tcp, err := h.Open(source)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(tcp)
		if scanner.Scan() {
			// skip headers
			_ = scanner.Text()
			for scanner.Scan() {
				line := scanner.Text()
				fields := strings.Fields(line)
				if len(fields) < 10 || fields[3] != "0A" {
					break
				}
				s := strings.SplitN(fields[1], ":", 2)
				if len(s) != 2 {
					continue
				}
				port, err := strconv.ParseInt(s[1], 16, 32)
				if err != nil {
					continue
				}
				inode, _ := strconv.Atoi(fields[9])
				ports[inode] = int(port)
			}
		}
	}
	return
}

// ListeningPorts returns all TCP ports currently open for the process
// running as the instance. An empty slice is returned if the process
// cannot be found. The instance may be on a remote host.
func ListeningPorts(c geneos.Instance) (ports []int) {
	var err error

	if !IsRunning(c) {
		return
	}

	sockets := sockets(c)
	if len(sockets) == 0 {
		return
	}

	tcpports := make(map[int]int) // key = socket inode
	if err = allTCPListenPorts(c.Host(), tcpports); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Error().Err(err).Msg("continuing")
	}

	for _, s := range sockets {
		if port, ok := tcpports[s]; ok {
			ports = append(ports, port)
			log.Debug().Msgf("process listening on %v", port)
		}
	}
	return
}

// AllListeningPorts returns a sorted list of all listening TCP ports on
// host h between min and max (inclusive). If min or max is -1 then no
// limit is imposed.
func AllListeningPorts(h *geneos.Host, min, max int) (ports []int) {
	var err error

	tcpports := make(map[int]int) // key = socket inode, value port
	if err = allTCPListenPorts(h, tcpports); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Error().Err(err).Msg("continuing")
	}
	for v := range tcpports {
		if min == -1 || v >= min {
			if max == -1 || v <= max {
				ports = append(ports, v)
			}
		}
	}
	sort.Ints(ports)

	return
}

type OpenFiles struct {
	Path   string
	Stat   fs.FileInfo
	FD     string
	FDMode fs.FileMode
}

// Files returns a map of file descriptor (int) to file details
// (InstanceProcFiles) for all open, real, files for the process running
// as the instance. All paths that are not absolute paths are ignored.
// An empty map is returned if the process cannot be found.
func Files(c geneos.Instance) (openfiles map[int]OpenFiles) {
	pid, err := GetPID(c)
	if err != nil {
		return
	}

	file := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := c.Host().ReadDir(file)
	if err != nil {
		return
	}

	openfiles = make(map[int]OpenFiles, len(fds))

	for _, ent := range fds {
		fd := ent.Name()
		dest, err := c.Host().Readlink(path.Join(file, fd))
		if err != nil {
			continue
		}
		if !filepath.IsAbs(dest) {
			continue
		}
		n, _ := strconv.Atoi(fd)

		fdPath := path.Join(file, fd)
		fdMode, err := c.Host().Lstat(fdPath)
		if err != nil {
			continue
		}

		s, err := c.Host().Stat(dest)
		if err != nil {
			continue
		}

		openfiles[n] = OpenFiles{
			Path:   dest,
			Stat:   s,
			FD:     fdPath,
			FDMode: fdMode.Mode(),
		}

		log.Debug().Msgf("\tfd %s points to %q", fd, dest)
	}
	return
}

// sockets returns a map[int]int of file descriptor to socket inode for all open
// files for the process running as the instance. An empty map is
// returned if the process cannot be found.
func sockets(c geneos.Instance) (links map[int]int) {
	var inode int
	links = make(map[int]int)
	pid, err := GetPID(c)
	if err != nil {
		return
	}
	file := fmt.Sprintf("/proc/%d/fd", pid)
	fds, err := c.Host().ReadDir(file)
	if err != nil {
		return
	}
	for _, ent := range fds {
		fd := ent.Name()
		dest, err := c.Host().Readlink(path.Join(file, fd))
		if err != nil {
			continue
		}
		if n, err := fmt.Sscanf(dest, "socket:[%d]", &inode); err == nil && n == 1 {
			f, _ := strconv.Atoi(fd)
			links[f] = inode
			log.Debug().Msgf("\tfd %s points to socket %q", fd, inode)
		}
	}
	return
}
