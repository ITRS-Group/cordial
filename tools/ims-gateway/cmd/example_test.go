package cmd

import (
	"os"
	"testing"

	"github.com/itrs-group/cordial/internal/configkeys"
)

// every setting in the example configuration must be one that is read,
// or be listed here as only used by the example itself
func TestExampleSettingsAreAllRead(t *testing.T) {
	b, err := os.ReadFile("../ims-gateway.example.yaml")
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

		"sdp.datacentre",
		"sdp.datacentres.*.auth",
		"sdp.datacentres.*.api",
		"sdp.portal",
		"sdp.client-id",
		"sdp.client-secret",
		"sdp.trace",
		"sdp.requests.search.**",
		"sdp.requests.get.**",
		"sdp.requests.columns",
		"sdp.requests.create.**",
		"sdp.requests.update.edit_request.**",
		"sdp.requests.update.add_note.**",
		"sdp.requests.transform.defaults",
		"sdp.requests.transform.remove",
		"sdp.requests.transform.rename",
		"sdp.requests.transform.exclude",
		"sdp.requests.transform.include",
		"sdp.requests.transform.must-include",
		"sdp.response.created",
		"sdp.response.updated",
		"sdp.response.failed",

		"snow.url",
		"snow.path",
		"snow.username",
		"snow.password",
		"snow.trace",
		"snow.tables",
	}
	// only referred to from templates in the example
	referenced := []string{"sdp.requests.correlation_id"}

	patterns := append(read, referenced...)
	if unread := configkeys.Unread(keys, patterns); len(unread) != 0 {
		t.Errorf("settings in the example that are not read: %v", unread)
	}
	if unused := configkeys.Unused(keys, patterns); len(unused) != 0 {
		t.Errorf("settings listed as read that are not in the example: %v", unused)
	}
	missing, err := configkeys.MissingFromSource(read, ".", "../internal/sdp", "../internal/snow", "../../../pkg/ims", "../../../pkg/ims/snow")
	if err != nil || len(missing) != 0 {
		t.Errorf("settings that are not in the source: %v (%v)", missing, err)
	}
}
