package instance

import (
	"log/slog"

	"github.com/itrs-group/cordial/pkg/logger"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var log = logger.Logger

// Logger returns a logger with the instance name, host and type in the
// context. The logger is configured with the "logger" prefix and an
// indent, and is set to the Info level.
func Logger(i geneos.Instance, groups ...string) (l *slog.Logger) {
	l = slog.New(logger.LogHandler)
	for _, group := range groups {
		l = l.WithGroup(group)
	}
	return l.With(LogGroup(i))
}

// LogGroup returns a slog attribute group containing the instance's
// component, name, and host.
func LogGroup(i geneos.Instance) slog.Attr {
	return slog.Group("instance",
		slog.String("component", i.Type().String()),
		slog.String("name", i.Name()),
		slog.String("host", i.Host().String()),
	)
}
