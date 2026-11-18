package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PagerDuty/go-pagerduty"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/internal/configkeys"
	"github.com/itrs-group/cordial/pkg/config"
)

func TestMain(m *testing.M) {
	*log = *slog.New(slog.DiscardHandler)
	m.Run()
}

// defaultsConfig returns the configuration with the shipped defaults
func defaultsConfig() *config.Config {
	return config.New(config.WithDefaults(defaults, "yaml"))
}

func TestGeneosTimestamp(t *testing.T) {
	orig := time.Local
	defer func() { time.Local = orig }()
	time.Local = time.FixedZone("TEST", 10*3600)

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		name, in, want string
	}{
		{"local time", "Wed Nov 18 09:30:00 2026", "2026-11-18T09:30:00+10:00"},
		{"single digit day", "Sun Nov  1 00:15:30 2026", "2026-11-01T00:15:30+10:00"},
		{"empty is now", "", "2026-01-02T03:04:05Z"},
		{"invalid is now", "yesterday", "2026-01-02T03:04:05Z"},
	}
	for _, test := range tests {
		if got := geneosTimestamp(test.in, now); got != test.want {
			t.Errorf("%s: geneosTimestamp(%q) = %q, want %q", test.name, test.in, got, test.want)
		}
	}
}

func TestBuildEventFromDefaults(t *testing.T) {
	t.Setenv("PAGERDUTY_INTEGRATION_KEY", "routing-key")
	t.Setenv("_ALERT_TYPE", "")
	t.Setenv("_VARIABLEPATH", "/path")
	t.Setenv("_RULE", "rule")
	t.Setenv("_MANAGED_ENTITY", "entity")
	t.Setenv("_SEVERITY", "critical")
	t.Setenv("_ALERT_TIME", "Wed Nov 18 09:30:00 2026")
	t.Setenv("_SAMPLER", "sampler")
	t.Setenv("_NETPROBE_HOST", "netprobe-host")
	t.Setenv("_SAMPLER_GROUP", "group")
	t.Setenv("_KBA_URLS", "https://example.com/1\nhttps://example.com/2")

	cf := defaultsConfig()
	config.Set(cf, "pagerduty.event.images", []any{
		map[string]any{"src": "https://example.com/i.png", "alt": "graph"},
		map[string]any{"href": "no source"},
	})

	event := buildEvent(cf, Trigger)
	p := event.Payload
	if event.RoutingKey != "routing-key" || event.Action != "trigger" || event.DedupKey != "/path" || event.Client != "ITRS Geneos" || event.ClientURL != "https://www.itrsgroup.com" {
		t.Errorf("event = %+v", event)
	}
	if p.Summary != "rule Triggered" || p.Source != "entity" || p.Severity != "critical" || p.Class != "sampler" || p.Group != "group" {
		t.Errorf("payload = %+v", p)
	}
	if p.Component != "netprobe-host" {
		t.Errorf("component = %q, want the default from the configuration", p.Component)
	}
	if len(event.Links) != 2 || event.Links[1].(Link).Href != "https://example.com/2" {
		t.Errorf("links = %v", event.Links)
	}
	if len(event.Images) != 1 || event.Images[0].(Image) != (Image{Src: "https://example.com/i.png", Alt: "graph"}) {
		t.Errorf("images = %v", event.Images)
	}
}

// the environment can be sent when there are no details configured
func TestBuildEventSendEnv(t *testing.T) {
	t.Setenv("IMS_TEST_VALUE", "value")
	cf := defaultsConfig()
	config.Set(cf, "pagerduty.send-env", true)
	event := buildEvent(cf, Trigger)
	if got := event.Payload.Details.(map[string]string)["IMS_TEST_VALUE"]; got != "value" {
		t.Errorf("details = %v", event.Payload.Details)
	}
}

func TestBuildEventActions(t *testing.T) {
	tests := []struct {
		name      string
		eventType eventType
		severity  string
		alertType string
		want      string
		wantSev   string
	}{
		{"trigger", Trigger, "critical", "", "trigger", "critical"},
		{"resolve command", Resolve, "critical", "", "resolve", "info"},
		{"ok severity", Trigger, "ok", "", "resolve", "info"},
		{"clear", Trigger, "critical", "CLEAR", "resolve", "info"},
		{"assign command", Assign, "critical", "", "acknowledge", "info"},
		{"suspend", Trigger, "critical", "suspend", "acknowledge", "info"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("_SEVERITY", test.severity)
			t.Setenv("_ALERT_TYPE", test.alertType)
			event := buildEvent(defaultsConfig(), test.eventType)
			if event.Action != test.want || event.Payload.Severity != test.wantSev {
				t.Errorf("action = %q, severity = %q, want %q, %q", event.Action, event.Payload.Severity, test.want, test.wantSev)
			}
		})
	}
}

func TestManageEvent(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, `{"status":"success","message":"Event processed","dedup_key":"d"}`)
	}))
	defer srv.Close()
	origOptions := clientOptions
	defer func() { clientOptions = origOptions }()
	clientOptions = []pagerduty.ClientOptions{pagerduty.WithV2EventsAPIEndpoint(srv.URL)}

	t.Setenv("_NETPROBE_HOST", "netprobe-host")
	cf := defaultsConfig()
	config.Set(cf, "pagerduty.event.images", []any{map[string]any{"src": "https://example.com/i.png"}})
	event := buildEvent(cf, Trigger)
	if err := manageEvent(cf, &event); err != nil {
		t.Fatal(err)
	}

	var sent struct {
		Payload struct{ Component string }
		Images  []struct{ Src string }
	}
	if err := json.Unmarshal(body, &sent); err != nil || sent.Payload.Component != "netprobe-host" || len(sent.Images) != 1 || sent.Images[0].Src != "https://example.com/i.png" {
		t.Errorf("sent %s (%v)", body, err)
	}
}

// the routing key is in the body and must not be sent on by a redirect
func TestManageEventDoesNotFollowRedirects(t *testing.T) {
	var other atomic.Int32
	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		other.Add(1)
	}))
	defer b.Close()
	a := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, b.URL, http.StatusPermanentRedirect)
	}))
	defer a.Close()
	origOptions := clientOptions
	defer func() { clientOptions = origOptions }()
	clientOptions = []pagerduty.ClientOptions{pagerduty.WithV2EventsAPIEndpoint(a.URL)}

	cf := defaultsConfig()
	event := buildEvent(cf, Trigger)
	if err := manageEvent(cf, &event); err == nil {
		t.Error("manageEvent succeeded, want an error for the redirect")
	}
	if n := other.Load(); n != 0 {
		t.Errorf("redirect target received %d requests, want none", n)
	}
}

// Pagerduty not answering must not leave the command waiting
func TestManageEventTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)
	origOptions := clientOptions
	defer func() { clientOptions = origOptions }()
	clientOptions = []pagerduty.ClientOptions{pagerduty.WithV2EventsAPIEndpoint(srv.URL)}

	cf := defaultsConfig()
	config.Set(cf, "pagerduty.timeout", "100ms")
	event := buildEvent(cf, Trigger)

	start := time.Now()
	if err := manageEvent(cf, &event); err == nil || time.Since(start) > 5*time.Second {
		t.Errorf("err = %v after %s, want a timeout", err, time.Since(start))
	}
}

// the routing key, and the environment that can be sent in the details,
// are secrets and must not be in the log
func TestFailureLogHasNoSecrets(t *testing.T) {
	t.Setenv("IMS_TEST_SECRET", "env-secret")
	cf := defaultsConfig()
	config.Set(cf, "pagerduty.routingkey", "the-routing-key")
	config.Set(cf, "pagerduty.send-env", true)
	event := buildEvent(cf, Trigger)

	var out strings.Builder
	logger := slog.New(slog.NewJSONHandler(&out, nil))
	logger.Error("error managing event", failureAttrs(&event)...)
	if strings.Contains(out.String(), "the-routing-key") || strings.Contains(out.String(), "env-secret") {
		t.Errorf("log = %s", out.String())
	}
}

func TestExecnameIsTheExecutableName(t *testing.T) {
	if execname != cordial.ExecutableName() || strings.ContainsAny(execname, `/\`) || strings.HasSuffix(strings.ToLower(execname), ".exe") {
		t.Errorf("execname = %q, want %q", execname, cordial.ExecutableName())
	}
}

// every setting in the defaults must be one that is read
func TestDefaultsAreAllRead(t *testing.T) {
	keys, err := configkeys.Leaves(defaults)
	if err != nil {
		t.Fatal(err)
	}
	read := []string{
		"pagerduty.authtoken",
		"pagerduty.routingkey",
		"pagerduty.timeout",
		"pagerduty.alert-type",
		"pagerduty.severity-map.*",
		"pagerduty.event.dedup-key",
		"pagerduty.event.client",
		"pagerduty.event.client_url",
		"pagerduty.event.links",
		"pagerduty.event.images",
		"pagerduty.event.payload.summary",
		"pagerduty.event.payload.source",
		"pagerduty.event.payload.severity",
		"pagerduty.event.payload.timestamp",
		"pagerduty.event.payload.class",
		"pagerduty.event.payload.component",
		"pagerduty.event.payload.group",
		"pagerduty.send-env",
	}
	if unread := configkeys.Unread(keys, read); len(unread) != 0 {
		t.Errorf("settings in the defaults that are not read: %v", unread)
	}
	if unused := configkeys.Unused(keys, read); len(unused) != 0 {
		t.Errorf("settings listed as read that are not in the defaults: %v", unused)
	}
	if missing, err := configkeys.MissingFromSource(read, "."); err != nil || len(missing) != 0 {
		t.Errorf("settings that are not in the source: %v (%v)", missing, err)
	}
}
