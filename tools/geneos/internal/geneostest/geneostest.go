/*
Copyright © 2026 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package geneostest provides helpers for tests that need a temporary
// Geneos installation root on the local host. It must only be used from
// tests.
package geneostest

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// NewRoot sets up a temporary Geneos root directory for the test, with
// the local host named hostname (`localhost` if empty), and returns the
// directory. For each of packages an empty `active_prod` package
// directory is created. The global configuration, hosts, log handler
// and any instances loaded are restored or unloaded when the test
// finishes.
func NewRoot(t testing.TB, hostname string, packages ...string) string {
	t.Helper()

	if hostname == "" {
		hostname = "localhost"
	}

	root := t.TempDir()
	key := cordial.ExecutableName()
	previousRoot := config.Get[string](config.Global(), key)
	previousLocal, previousAll, previousHostname := geneos.LOCAL, geneos.ALL, geneos.LOCALHOST
	previousHandler := logger.LogHandler
	t.Cleanup(func() {
		for _, i := range instance.Instances(geneos.LOCAL, nil) {
			_ = i.Unload()
		}
		// profile instances are not included in all instances
		if pct := geneos.ParseComponent("profile"); pct != nil {
			for _, i := range instance.Instances(geneos.LOCAL, pct) {
				_ = i.Unload()
			}
		}
		config.Set(config.Global(), key, previousRoot)
		geneos.LOCAL, geneos.ALL, geneos.LOCALHOST = previousLocal, previousAll, previousHostname
		logger.LogHandler = previousHandler
	})

	config.Set(config.Global(), key, root)
	logger.LogHandler = slog.NewTextHandler(io.Discard, nil)
	geneos.LOCALHOST = hostname
	geneos.LOCAL = nil
	geneos.LOCAL = geneos.NewHost(geneos.LOCALHOST)
	geneos.ALL = geneos.NewHost(geneos.ALLHOSTS)

	for _, pkg := range packages {
		if err := os.MkdirAll(filepath.Join(root, "packages", pkg, "active_prod"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
