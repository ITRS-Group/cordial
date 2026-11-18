package cmd

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
)

func TestLogLevel(t *testing.T) {
	oldDebug, oldTrace := debug, trace
	t.Cleanup(func() { debug, trace = oldDebug, oldTrace })

	for _, test := range []struct {
		name         string
		debug, trace bool
		want         slog.Level
	}{
		{"default", false, false, slog.LevelInfo},
		{"debug", true, false, slog.LevelDebug},
		{"trace", false, true, levelTrace},
		{"trace implies debug", true, true, levelTrace},
	} {
		t.Run(test.name, func(t *testing.T) {
			debug, trace = test.debug, test.trace
			if got := logLevel(); got != test.want {
				t.Errorf("logLevel() = %v, want %v", got, test.want)
			}
			if test.trace && levelTrace >= slog.LevelDebug {
				t.Errorf("trace level %v is not below debug", levelTrace)
			}
		})
	}
}

// the flags must really set the level of the log
func TestInitConfigSetsLogLevel(t *testing.T) {
	dir := t.TempDir()
	oldCf, oldLogFile, oldDebug, oldTrace, oldLog := cf, logFile, debug, trace, log
	t.Cleanup(func() {
		cf, logFile, debug, trace, log = oldCf, oldLogFile, oldDebug, oldTrace, oldLog
		logger.LogLevel.Set(slog.LevelInfo)
	})

	for _, test := range []struct {
		name         string
		debug, trace bool
		want         slog.Level
		wantDebug    bool
		wantQuery    bool
	}{
		{"default", false, false, slog.LevelInfo, false, false},
		{"-d", true, false, slog.LevelDebug, true, false},
		{"-t", false, true, levelTrace, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			cf = testConfig(t, dir, "gdna:\n  schedule: \"0 0 1 1 *\"\n")
			logFile = filepath.Join(dir, "gdna.log")
			debug, trace = test.debug, test.trace
			os.Remove(logFile)

			initConfig(&cobra.Command{Use: "test"})
			if got := logger.LogLevel.Level(); got != test.want {
				t.Fatalf("log level = %v, want %v", got, test.want)
			}

			traceQuery(t.Context(), "SELECT 1")
			log.Debug("a debug message")

			b, err := os.ReadFile(logFile)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(string(b), "a debug message"); got != test.wantDebug {
				t.Errorf("debug message logged = %t, want %t", got, test.wantDebug)
			}
			if got := strings.Contains(string(b), "query trace"); got != test.wantQuery {
				t.Errorf("query trace logged = %t, want %t", got, test.wantQuery)
			}
		})
	}
}

func TestLogMaxAge(t *testing.T) {
	for _, test := range []struct {
		name string
		yaml string
		want int
	}{
		// the default is the documented one
		{"default", "", 14},
		{"documented max-age", "gdna:\n  log:\n    max-age: 7\n", 7},
		{"deprecated stale-after is still used if set", "gdna:\n  log:\n    stale-after: 30\n", 30},
	} {
		t.Run(test.name, func(t *testing.T) {
			cf := testConfig(t, t.TempDir(), test.yaml)
			if got := logMaxAge(cf); got != test.want {
				t.Errorf("logMaxAge() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestSMTPTimeout(t *testing.T) {
	for _, test := range []struct {
		name string
		yaml string
		want time.Duration
	}{
		{"default", "", 10 * time.Second},
		{"documented email.timeout", "email:\n  timeout: 45s\n", 45 * time.Second},
		{"legacy _smtp_timeout is still used if set", "_smtp_timeout: 5\n", 5 * time.Second},
		{"never zero", "email:\n  timeout: 0s\n", 10 * time.Second},
	} {
		t.Run(test.name, func(t *testing.T) {
			cf := testConfig(t, t.TempDir(), test.yaml)
			if got := smtpTimeout(cf); got != test.want {
				t.Errorf("smtpTimeout() = %v, want %v", got, test.want)
			}
		})
	}
}

// the defaults shown in the help of the flags must be those of the
// configuration, and a port is not limited to a signed 16-bit value
func TestNetprobeFlagDefaultsMatchConfig(t *testing.T) {
	cf := config.New(config.WithDefaults(defaults, "yaml"))

	for _, test := range []struct {
		cmd *cobra.Command
		tls string
	}{
		{startCmd, "secure"},
		{reportCmd, "tls"},
	} {
		t.Run(test.cmd.Name(), func(t *testing.T) {
			for _, f := range []struct{ flag, key string }{
				{"hostname", "geneos.netprobe.hostname"},
				{"port", "geneos.netprobe.port"},
				{test.tls, "geneos.netprobe.secure"},
				{"skip-verify", "geneos.netprobe.skip-verify"},
				{"entity", "geneos.entity"},
				{"sampler", "geneos.sampler"},
			} {
				flag := test.cmd.Flags().Lookup(f.flag)
				if flag == nil {
					t.Fatalf("no flag %q", f.flag)
				}
				if want := fmt.Sprint(config.Get[any](cf, f.key)); flag.DefValue != want {
					t.Errorf("--%s default = %q, want the configuration default %q for %s", f.flag, flag.DefValue, want, f.key)
				}
			}

			t.Cleanup(func() {
				test.cmd.Flags().VisitAll(func(f *pflag.Flag) {
					f.Value.Set(f.DefValue)
					f.Changed = false
				})
				fetchCmdSources = nil // setting the default of this flag adds an empty source
			})
			if err := test.cmd.Flags().Parse([]string{"--port", "40000"}); err != nil {
				t.Errorf("--port 40000: %v", err)
			}
			if netprobePort != 40000 {
				t.Errorf("port = %d, want 40000", netprobePort)
			}
		})
	}
}

// walkKeys returns the full name of every setting in a nested map
func walkKeys(prefix string, m map[string]any) (keys []string) {
	for k, v := range m {
		name := k
		if prefix != "" {
			name = prefix + "." + k
		}
		if sub, ok := v.(map[string]any); ok && len(sub) > 0 {
			keys = append(keys, walkKeys(name, sub)...)
			continue
		}
		keys = append(keys, name)
	}
	slices.Sort(keys)
	return
}

// documentedOnly are the settings in the shipped defaults and example
// that are not read by the program in Go code, with the reason
var documentedOnly = map[string]string{
	"gdna.site-name": "only used in the queries of the summary report",
}

// every setting in the defaults and the example configuration has to be
// read by the Go code, directly or by one of the sections that are only
// used in queries, or be listed above as documented only, so that a
// setting that is documented but does nothing is found
func TestShippedSettingsAreRead(t *testing.T) {
	var source strings.Builder
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		source.Write(b)
	}
	code := source.String()

	example, err := os.ReadFile(filepath.Join("..", "gdna.example.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	// sections whose settings are used in the queries and not in Go code
	queryOnly := []string{"db.", "reports.", "plugins.", "filters.", "dashboards.", "schema-updates."}

	read := func(key string) bool {
		// the key as one string, or as the arguments of a Join()
		parts := strings.Split(key, ".")
		for i := range parts {
			parts[i] = regexp.QuoteMeta(parts[i])
		}
		joined := `"` + strings.Join(parts, `\.`) + `"`
		args := `"` + strings.Join(parts, `"\s*,\s*"`) + `"`
		// a section read as a whole, such as a list or a map
		for n := len(parts) - 1; n >= 1; n-- {
			prefix := `"` + strings.Join(parts[:n], `\.`) + `"`
			if n == 1 {
				continue
			}
			if regexp.MustCompile(prefix+`[,)]`).MatchString(code) && regexp.MustCompile(`"`+parts[n]+`"`).MatchString(code) {
				return true
			}
		}
		return regexp.MustCompile(joined).MatchString(code) || regexp.MustCompile(args).MatchString(code)
	}

	for name, yaml := range map[string][]byte{"defaults": defaults, "example": example} {
		cf := config.New(config.WithDefaults(yaml, "yaml"))
		for _, key := range walkKeys("", cf.AllSettings()) {
			if _, ok := documentedOnly[key]; ok || slices.ContainsFunc(queryOnly, func(p string) bool { return strings.HasPrefix(key, p) }) {
				continue
			}
			if !read(key) {
				t.Errorf("%s configuration setting %q is not read by the program and is not documented only", name, key)
			}
		}
	}
}

func TestPrintPublicKey(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("HOME", configDir)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	// the key is in the setting, or is in the file of the default key
	// if the setting is empty, as it is in the default configuration
	oldCf := cf
	t.Cleanup(func() { cf = oldCf })
	quietLog(t)
	cf = testConfig(t, t.TempDir(), "")

	var out bytes.Buffer
	if err = printPublicKey(&out); err == nil {
		t.Error("printPublicKey succeeded with no key set and no default key file")
	}

	defaultKey := filepath.Join(configDir, "geneos", "gdna-private-key.pem")
	if err = os.MkdirAll(filepath.Dir(defaultKey), 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(defaultKey, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err = printPublicKey(&out); err != nil || !strings.Contains(out.String(), "PUBLIC KEY") {
		t.Errorf("printPublicKey with the default key file = %q, %v", out.String(), err)
	}

	// the setting is used before the default, as the key or as a path
	other := filepath.Join(t.TempDir(), "other.pem")
	os.WriteFile(other, keyPEM, 0o600)
	for name, setting := range map[string]string{"path": other, "key": string(keyPEM)} {
		os.Remove(defaultKey)
		config.Set(cf, "gdna.licd-private-key", setting)
		out.Reset()
		if err = printPublicKey(&out); err != nil || !strings.Contains(out.String(), "PUBLIC KEY") {
			t.Errorf("printPublicKey with the key set as a %s = %q, %v", name, out.String(), err)
		}
	}
}
