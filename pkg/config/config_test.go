package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReadPrecedence(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.defaults.yaml", "a: defaults-file\nb: defaults-file\nd: defaults-file\n")
	writeFile(t, dir, "app.yaml", "a: file\nenv: file\nset: file\n")
	embedded := []byte("a: embedded\nb: embedded\nc: embedded\nd: embedded\ne: embedded\n")

	from := New()
	for _, k := range []string{"a", "b", "c", "d", "e", "f"} {
		Set(from, k, "from")
	}

	t.Setenv("CORDIALPRECEDENCE_ENV", "environment")

	cf, err := Read("app",
		FromDir(dir),
		Format("yaml"),
		WithDefaults(embedded, "yaml"),
		DefaultsFrom(from),
		WithEnvs("CORDIALPRECEDENCE", "_"),
	)
	if err != nil {
		t.Fatal(err)
	}
	Set(cf, "set", "explicit")

	want := map[string]string{
		"a":   "file",
		"b":   "defaults-file",
		"c":   "embedded",
		"d":   "defaults-file",
		"e":   "embedded",
		"f":   "from",
		"env": "environment",
		"set": "explicit",
	}
	for k, v := range want {
		if got := Get[string](cf, k); got != v {
			t.Errorf("Get(%q) = %q, want %q", k, got, v)
		}
	}
}

func TestReadUseDefaultsFalse(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.defaults.yaml", "fromfile: yes\n")
	writeFile(t, dir, "app.yaml", "main: yes\n")

	from := New()
	Set(from, "fromcf", "yes")

	cf, err := Read("app",
		FromDir(dir),
		Format("yaml"),
		UseDefaults(false),
		WithDefaults([]byte("embedded: yes\n"), "yaml"),
		DefaultsFrom(from),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !cf.IsSet("main") {
		t.Error("main configuration not loaded")
	}
	for _, k := range []string{"embedded", "fromfile", "fromcf"} {
		if cf.IsSet(k) {
			t.Errorf("%q is set although defaults are disabled", k)
		}
	}

	// the same applies to the global configuration
	old := Global()
	t.Cleanup(func() {
		globalMutex.Lock()
		global = old
		globalMutex.Unlock()
	})
	g, err := Read("app", UseGlobal(), FromDir(dir), Format("yaml"), UseDefaults(false), WithDefaults([]byte("embedded: yes\n"), "yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if g.IsSet("embedded") {
		t.Error("global config has embedded defaults although defaults are disabled")
	}
	if got := g.ConfigType(); got != "yaml" {
		t.Errorf("global ConfigType() = %q, want yaml", got)
	}
	if g != Global() {
		t.Error("Read(UseGlobal()) did not return the global config")
	}
}

func TestReadAlwaysReturnsConfig(t *testing.T) {
	dir := t.TempDir()
	bad := writeFile(t, dir, "bad.yaml", "a: [unterminated\n")
	writeFile(t, dir, "dir.yaml", "a: [unterminated\n")

	tests := []struct {
		name    string
		options []FileOption
	}{
		{name: "bad file path", options: []FileOption{FilePath(bad)}},
		{name: "missing but must exist", options: []FileOption{FilePath(filepath.Join(dir, "missing.yaml")), MustExist()}},
		{name: "bad file in directory", options: []FileOption{FromDir(dir), Format("yaml")}},
		{name: "bad embedded defaults", options: []FileOption{FromDir(dir), WithDefaults([]byte("a: [x\n"), "yaml"), StopOnInternalDefaultsErrors()}},
		{name: "bad reader", options: []FileOption{Reader(strings.NewReader("a: [x\n")), Format("yaml")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cf, err := Read("dir", append(test.options, SkipWorkingDir(), SkipSystemDir(), SkipUserConfDir())...)
			if err == nil {
				t.Error("expected an error")
			}
			if cf == nil {
				t.Fatal("Read() returned a nil config with the error")
			}
		})
	}
}

func TestReadWatchConfigWithFilePath(t *testing.T) {
	dir := t.TempDir()
	file := writeFile(t, dir, "watched.yaml", "a: 1\n")

	changed := make(chan struct{}, 8)
	cf, err := Read("watched",
		FilePath(file),
		UseDefaults(false),
		WatchConfig(func(fsnotify.Event) {
			select {
			case changed <- struct{}{}:
			default:
			}
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := cf.ConfigFileUsed(); got != file {
		t.Fatalf("ConfigFileUsed() = %q, want %q", got, file)
	}

	// give the watcher goroutine time to start and then keep changing
	// the file until the notification is seen
	deadline := time.After(10 * time.Second)
	for i := 0; ; i++ {
		time.Sleep(100 * time.Millisecond)
		writeFile(t, dir, "watched.yaml", "a: "+strconv.Itoa(i+2)+"\n")
		select {
		case <-changed:
			return
		case <-deadline:
			t.Fatal("no change notification for a file set with FilePath()")
		default:
		}
	}
}

func TestLoadOptionsWhenUserConfigDirFails(t *testing.T) {
	old := userConfigPath
	userConfigPath = func(...string) (string, error) { return "", ErrNoUserConfigDir }
	t.Cleanup(func() { userConfigPath = old })

	opts := evalLoadOptions("module", FilePath("/x/y/module.yaml"))
	if opts.format != "yaml" {
		t.Errorf("format = %q, want yaml", opts.format)
	}
	if opts.userConfDir != "" {
		t.Errorf("userConfDir = %q, want empty", opts.userConfDir)
	}
	if opts.appName != "module" {
		t.Errorf("appName = %q, want module", opts.appName)
	}

	opts = evalLoadOptions("module")
	if opts.format != defaultFileFormat {
		t.Errorf("format = %q, want %q", opts.format, defaultFileFormat)
	}
	if got := Path("module", SkipWorkingDir(), SkipSystemDir()); strings.HasSuffix(got, "module.") || got == "" {
		t.Errorf("Path() = %q, want a file name with an extension", got)
	}
}

func TestWithEnvsEmptyDelimiter(t *testing.T) {
	t.Setenv("CORDIALENVS_SERVER_PORT", "8443")
	cf, err := Read("envs", Reader(strings.NewReader("server:\n  port: 1\n")), Format("yaml"), WithEnvs("CORDIALENVS", ""))
	if err != nil {
		t.Fatal(err)
	}
	// the environment overrides the value in the configuration
	if got := Get[string](cf, "server.port"); got != "8443" {
		t.Errorf("Get() = %q, want value from environment", got)
	}
}

func TestSubKeepsDelimiter(t *testing.T) {
	cf := New(KeyDelimiter("::"))
	Set(cf, "top::one", "1")

	for _, key := range []string{"top", "missing"} {
		sub := cf.Sub(key)
		Set(sub, sub.Join("a", "b"), "x")
		if got := Get[string](sub, "a::b"); got != "x" {
			t.Errorf("Sub(%q): Get(a::b) = %q, want x", key, got)
		}
		if keys := sub.AllKeys(); !contains(keys, "a::b") {
			t.Errorf("Sub(%q): keys = %v, want a::b stored as a nested key", key, keys)
		}
	}
}

func contains(s []string, v string) bool {
	for _, i := range s {
		if i == v {
			return true
		}
	}
	return false
}

func TestSetMapsWithMixedCaseKeys(t *testing.T) {
	cf := New()
	for range 2 {
		Set(cf, "m", map[string]string{"Foo": "1", "Bar": "2"})
		got := Get[map[string]string](cf, "m")
		if len(got) != 2 || got["foo"] != "1" || got["bar"] != "2" {
			t.Fatalf("map[string]string = %v, want both keys kept", got)
		}
	}
	Set(cf, "m", map[string]string{"Foo": "1"})
	if got := Get[map[string]string](cf, "m"); len(got) != 1 || got["foo"] != "1" {
		t.Errorf("map[string]string after removing a key = %v", got)
	}

	for range 2 {
		Set(cf, "a", map[string]any{"Foo": 1, "Bar": "x"})
		got := Get[map[string]any](cf, "a")
		if len(got) != 2 || got["foo"] == nil || got["bar"] != "x" {
			t.Fatalf("map[string]any = %v, want both keys kept", got)
		}
	}
}

func TestSetDoesNotModifyCallersSlice(t *testing.T) {
	cf := New()
	Set(cf, "home", "/opt/geneos")
	in := []string{"/opt/geneos/a", "b"}
	Set(cf, "list", in, Replace("home"))
	if in[0] != "/opt/geneos/a" {
		t.Errorf("caller's slice modified: %v", in)
	}
	if got := Get[[]string](cf, "list", NoExpand()); got[0] != "${config:home}/a" {
		t.Errorf("stored list = %v, want home replaced", got)
	}
}

func TestReplaceEdgeCases(t *testing.T) {
	cf := New()
	Set(cf, "home", "/opt/geneos")
	Set(cf, "empty", "")

	tests := []struct {
		name  string
		value string
		names []string
		want  string
	}{
		{name: "simple", value: "/opt/geneos/x", names: []string{"home"}, want: "${config:home}/x"},
		{name: "text after an expand is replaced", value: "${config:a}/opt/geneos/x", names: []string{"home"}, want: "${config:a}${config:home}/x"},
		{name: "text inside an expand is kept", value: "${env:/opt/geneos}/opt/geneos", names: []string{"home"}, want: "${env:/opt/geneos}${config:home}"},
		{name: "closing brace before an expand", value: "a} ${config:b}/opt/geneos", names: []string{"home"}, want: "a} ${config:b}${config:home}"},
		{name: "unterminated expand", value: "/opt/geneos ${config:b", names: []string{"home"}, want: "${config:home} ${config:b"},
		{name: "empty value is ignored", value: "abc", names: []string{"empty"}, want: "abc"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var options []ExpandOption
			for _, n := range test.names {
				options = append(options, Replace(n))
			}
			opts := evalExpandOptions(cf, options...)
			if got := cf.replaceStringParam(test.value, opts); got != test.want {
				t.Errorf("replaceStringParam(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestSetKeyValuePairs(t *testing.T) {
	cf := New()
	if err := cf.SetKeyValuePairs("a=1", "b==value", "c=", "d+=x", "d+=-y", "e+z"); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{"a": "1", "b": "=value", "c": "", "d": "x -y", "e": "z"} {
		if got, _ := Lookup[string](cf, k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if err := cf.SetKeyValuePairs("novalue"); err == nil {
		t.Error("item without a separator did not return an error")
	}
}

func TestPromoteFromStoresRawValue(t *testing.T) {
	t.Setenv("CORDIAL_TEST_PROMOTE", "expanded")
	cf := New()
	Set(cf, "old", "${env:CORDIAL_TEST_PROMOTE}")

	if got := Get[string](cf, "new", PromoteFrom("old")); got != "expanded" {
		t.Errorf("Get() = %q, want expanded value", got)
	}
	if got := Get[string](cf, "new", NoExpand()); got != "${env:CORDIAL_TEST_PROMOTE}" {
		t.Errorf("promoted value stored as %q, want the original unexpanded", got)
	}
	if cf.IsSet("old") {
		t.Error("legacy key not removed")
	}
}

func TestPromoteFromConcurrent(t *testing.T) {
	cf := New()
	Set(cf, "old", "v")
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 100 {
				if got := Get[string](cf, "new", PromoteFrom("old")); got != "v" {
					t.Errorf("Get() = %q, want v", got)
					return
				}
				_ = Get[string](cf, "other")
			}
		})
	}
	wg.Wait()
}

type namedString string

func TestUnmarshalDecoderOptions(t *testing.T) {
	t.Setenv("CORDIAL_TEST_DECODE", "from-env")

	t.Run("named string types", func(t *testing.T) {
		cf := New()
		Set(cf, "n", namedString("${env:CORDIAL_TEST_DECODE}"))
		if got := Get[namedString](cf, "n"); got != "from-env" {
			t.Errorf("Get[namedString] = %q, want expanded", got)
		}
	})

	t.Run("default options of the config", func(t *testing.T) {
		cf := New(SetDefaultExpandOptions(NoExpand()))
		Set(cf, "m", map[string]any{"a": "${env:CORDIAL_TEST_DECODE}"})
		if got := Get[map[string]string](cf, "m"); got["a"] != "${env:CORDIAL_TEST_DECODE}" {
			t.Errorf("map value = %q, want unexpanded because of the config's default NoExpand", got["a"])
		}
		cf.SetDefaultExpandOptions()
		if got := Get[map[string]string](cf, "m"); got["a"] != "from-env" {
			t.Errorf("map value = %q, want expanded", got["a"])
		}
	})

	t.Run("callers options are not modified", func(t *testing.T) {
		cf := New()
		Set(cf, "m", map[string]any{"a": "1"})
		options := make([]ExpandOption, 0, 4)
		var out map[string]string
		if err := cf.UnmarshalKey("m", &out, options...); err != nil {
			t.Fatal(err)
		}
		if options[:1][0] != nil {
			t.Error("UnmarshalKey wrote to the caller's options slice")
		}
	})
}

func TestIgnoreKeys(t *testing.T) {
	tests := []struct {
		name   string
		ignore []string
		want   map[string]bool // key -> written
	}{
		{name: "leaf", ignore: []string{"a.b"}, want: map[string]bool{"a.b": false, "a.c": true, "x": true}},
		{name: "case insensitive", ignore: []string{"A.B", "X"}, want: map[string]bool{"a.b": false, "a.c": true, "x": false}},
		{name: "parent", ignore: []string{"a"}, want: map[string]bool{"a.b": false, "a.c": false, "ab": true, "x": true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cf := New()
			for k := range test.want {
				Set(cf, k, "v")
			}
			file := filepath.Join(t.TempDir(), "out.json")
			if err := cf.Write("out", FilePath(file), IgnoreKeys(test.ignore...)); err != nil {
				t.Fatal(err)
			}
			out, err := Read("out", FilePath(file), UseDefaults(false), MustExist())
			if err != nil {
				t.Fatal(err)
			}
			for k, written := range test.want {
				if out.IsSet(k) != written {
					t.Errorf("key %q: written = %v, want %v", k, out.IsSet(k), written)
				}
			}
		})
	}
}

func TestGlobalAccessIsRaceFree(t *testing.T) {
	old := Global()
	t.Cleanup(func() {
		globalMutex.Lock()
		global = old
		globalMutex.Unlock()
	})

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 100 {
				if got := Join("a", "b"); got != "a.b" {
					t.Errorf("Join() = %q", got)
					return
				}
				_ = AppConfigDir()
				_ = Global()
				_ = ExpandStringSlice([]string{"x"})
			}
		})
	}
	wg.Go(func() {
		for range 20 {
			ResetConfig()
		}
	})
	wg.Wait()
}

func TestNewDoesNotShareState(t *testing.T) {
	a := New()
	b := New()
	Set(a, "k", "v")
	if b.IsSet("k") {
		t.Error("settings shared between Config objects")
	}
}
