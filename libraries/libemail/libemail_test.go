package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	m.Run()
}

// smtpServer is a fake SMTP server that keeps the messages it is sent
type smtpServer struct {
	addr     string
	tlsCert  *tls.Certificate
	mu       sync.Mutex
	messages []string
}

func newSMTPServer(t *testing.T) *smtpServer {
	return startSMTPServer(t, nil)
}

func newTLSSMTPServer(t *testing.T) *smtpServer {
	t.Helper()
	httpServer := httptest.NewTLSServer(http.NotFoundHandler())
	cert := httpServer.TLS.Certificates[0]
	httpServer.Close()
	return startSMTPServer(t, &cert)
}

func startSMTPServer(t *testing.T, tlsCert *tls.Certificate) *smtpServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &smtpServer{addr: ln.Addr().String(), tlsCert: tlsCert}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.serve(conn)
		}
	}()
	return s
}

func (s *smtpServer) serve(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	tlsActive := false
	fmt.Fprint(conn, "220 test\r\n")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		switch cmd := strings.ToUpper(strings.TrimSpace(line)); {
		case strings.HasPrefix(cmd, "EHLO"):
			if s.tlsCert != nil && !tlsActive {
				fmt.Fprint(conn, "250-test\r\n250-STARTTLS\r\n250 ok\r\n")
			} else {
				fmt.Fprint(conn, "250 test\r\n")
			}
		case strings.HasPrefix(cmd, "HELO"):
			fmt.Fprint(conn, "250 test\r\n")
		case strings.HasPrefix(cmd, "STARTTLS"):
			fmt.Fprint(conn, "220 ready\r\n")
			tlsConn := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{*s.tlsCert}})
			if err := tlsConn.Handshake(); err != nil {
				return
			}
			conn = tlsConn
			r = bufio.NewReader(conn)
			tlsActive = true
		case strings.HasPrefix(cmd, "DATA"):
			fmt.Fprint(conn, "354 go\r\n")
			var data strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
				data.WriteString(l)
			}
			s.mu.Lock()
			s.messages = append(s.messages, data.String())
			s.mu.Unlock()
			fmt.Fprint(conn, "250 queued\r\n")
		case strings.HasPrefix(cmd, "QUIT"):
			fmt.Fprint(conn, "221 bye\r\n")
			return
		default:
			fmt.Fprint(conn, "250 ok\r\n")
		}
	}
}

func (s *smtpServer) sent() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.messages...)
}

func (s *smtpServer) args(extra ...string) []string {
	host, port, _ := net.SplitHostPort(s.addr)
	return append([]string{
		"_SMTP_SERVER=" + host, "_SMTP_PORT=" + port, "_SMTP_TLS=none",
		"_TO=ops@example.com", "_FROM=geneos@example.com",
	}, extra...)
}

func TestParseArgs(t *testing.T) {
	got := parseArgs([]string{"_TO=a@example.com", "_FLAG", "_EQUALS=a=b", "_Mixed=Case"})
	want := EMailConfig{"_TO": "a@example.com", "_FLAG": "", "_EQUALS": "a=b", "_Mixed": "Case"}
	if len(got) != len(want) {
		t.Errorf("parseArgs() = %v, want %v", got, want)
	}
	for k, v := range want {
		if g, ok := got[k]; !ok || g != v {
			t.Errorf("parseArgs()[%q] = %q, %t, want %q", k, g, ok, v)
		}
	}
}

func TestProtectReturnsFailureOnPanic(t *testing.T) {
	if got := protect("test", func() int { panic("boom") }); got != 1 {
		t.Errorf("protect() = %d, want 1", got)
	}
	if got := protect("test", func() int { return 0 }); got != 0 {
		t.Errorf("protect() = %d, want 0", got)
	}
}

func TestValidWebhook(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://contoso.webhook.office.com/webhookb2/abc", true},
		{"https://contoso.webhook.office365.com/webhookb2/abc", true},
		{"https://outlook.office.com/webhook/abc", true},
		{"https://prod-01.westeurope.logic.azure.com:443/workflows/abc", true},
		{"https://env.00.environment.api.powerplatform.com/powerautomate/x", true},
		{"https://env.00.environment.api.powerplatform.com:443/x", true},
		{"http://contoso.webhook.office.com/x", false},
		{"https://contoso.webhook.office.com.example.com/x", false},
		{"https://x.azure.com.example.com/x", false},
		{"https://x.environment.api.powerplatform.com.example.com/x", false},
		{"https://x.azure.com@example.com/x", false},
		{"https://example.com/x.azure.com", false},
		{"https://example.com?x=.azure.com", false},
		{"https://x.azure.com:8443/x", false},
		{"https://azure.com/x", false},
		{"", false},
		{"not a url", false},
	}
	for _, test := range tests {
		if got := validWebhook(test.url); got != test.want {
			t.Errorf("validWebhook(%q) = %t, want %t", test.url, got, test.want)
		}
	}
}

// webhooks returns TLS servers, with the status each replies with, and
// makes them valid webhooks that are sent to using a client that trusts
// them
func webhooks(t *testing.T, statuses ...int) (urls []string, bodies func(i int) string) {
	t.Helper()
	var mu sync.Mutex
	got := make([]string, len(statuses))

	for i, status := range statuses {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			mu.Lock()
			got[i] = string(b)
			mu.Unlock()
			w.WriteHeader(status)
		}))
		t.Cleanup(srv.Close)
		urls = append(urls, srv.URL)
	}

	// the servers have different certificates, so trust all of them
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	origValid, origClient := webhookValid, webhookClient
	webhookValid = func(string) bool { return true }
	webhookClient = func(time.Duration) *http.Client { return client }
	t.Cleanup(func() { webhookValid, webhookClient = origValid, origClient })

	return urls, func(i int) string {
		mu.Lock()
		defer mu.Unlock()
		return got[i]
	}
}

func TestSendToMsTeamsWorkflow(t *testing.T) {
	t.Run("no webhooks", func(t *testing.T) {
		for _, args := range [][]string{nil, {"_TO="}, {"_TO=not-a-webhook"}} {
			if got := goSendToMsTeamsWorkflow(parseArgs(args)); got != 1 {
				t.Errorf("goSendToMsTeamsWorkflow(%v) = %d, want 1", args, got)
			}
		}
	})

	t.Run("every webhook gets the whole message", func(t *testing.T) {
		urls, bodies := webhooks(t, 200, 200)
		got := goSendToMsTeamsWorkflow(parseArgs([]string{"_TO=" + strings.Join(urls, " | "), "_SUBJECT=Subject", "_TEMPLATE_TEXT=Value {{._VALUE}}", "_VALUE=42"}))
		if got != 0 {
			t.Fatalf("goSendToMsTeamsWorkflow() = %d, want 0", got)
		}
		if !strings.Contains(bodies(0), "Value 42") || !strings.Contains(bodies(0), "Subject") || bodies(0) != bodies(1) {
			t.Errorf("bodies = %q, %q", bodies(0), bodies(1))
		}
	})

	t.Run("fails if none were sent", func(t *testing.T) {
		urls, _ := webhooks(t, 500, 400)
		if got := goSendToMsTeamsWorkflow(parseArgs([]string{"_TO=" + strings.Join(urls, "|")})); got != 1 {
			t.Errorf("goSendToMsTeamsWorkflow() = %d, want 1", got)
		}
	})

	t.Run("works if any was sent", func(t *testing.T) {
		urls, _ := webhooks(t, 500, 202)
		if got := goSendToMsTeamsWorkflow(parseArgs([]string{"_TO=" + strings.Join(urls, "|")})); got != 0 {
			t.Errorf("goSendToMsTeamsWorkflow() = %d, want 0", got)
		}
	})

	// the setting is _TEMPLATE_TEXT_ONLY, which was not recognised
	t.Run("text only is a text template", func(t *testing.T) {
		text := func(extra ...string) string {
			urls, bodies := webhooks(t, 200)
			args := append([]string{"_TO=" + urls[0], "_FORMAT=Value {{._VALUE}}", "_VALUE=<b>"}, extra...)
			if got := goSendToMsTeamsWorkflow(parseArgs(args)); got != 0 {
				t.Fatalf("goSendToMsTeamsWorkflow() = %d, want 0", got)
			}
			var card struct {
				Attachments []struct {
					Content struct{ Body []struct{ Text string } }
				}
			}
			if err := json.Unmarshal([]byte(bodies(0)), &card); err != nil || len(card.Attachments) != 1 || len(card.Attachments[0].Content.Body) != 2 {
				t.Fatalf("body = %q (%v)", bodies(0), err)
			}
			return card.Attachments[0].Content.Body[1].Text
		}

		if got := text(); got != "Value &lt;b&gt;" {
			t.Errorf("html template text = %q", got)
		}
		if got := text("_TEMPLATE_TEXT_ONLY=true"); got != "Value <b>" {
			t.Errorf("text template text = %q, want the value not escaped as HTML", got)
		}
	})

	t.Run("an invalid timeout is the default", func(t *testing.T) {
		urls, _ := webhooks(t, 200)
		if got := goSendToMsTeamsWorkflow(parseArgs([]string{"_TO=" + urls[0], "_TIMEOUT=abc"})); got != 0 {
			t.Errorf("goSendToMsTeamsWorkflow() = %d, want 0", got)
		}
	})
}

func TestDialServerTLSInsecure(t *testing.T) {
	tests := []struct {
		name     string
		insecure bool
		wantSend bool
	}{
		{name: "insecure enabled", insecure: true, wantSend: true},
		{name: "insecure disabled"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			srv := newTLSSMTPServer(t)
			insecure := strconv.FormatBool(test.insecure)
			client, err := dialServer(parseArgs(srv.args("_SMTP_TLS=force", "_SMTP_TLS_INSECURE="+insecure)))
			if err != nil {
				t.Fatal(err)
			}
			message, err := setupMail(parseArgs(srv.args()))
			if err != nil {
				t.Fatal(err)
			}
			err = client.DialAndSend(message)
			if got := err == nil; got != test.wantSend {
				t.Errorf("DialAndSend success = %t, want %t (error: %v)", got, test.wantSend, err)
			}
		})
	}
}

func TestDialServerInvalidPortUsesDefault(t *testing.T) {
	client, err := dialServer(EMailConfig{"_SMTP_PORT": "xyz"})
	if err != nil {
		t.Fatal(err)
	}
	if client.ServerAddr() != "localhost:25" {
		t.Errorf("server address = %q, want localhost:25", client.ServerAddr())
	}
}

func TestDialServerRejectsNonPositiveTimeout(t *testing.T) {
	for _, timeout := range []string{"0", "-1"} {
		t.Run(timeout, func(t *testing.T) {
			conf := EMailConfig{"_SMTP_TIMEOUT": timeout, "_SMTP_PASSWORD": "secret"}
			if _, err := dialServer(conf); err == nil {
				t.Fatalf("dialServer() accepted _SMTP_TIMEOUT=%s", timeout)
			}
			if _, ok := conf["_SMTP_PASSWORD"]; ok {
				t.Error("dialServer() retained SMTP password after failure")
			}
		})
	}
}

// the settings that are not for the formats and templates must have
// been removed from what they are given
func TestSendMailRemovesSettings(t *testing.T) {
	srv := newSMTPServer(t)
	conf := parseArgs(srv.args(
		"_SMTP_USERNAME=u", "_SMTP_PASSWORD=secret", "_SMTP_TLS_INSECURE=true",
		"_RESUME_SUBJECT=resumed", "_SUMMARY_SUBJECT=summary", "_ALERT_SUBJECT=alert", "_FORMAT=Body",
	))
	// authentication is not offered, so this fails once the settings
	// have been used and removed
	_ = sendMail(conf)
	for _, key := range []string{"_RESUME_SUBJECT", "_SUMMARY_SUBJECT", "_ALERT_SUBJECT", "_FORMAT", "_SMTP_PASSWORD", "_SMTP_TLS_INSECURE", "_SMTP_USERNAME"} {
		if _, ok := conf[key]; ok {
			t.Errorf("%s was not removed", key)
		}
	}
}

// the values from the data that triggered an alert must be used exactly
// as they are, which includes not being expanded
func TestSendMailValues(t *testing.T) {
	secretFile := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secretFile, []byte("file-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("IMS_TEST_SECRET", "env-secret")

	for _, value := range []string{"${env:IMS_TEST_SECRET}", "${file:" + secretFile + "}", "${_TO}"} {
		t.Run("SendMail "+value, func(t *testing.T) {
			srv := newSMTPServer(t)
			conf := parseArgs(srv.args("_FORMAT=Value is %(_VALUE)", "_SUBJECT=Subject", "_VALUE="+value))
			if got := sendMail(conf); got != 0 {
				t.Fatalf("sendMail() = %d, want 0", got)
			}
			if m := srv.sent(); len(m) != 1 || !strings.Contains(m[0], "Value is "+value) {
				t.Errorf("messages = %q", m)
			}
		})

		t.Run("GoSendMail "+value, func(t *testing.T) {
			srv := newSMTPServer(t)
			conf := parseArgs(srv.args("_TEMPLATE_TEXT_ONLY=true", "_TEMPLATE_TEXT=Value {{._VALUE}}", "_SUBJECT=Subject", "_VALUE="+value))
			if got := goSendMail(conf); got != 0 {
				t.Fatalf("goSendMail() = %d, want 0", got)
			}
			if m := srv.sent(); len(m) != 1 || !strings.Contains(m[0], "Value "+value) {
				t.Errorf("messages = %q", m)
			}
		})
	}
}

func TestGoSendMailEmbedsLogo(t *testing.T) {
	srv := newSMTPServer(t)
	conf := parseArgs(srv.args(
		"_TEMPLATE_TEXT=plain body",
		"_TEMPLATE_HTML=<html><img src=\"cid:logo.png\"></html>",
	))

	if got := goSendMail(conf); got != 0 {
		t.Fatalf("goSendMail() = %d, want 0", got)
	}
	messages := srv.sent()
	if len(messages) != 1 {
		t.Fatalf("sent %d messages, want 1", len(messages))
	}
	for _, want := range []string{"Content-Type: multipart/related", "Content-ID: <logo.png>", "cid:logo.png"} {
		if !strings.Contains(strings.ToLower(messages[0]), strings.ToLower(want)) {
			t.Errorf("message does not contain %q", want)
		}
	}
}
