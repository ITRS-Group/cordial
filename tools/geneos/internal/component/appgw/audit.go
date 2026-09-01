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

package appgw

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

const (
	defaultAuditMaxBytes = 10 * 1024 * 1024 // 10 MiB
	defaultAuditMaxFiles = 5
)

// AuditEvent appends a JSON line to the instance audit log.
//
// Each record is written with fields in a fixed order: timestamp, event,
// username, module, then event-specific details (sorted by key).
func AuditEvent(i geneos.Instance, event string, fields map[string]string) error {
	if i == nil || event == "" {
		return nil
	}

	h := i.Host()
	if h == nil {
		return nil
	}

	ct := i.Type()
	auditPath := auditLogPath(i)
	maxBytes := int64(config.Get[int](config.Global(), config.Join(ct.String(), "audit-max-bytes"), config.DefaultValue(defaultAuditMaxBytes)))
	maxFiles := config.Get[int](config.Global(), config.Join(ct.String(), "audit-max-files"), config.DefaultValue(defaultAuditMaxFiles))
	if maxFiles < 1 {
		maxFiles = defaultAuditMaxFiles
	}

	if err := rotateAuditIfNeeded(h, auditPath, maxBytes, maxFiles); err != nil {
		return err
	}

	line, err := formatAuditRecord(ct.String(), event, h.Username(), fields)
	if err != nil {
		return err
	}

	existing, err := h.ReadFile(auditPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	return h.WriteFile(auditPath, append(existing, line...), 0664)
}

func auditLogFile(module string) string {
	return module + "-audit.log"
}

func auditLogPath(i geneos.Instance) string {
	name := config.Get[string](i.Config(), "audit-log")
	if name == "" {
		name = config.Get[string](config.Global(), config.Join(i.Type().String(), "audit-log-file"), config.DefaultValue(auditLogFile(i.Type().String())))
	}
	return path.Join(i.Home(), path.Base(name))
}

func formatAuditRecord(module, event, username string, fields map[string]string) ([]byte, error) {
	ts := time.Now().UTC().Format(time.RFC3339)

	detailKeys := slices.Collect(maps.Keys(fields))
	detailKeys = slices.DeleteFunc(detailKeys, func(k string) bool { return fields[k] == "" })
	slices.Sort(detailKeys)

	parts := []string{
		`"timestamp":` + jsonLiteral(ts),
		`"event":` + jsonLiteral(event),
	}
	if username != "" {
		parts = append(parts, `"username":`+jsonLiteral(username))
	}
	parts = append(parts, `"module":`+jsonLiteral(module))
	for _, k := range detailKeys {
		parts = append(parts, `"`+k+`":`+auditFieldJSON(k, fields[k]))
	}

	return []byte("{" + strings.Join(parts, ",") + "}\n"), nil
}

func jsonLiteral(v string) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `""`
	}
	return string(b)
}

func auditFieldJSON(key, value string) string {
	switch key {
	case "pid", "bytes", "oldPid", "newPid":
		if n, err := strconv.Atoi(value); err == nil {
			return strconv.Itoa(n)
		}
	}
	return jsonLiteral(value)
}

func rotateAuditIfNeeded(h *geneos.Host, auditPath string, maxBytes int64, maxFiles int) error {
	if maxBytes < 1 {
		return nil
	}

	st, err := h.Stat(auditPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if st.Size() < maxBytes {
		return nil
	}
	return rotateAuditLog(h, auditPath, maxFiles)
}

func rotateAuditLog(h *geneos.Host, auditPath string, maxFiles int) error {
	if maxFiles < 1 {
		maxFiles = defaultAuditMaxFiles
	}

	oldest := fmt.Sprintf("%s.%d", auditPath, maxFiles)
	_ = h.Remove(oldest)

	for i := maxFiles - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", auditPath, i)
		dst := fmt.Sprintf("%s.%d", auditPath, i+1)
		if _, err := h.Stat(src); err == nil {
			if err := h.Rename(src, dst); err != nil {
				return err
			}
		}
	}

	if _, err := h.Stat(auditPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return h.Rename(auditPath, auditPath+".1")
}

func auditImport(i geneos.Instance, dest string) error {
	if i == nil || dest == "" {
		return nil
	}

	yamlPath := instance.HomeRel(i, dest)
	fields := map[string]string{"file": yamlPath}

	if data, err := i.Host().ReadFile(yamlPath); err == nil {
		sum := sha256.Sum256(data)
		fields["bytes"] = fmt.Sprintf("%d", len(data))
		fields["sha256"] = hex.EncodeToString(sum[:])
	}

	return AuditEvent(i, "import", fields)
}
