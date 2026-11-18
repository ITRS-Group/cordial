package cmd

import (
	"os"
	"testing"

	"github.com/itrs-group/cordial/internal/configkeys"
)

// every setting in the example configuration must be one that is read
func TestExampleSettingsAreAllRead(t *testing.T) {
	b, err := os.ReadFile("../servicenow.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	keys, err := configkeys.Leaves(b)
	if err != nil {
		t.Fatal(err)
	}

	read := []string{
		"api.apikey",
		"api.host",
		"api.port",
		"servicenow.geneosseveritymap.*",
		"servicenow.incident-query",
		"servicenow.incident-user.field",
		"servicenow.incident-user.lookup",
		"servicenow.incidentdefaults.*",
		"servicenow.incidentstatedefaults.*.*",
		"servicenow.incidentstates.*",
		"servicenow.incidenttable",
		"servicenow.instance",
		"servicenow.password",
		"servicenow.queryresponsefields",
		"servicenow.searchtype",
		"servicenow.username",
	}
	if unread := configkeys.Unread(keys, read); len(unread) != 0 {
		t.Errorf("settings in the example that are not read: %v", unread)
	}
	if unused := configkeys.Unused(keys, read); len(unused) != 0 {
		t.Errorf("settings listed as read that are not in the example: %v", unused)
	}
	missing, err := configkeys.MissingFromSource(read, ".", "../snow")
	if err != nil || len(missing) != 0 {
		t.Errorf("settings that are not in the source: %v (%v)", missing, err)
	}
}
