package geneos

import (
	"testing"

	"github.com/itrs-group/cordial/pkg/config"
)

func TestAuditWriterSettings(t *testing.T) {
	instance := config.New()
	config.Set(instance, "audit.max-size", 50)
	config.Set(instance, "audit.compression", "zstd")

	global := config.New()
	config.Set(global, "audit.max-age", 7)
	config.Set(global, "audit.max-size", 99)
	config.Set(global, "audit.max-backups", 2)

	noCompression := config.New()
	config.Set(noCompression, "audit.compression", false)

	tests := []struct {
		name        string
		cfs         []*config.Config
		size, age   int
		backups     int
		compression string
	}{
		{"defaults", nil, 10, 30, 5, "gzip"},
		{"nil configs", []*config.Config{nil}, 10, 30, 5, "gzip"},
		{"global only", []*config.Config{global}, 99, 7, 2, "gzip"},
		{"instance then global", []*config.Config{instance, global}, 50, 7, 2, "zstd"},
		{"global then instance", []*config.Config{global, instance}, 99, 7, 2, "zstd"},
		{"compression disabled", []*config.Config{noCompression}, 10, 30, 5, "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := AuditWriter("audit.log", tt.cfs...)
			if w.Filename != "audit.log" {
				t.Errorf("Filename = %q", w.Filename)
			}
			if w.MaxSize != tt.size || w.MaxAge != tt.age || w.MaxBackups != tt.backups || w.Compression != tt.compression {
				t.Errorf("settings = %d MiB, %d days, %d backups, %q; want %d, %d, %d, %q",
					w.MaxSize, w.MaxAge, w.MaxBackups, w.Compression,
					tt.size, tt.age, tt.backups, tt.compression)
			}
		})
	}
}

func TestAuditCompression(t *testing.T) {
	for value, want := range map[string]string{
		"":        "gzip",
		"gzip":    "gzip",
		"GZIP":    "gzip",
		"true":    "gzip",
		"zstd":    "zstd",
		"none":    "none",
		"false":   "none",
		"unknown": "gzip",
	} {
		if got := auditCompression(value); got != want {
			t.Errorf("auditCompression(%q) = %q, want %q", value, got, want)
		}
	}
}
