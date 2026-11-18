package geneos

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
)

// useTestConfigDir points the user configuration directory, and the
// global application configuration directory, at a new temporary
// directory for the test and returns the application directory.
func useTestConfigDir(t *testing.T) string {
	t.Helper()
	readGlobal := func() {
		if _, err := config.Read(cordial.ExecutableName(),
			config.UseGlobal(),
			config.SkipWorkingDir(),
			config.SkipSystemDir(),
			config.UseDefaults(false),
		); err != nil {
			t.Fatal(err)
		}
	}
	// registered before the environment is changed so that it runs
	// after the environment has been restored
	t.Cleanup(readGlobal)

	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	readGlobal()
	if config.AppConfigDir() == "" {
		t.Fatal("no application configuration directory")
	}
	return config.AppConfigDir()
}

// useTestComponents sets the names used for certificates and registers
// the root component, as Init() does, for the test
func useTestComponents(t *testing.T) {
	t.Helper()
	previousSigning, previousChain := SigningCertBasename, DeprecatedChainCertFile
	t.Cleanup(func() {
		SigningCertBasename, DeprecatedChainCertFile = previousSigning, previousChain
		delete(registeredComponents, RootComponentName)
		delete(initDirs, RootComponentName)
	})
	SigningCertBasename = "testsigning"
	DeprecatedChainCertFile = "testchain.pem"
	RootComponent.Register(nil)
}

var caFiles = []string{
	RootCABasename + certs.PEMExtension,
	RootCABasename + certs.KEYExtension,
	"testsigning" + certs.PEMExtension,
	"testsigning" + certs.KEYExtension,
}

func readCAFiles(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	for _, f := range caFiles {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatal(err)
		}
		files[f] = data
	}
	return files
}

func sameCAFiles(t *testing.T, got, want map[string][]byte) {
	t.Helper()
	for f, data := range want {
		if string(got[f]) != string(data) {
			t.Errorf("%s was changed", f)
		}
	}
}

func TestInitialiseNeverRegeneratesCertificates(t *testing.T) {
	confDir := useTestConfigDir(t)
	useTestComponents(t)
	root := filepath.Join(t.TempDir(), "geneos")
	setTestHosts(t, root)

	t.Cleanup(func() {
		globalAuditMutex.Lock()
		defer globalAuditMutex.Unlock()
		if globalAuditWriter != nil {
			globalAuditWriter.Close()
			globalAuditWriter = nil
		}
		logger.NewGlobalAuditLogger(io.Discard)
		config.Delete(config.Global(), cordial.ExecutableName())
	})

	if err := Initialise("init", LOCAL, UseRoot(root)); err != nil {
		t.Fatalf("Initialise() error = %v", err)
	}
	created := readCAFiles(t, confDir)

	// forcing an initialisation, which allows a directory that is not
	// empty, must still keep the user-wide root and signing certificates
	for _, override := range []bool{true, true} {
		if err := Initialise("init", LOCAL, UseRoot(root), Override(override)); err != nil {
			t.Fatalf("Initialise(Override(%v)) error = %v", override, err)
		}
		sameCAFiles(t, readCAFiles(t, confDir), created)
	}

	// only an explicit request replaces them
	if err := TLSInit(LOCALHOST, true, certs.DefaultKeyType); err != nil {
		t.Fatal(err)
	}
	renewed := readCAFiles(t, confDir)
	if string(renewed[RootCABasename+certs.PEMExtension]) == string(created[RootCABasename+certs.PEMExtension]) {
		t.Error("TLSInit(overwrite=true) did not replace the root certificate")
	}
}

func TestTLSInitKeepsExistingRoot(t *testing.T) {
	confDir := useTestConfigDir(t)
	useTestComponents(t)
	setTestHosts(t, t.TempDir())

	if err := TLSInit(LOCALHOST, false, certs.DefaultKeyType); err != nil {
		t.Fatal(err)
	}
	created := readCAFiles(t, confDir)

	t.Run("complete", func(t *testing.T) {
		if err := TLSInit(LOCALHOST, false, certs.DefaultKeyType); err != nil {
			t.Fatal(err)
		}
		sameCAFiles(t, readCAFiles(t, confDir), created)
	})

	t.Run("missing signing certificate", func(t *testing.T) {
		for _, f := range caFiles[2:] {
			if err := os.Remove(filepath.Join(confDir, f)); err != nil {
				t.Fatal(err)
			}
		}
		if err := TLSInit(LOCALHOST, false, certs.DefaultKeyType); err != nil {
			t.Fatal(err)
		}
		recreated := readCAFiles(t, confDir)
		sameCAFiles(t, map[string][]byte{caFiles[0]: recreated[caFiles[0]], caFiles[1]: recreated[caFiles[1]]},
			map[string][]byte{caFiles[0]: created[caFiles[0]], caFiles[1]: created[caFiles[1]]})
		if string(recreated[caFiles[2]]) == string(created[caFiles[2]]) {
			t.Error("signing certificate was not recreated")
		}
		if _, err := readSigningCertificate(); err != nil {
			t.Errorf("new signing certificate is not valid for the root: %v", err)
		}
	})

	t.Run("invalid root", func(t *testing.T) {
		// a signing certificate is not valid as a root
		if err := os.Rename(filepath.Join(confDir, caFiles[2]), filepath.Join(confDir, caFiles[0])); err != nil {
			t.Fatal(err)
		}
		if _, _, err := ReadRootCertificateAndKey(); err == nil {
			t.Fatal("ReadRootCertificateAndKey() accepted a certificate that is not a root CA")
		}
		before := readFileOrEmpty(t, filepath.Join(confDir, caFiles[0]))
		if err := TLSInit(LOCALHOST, false, certs.DefaultKeyType); err == nil {
			t.Error("TLSInit(overwrite=false) replaced an invalid root without being asked to")
		}
		if got := readFileOrEmpty(t, filepath.Join(confDir, caFiles[0])); string(got) != string(before) {
			t.Error("the invalid root certificate was changed")
		}
	})
}

func readFileOrEmpty(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSaveGlobalConfigIsLossless(t *testing.T) {
	confDir := useTestConfigDir(t)

	existing := map[string]any{
		"download": map[string]any{"url": "https://example.test/", "timeout": "5m"},
		"audit":    map[string]any{"max-size": 20, "commands": false},
		"other":    "keep",
		"count":    3,
	}
	data, err := json.Marshal(existing)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(confDir, cordial.ExecutableName()+".json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	config.Set(config.Global(), cordial.ExecutableName(), "/new/root")
	config.Set(config.Global(), "environmentonly", "should not be saved")
	t.Cleanup(func() {
		config.Delete(config.Global(), cordial.ExecutableName())
		config.Delete(config.Global(), "environmentonly")
	})

	if err := SaveGlobalConfig(cordial.ExecutableName(), cordial.ExecutableName()); err != nil {
		t.Fatal(err)
	}

	saved, err := os.ReadFile(filepath.Join(confDir, cordial.ExecutableName()+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(saved, &got); err != nil {
		t.Fatal(err)
	}

	// the test binary name can contain a dot, which is a key delimiter
	var root any = got
	for _, k := range strings.Split(cordial.ExecutableName(), ".") {
		m, _ := root.(map[string]any)
		root = m[k]
	}
	if root != "/new/root" {
		t.Errorf("root = %v, want /new/root", root)
	}
	if _, ok := got["environmentonly"]; ok {
		t.Error("a value that was not asked for was saved")
	}
	if got["other"] != "keep" || got["count"] != float64(3) {
		t.Errorf("scalar values were lost: %v", got)
	}
	if dl, _ := got["download"].(map[string]any); dl["url"] != "https://example.test/" || dl["timeout"] != "5m" {
		t.Errorf("nested download values were lost: %v", got["download"])
	}
	if audit, _ := got["audit"].(map[string]any); audit["max-size"] != float64(20) || audit["commands"] != false {
		t.Errorf("nested audit values with number and boolean types were lost: %v", got["audit"])
	}
}
