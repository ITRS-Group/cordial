package geneos

import (
	"os"
	"path/filepath"
	"time"

	"github.com/DeRuina/timberjack"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/logger"
)

var (
	auditCommands bool
	auditActions  bool
)

// InitGlobalAudit initialises the global audit logging. It returns an
// error if the audit directory cannot be created.
func InitGlobalAudit() (err error) {
	cf := config.Global()

	// TODO: This doesn't work until the directory structure is built,
	// but do not error
	if LocalRoot() == "" {
		return nil
	}

	defaultFile := filepath.Join(LocalRoot(), "audit", "audit.log")
	if err = os.MkdirAll(filepath.Dir(defaultFile), 0750); err != nil {
		return
	}

	// initialise global audit logging after loading global config,
	// which may contain defaults etc.
	logger.NewGlobalAuditLogger(&timberjack.Logger{
		Filename:         config.Get[string](cf, cf.Join("audit", "file"), config.DefaultValue(defaultFile)),
		MaxSize:          config.Get[int](cf, cf.Join("audit", "max-size"), config.DefaultValue(config.DefaultValue(10))),
		MaxAge:           config.Get[int](cf, cf.Join("audit", "max-age"), config.DefaultValue(config.DefaultValue(30))),
		MaxBackups:       config.Get[int](cf, cf.Join("audit", "max-backups"), config.DefaultValue(config.DefaultValue(5))),
		Compression:      config.Get[string](cf, cf.Join("audit", "compression"), config.DefaultValue(config.DefaultValue("gzip"))),
		BackupTimeFormat: "20060102150405",
		RotationInterval: time.Hour * 24 * 7,
		RotateAt:         []string{"00:00"},
	})
	return nil
}

func AuditCommands() bool {
	return auditCommands
}

func AuditActions() bool {
	return auditActions
}

func SetAuditCommands(enabled bool) {
	auditCommands = enabled
}

func SetAuditActions(enabled bool) {
	auditActions = enabled
}
