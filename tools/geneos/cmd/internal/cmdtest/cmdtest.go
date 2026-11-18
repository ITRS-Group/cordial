// Package cmdtest provides helpers for running `geneos` commands
// in-process, against a temporary Geneos root, from tests.
//
// It is only intended to be imported from _test.go files.
package cmdtest

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// StderrCaptured is true if everything written to standard error is
// captured, including by code that saved the original os.Stderr.
// Tests of error output should be skipped if it is false.
const StderrCaptured = stderrCaptured

// Result is the outcome of running a command.
type Result struct {
	Stdout string
	Stderr string
	Err    error
}

// Env isolates the test from the real user and global configuration
// and returns the path to an empty Geneos root directory. The process
// global state changed by running commands is restored when the test
// finishes.
func Env(t *testing.T) (root string) {
	t.Helper()

	home := t.TempDir()
	root = filepath.Join(t.TempDir(), "geneos")
	if err := os.MkdirAll(root, 0o775); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("GENEOS_HOME", root)
	t.Setenv("ITRS_HOME", "")
	t.Setenv("CORDIAL_LOG_OUTPUT", "")

	key := cordial.ExecutableName()
	previousRoot := config.Get[string](config.Global(), key)
	previousLocal, previousAll, previousUnknown, previousHostname := geneos.LOCAL, geneos.ALL, geneos.UNKNOWN, geneos.LOCALHOST
	previousHandler := logger.LogHandler
	t.Cleanup(func() {
		// instances are cached by name, so forget those from this root
		if geneos.LOCAL != nil {
			for _, i := range instance.Instances(geneos.LOCAL, nil) {
				_ = i.Unload()
			}
		}
		config.Set(config.Global(), key, previousRoot)
		geneos.LOCAL, geneos.ALL, geneos.UNKNOWN, geneos.LOCALHOST = previousLocal, previousAll, previousUnknown, previousHostname
		logger.LogHandler = previousHandler
	})
	config.Set(config.Global(), key, root)
	return
}

// AddHost creates a remote host called name, with the given key/value
// configuration pairs, in the host configuration file of the isolated
// user configuration. It does not connect to the host.
func AddHost(t *testing.T, name string, kv ...string) {
	t.Helper()

	geneos.InitHosts(cordial.ExecutableName())
	h := geneos.NewHost(name)
	for i := 0; i+1 < len(kv); i += 2 {
		config.Set(h.Config, kv[i], kv[i+1])
	}
	h.Valid()
	if err := geneos.SaveHostConfig(); err != nil {
		t.Fatal(err)
	}
}

// Run executes the root command with args, capturing os.Stdout and
// os.Stderr, and restores all the flags of all the commands to their
// previous values afterwards so that tests do not influence each other.
func Run(t *testing.T, root *cobra.Command, args ...string) (res Result) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	restoreFlags := snapshotFlags(root)
	restore := capture(t, &stdout, &stderr)
	defer func() {
		restore()
		res.Stdout, res.Stderr = stdout.String(), stderr.String()
		restoreFlags()
	}()

	root.SetArgs(args)
	res.Err = root.Execute()
	return
}

func capture(t *testing.T, stdout, stderr io.Writer) (restore func()) {
	t.Helper()

	type pipe struct {
		r, w *os.File
		done chan struct{}
	}
	start := func(dst io.Writer) *pipe {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		p := &pipe{r: r, w: w, done: make(chan struct{})}
		go func() {
			defer close(p.done)
			_, _ = io.Copy(dst, r)
		}()
		return p
	}

	oldOut, oldErr := os.Stdout, os.Stderr
	po, pe := start(stdout), start(stderr)
	os.Stdout, os.Stderr = po.w, pe.w
	restoreFd := redirectStderrFd(pe.w)

	return func() {
		restoreFd()
		os.Stdout, os.Stderr = oldOut, oldErr
		for _, p := range []*pipe{po, pe} {
			p.w.Close()
			<-p.done
			p.r.Close()
		}
	}
}

// snapshotFlags records the values of the flags of c and its
// sub-commands and returns a function that restores them, so that
// tests do not influence each other.
func snapshotFlags(c *cobra.Command) (restore func()) {
	var restores []func()

	save := func(f *pflag.Flag) {
		changed := f.Changed
		restores = append(restores, func() { f.Changed = changed })

		if sv, ok := f.Value.(pflag.SliceValue); ok {
			values := slices.Clone(sv.GetSlice())
			restores = append(restores, func() { _ = sv.Replace(values) })
			return
		}
		// flag values are pointers to the variables they set
		if v := reflect.ValueOf(f.Value); v.Kind() == reflect.Pointer && !v.IsNil() {
			elem := v.Elem()
			saved := reflect.New(elem.Type()).Elem()
			saved.Set(elem)
			restores = append(restores, func() { elem.Set(saved) })
		}
	}

	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		c.Flags().VisitAll(save)
		c.PersistentFlags().VisitAll(save)
		for _, child := range c.Commands() {
			walk(child)
		}
	}
	walk(c)

	return func() {
		for _, r := range slices.Backward(restores) {
			r()
		}
	}
}
