package includes

import (
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

func auditIncludeFile(h *geneos.Host, action, file string, result error, attrs ...any) {
	if !geneos.AuditActions() {
		return
	}
	var outcome any = "success"
	if result != nil {
		outcome = result
	}
	fields := []any{slog.String("host", h.String()), slog.String("file", file), slog.Any("result", outcome)}
	logger.Audit().Event("action/includes/"+action, append(fields, attrs...)...)
}

func auditIncludeFix(i geneos.Instance, file, backup string, changes map[string]string, result error) {
	if !geneos.AuditActions() {
		return
	}
	for _, location := range slices.Sorted(maps.Keys(changes)) {
		attrs := []any{
			slog.String("file", file),
			slog.String("oldLocation", location),
			slog.String("newLocation", changes[location]),
			slog.String("description", fmt.Sprintf("include location %q -> %q", location, changes[location])),
		}
		if backup != "" {
			attrs = append(attrs, slog.String("backup", backup))
		}
		instance.AuditEvent(i, result, "action/includes/fix", attrs...)
	}
}
