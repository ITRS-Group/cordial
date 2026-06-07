package restore

import (
	"io"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

type restoreOptions struct {
	compression string
	shared      bool
	list        bool
	names       []string
	component   *geneos.Component
	host        *geneos.Host
	progress    io.Writer
}

var globalRestoreOptions = restoreOptions{
	compression: "gzip",
	names:       []string{"all"},
	progress:    io.Discard,
}

// RestoreOption controls the behaviour of the Restore function
type RestoreOption func(*restoreOptions)

func evalRestoreOptions(options ...RestoreOption) *restoreOptions {
	opts := globalRestoreOptions
	for _, o := range options {
		o(&opts)
	}
	return &opts
}

// Compression sets the compression type to use for backup and restore. Valid values are "gzip", "bzip2", "xz" and "none"
func Compression(compression string) RestoreOption {
	return func(ro *restoreOptions) {
		ro.compression = compression
	}
}

// Shared sets the shared flag for restore, which allows restoring to a
// shared location
func Shared(shared bool) RestoreOption {
	return func(ro *restoreOptions) {
		ro.shared = shared
	}
}

// List sets the list flag for restore, which lists the contents of the
// backup file without restoring
func List(list bool) RestoreOption {
	return func(ro *restoreOptions) {
		ro.list = list
	}
}

// Names sets the names of the components to restore, if not set all
// components will be restored. Names can have the form "NAME" or
// "DEST=NAME", where NAME is the name of the component in the backup
// file and DEST is the name of the component to restore to. If DEST is
// not given, the component will be restored to its original name.
func Names(names ...string) RestoreOption {
	return func(ro *restoreOptions) {
		ro.names = names
	}
}

// Component sets the component to which the restore will be applied, if
// not set the restore will be applied to all component types
func Component(ct *geneos.Component) RestoreOption {
	return func(ro *restoreOptions) {
		ro.component = ct
	}
}

// Host sets the host to which the restore will be applied, if not set
// the local host will be used
func Host(h *geneos.Host) RestoreOption {
	return func(ro *restoreOptions) {
		ro.host = h
	}
}

// ProgressTo sets the writer to which progress will be written during
// restore. If not set, progress will not be written.
func ProgressTo(w io.Writer) RestoreOption {
	return func(ro *restoreOptions) {
		ro.progress = w
	}
}
