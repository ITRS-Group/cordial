package client

import (
	"os"
	"testing"

	"github.com/itrs-group/cordial/utils/testutil/configkeys"
)

// every setting in the example configuration must be one that is read
func TestExampleSettingsAreAllRead(t *testing.T) {
	b, err := os.ReadFile("../../servicenow2.client.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	keys, err := configkeys.Leaves(b)
	if err != nil {
		t.Fatal(err)
	}

	read := []string{
		"defaults",
		"profiles.*",
		"proxy.url",
		"proxy.authentication.token",
		"proxy.tls.chain",
		"proxy.tls.skip-verify",
		"proxy.default-query",
	}
	if unread := configkeys.Unread(keys, read); len(unread) != 0 {
		t.Errorf("settings in the example that are not read: %v", unread)
	}
	if unused := configkeys.Unused(keys, read); len(unused) != 0 {
		t.Errorf("settings listed as read that are not in the example: %v", unused)
	}
	missing, err := configkeys.MissingFromSource(read, ".", "../../../../pkg/ims")
	if err != nil || len(missing) != 0 {
		t.Errorf("settings that are not in the source: %v (%v)", missing, err)
	}
}
