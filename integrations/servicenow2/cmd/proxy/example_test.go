package proxy

import (
	"os"
	"testing"

	"github.com/itrs-group/cordial/internal/configkeys"
)

// every setting in the example configuration must be one that is read
func TestExampleSettingsAreAllRead(t *testing.T) {
	b, err := os.ReadFile("../../servicenow2.proxy.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	keys, err := configkeys.Leaves(b)
	if err != nil {
		t.Fatal(err)
	}

	read := []string{
		"server.listen",
		"server.path",
		"server.tls.enabled",
		"server.tls.certificate",
		"server.tls.private-key",
		"server.authentication.token",
		"servicenow.url",
		"servicenow.path",
		"servicenow.username",
		"servicenow.password",
		"servicenow.trace",
		"servicenow.tables",
	}
	if unread := configkeys.Unread(keys, read); len(unread) != 0 {
		t.Errorf("settings in the example that are not read: %v", unread)
	}
	if unused := configkeys.Unused(keys, read); len(unused) != 0 {
		t.Errorf("settings listed as read that are not in the example: %v", unused)
	}
	missing, err := configkeys.MissingFromSource(read, ".", "../../../../pkg/ims", "../../../../pkg/ims/snow")
	if err != nil || len(missing) != 0 {
		t.Errorf("settings that are not in the source: %v (%v)", missing, err)
	}
}
