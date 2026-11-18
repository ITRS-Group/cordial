/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import "C"

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unsafe"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/email"
)

//go:embed text.gotmpl
var defTextTemplate string

//go:embed html.gotmpl
var defHTMLTemplate string

//go:embed css.gotmpl
var defCSSTemplate string

//go:embed textmsteams.gotmpl
var defMsTeamsTextTemplate string

//go:embed htmlmsteams.gotmpl
var defMsTeamsHTMLTemplate string

//go:embed css.gotmpl
var defMsTeamsCSSTemplate string

//go:embed logo.png
var logo []byte

const msTeamsMessageCard = "MessageCard"
const geneosThemecolor = "#46e1d7"

// DefaultWebhookHostPattern is the pattern that the whole of the host of
// a webhook URL must match
const DefaultWebhookHostPattern = `^(?:.*\.webhook|outlook)\.office(?:365)?\.com$`

// DefaultMsTeamsTimeout TODO: Describe its purpose and intended use.
const DefaultMsTeamsTimeout = 2000

type msTeamsBasicTextNotifPostData struct {
	Type       string `json:"@type"`
	Title      string `json:"title,omitempty"`
	Text       string `json:"text,omitempty"`
	ThemeColor string `json:"themeColor,omitempty"`
}

// SendMail tries to duplicate the exact behaviour of libemail.so's SendMail function
// but with the addition of more modern SMTP and TLS / authentication
//
// text only, using formats, either defaults or passed in

// for security all "meta" parameters to do with the SMTP configuration and other headers are removed
// unless _DEBUG is set to "true"

//export SendMail
func SendMail(n C.int, args **C.char) C.int {
	return C.int(protect("SendMail", func() int { return sendMail(parseArgs(goArgs(n, args))) }))
}

// protect runs f, which is the body of an exported function, and returns
// its result, or 1 if it panics. Every exported function must use it, as
// a panic that crosses into C ends the program that has loaded the
// library, which is the Geneos Gateway. The panic is logged and the
// failure code returned instead.
func protect(name string, f func() int) (rc int) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("ERR: %s: unexpected failure: %v", name, r)
			rc = 1
		}
	}()
	return f()
}

func sendMail(conf *config.Config, values map[string]string) int {
	d, err := email.Dial(conf)
	if err != nil {
		log.Println(err)
		return 1
	}

	m, err := email.UpdateEnvelope(conf, true)
	if err != nil {
		log.Println(err)
		return 1
	}

	// From doc:
	// "If an _ALERT parameter is present libemail assumes it is being called as part of a gateway alert
	// and will use the appropriate format depending on the value of _ALERT_TYPE (Alert, Clear, Suspend,
	// or Resume). If no _ALERT parameter is specified libemail assumes it is being called as part of an
	// action and uses _FORMAT."
	//
	// A user defined format will always override the default format. If the _FORMAT parameter is
	// specified by the user then this will override any default formats whether or not _ALERT is present.
	//
	// Subjects behave in the same way as formats."
	//
	// Note: "ThrottleSummary" is also mentioned later, but is the same as above
	var format, subject string

	subject = config.Get[string](conf, "_SUBJECT", config.DefaultValue(defaultSubject[_SUBJECT]))

	if conf.IsSet("_FORMAT") {
		format = config.Get[string](conf, "_FORMAT")
	} else if conf.IsSet("_ALERT") {
		switch config.Get[string](conf, "_ALERT_TYPE") {
		case "Alert":
			format = config.Get[string](conf, "_ALERT_FORMAT", config.DefaultValue(defaultFormat[_ALERT_FORMAT]))
			subject = config.Get[string](conf, "_ALERT_SUBJECT", config.DefaultValue(defaultSubject[_ALERT_SUBJECT]))
		case "Clear":
			format = config.Get[string](conf, "_CLEAR_FORMAT", config.DefaultValue(defaultFormat[_CLEAR_FORMAT]))
			subject = config.Get[string](conf, "_CLEAR_SUBJECT", config.DefaultValue(defaultSubject[_CLEAR_SUBJECT]))
		case "Suspend":
			format = config.Get[string](conf, "_SUSPEND_FORMAT", config.DefaultValue(defaultFormat[_SUSPEND_FORMAT]))
			subject = config.Get[string](conf, "_SUSPEND_SUBJECT", config.DefaultValue(defaultSubject[_SUSPEND_SUBJECT]))
		case "Resume":
			format = config.Get[string](conf, "_RESUME_FORMAT", config.DefaultValue(defaultFormat[_RESUME_FORMAT]))
			subject = config.Get[string](conf, "_RESUME_SUBJECT", config.DefaultValue(defaultSubject[_RESUME_SUBJECT]))
		case "ThrottleSummary":
			format = config.Get[string](conf, "_SUMMARY_FORMAT", config.DefaultValue(defaultFormat[_SUMMARY_FORMAT]))
			subject = config.Get[string](conf, "_SUMMARY_SUBJECT", config.DefaultValue(defaultSubject[_SUMMARY_SUBJECT]))
		default:
			format = defaultFormat[_FORMAT]
		}
	} else {
		format = defaultFormat[_FORMAT]
	}

	body := replArgs(format, values)
	m.SetGenHeader("Subject", subject)
	m.SetBodyString("text/plain", body)

	if err = d.DialAndSend(m); err != nil {
		log.Println(err)
		return 1
	}
	return 0
}

var replArgsRE = regexp.MustCompile(`%\([^\)]*\)`)

// substitute placeholder of the form %(XXX) for the value of XXX or empty and
// return the result as a new string. The values are used exactly as they
// were passed, as they can come from the data that triggered the alert
// and must not be expanded.
func replArgs(format string, values map[string]string) string {
	result := replArgsRE.ReplaceAllStringFunc(format, func(key string) string {
		// strip containing "%(...)" - as we are here, the regexp must have matched OK
		// so no further check required. No match returns empty string.
		return values[key[2:len(key)-1]]
	})

	return result
}

//export GoSendMail
func GoSendMail(n C.int, args **C.char) C.int {
	return C.int(protect("GoSendMail", func() int { return goSendMail(parseArgs(goArgs(n, args))) }))
}

func goSendMail(conf *config.Config, values map[string]string) int {
	d, err := email.Dial(conf)
	if err != nil {
		log.Println(err)
		return 1
	}

	m, err := email.UpdateEnvelope(conf, true)
	if err != nil {
		log.Println(err)
		return 1
	}

	// The subject follows the same rules as the original SendMail function
	subject := config.Get[string](conf, "_SUBJECT", config.DefaultValue(defaultSubject[_SUBJECT]))

	// there is a default template that contains embedded tests for which type of
	// alert, if any. This can be overridden with a template file or a template string
	//
	// first grab a suitable Subject if this is an Alert, overridden below if
	// a template file or string is specified
	if conf.IsSet("_ALERT") {
		switch config.Get[string](conf, "_ALERT_TYPE") {
		case "Alert":
			subject = config.Get[string](conf, "_ALERT_SUBJECT", config.DefaultValue(defaultSubject[_ALERT_SUBJECT]))
		case "Clear":
			subject = config.Get[string](conf, "_CLEAR_SUBJECT", config.DefaultValue(defaultSubject[_CLEAR_SUBJECT]))
		case "Suspend":
			subject = config.Get[string](conf, "_SUSPEND_SUBJECT", config.DefaultValue(defaultSubject[_SUSPEND_SUBJECT]))
		case "Resume":
			subject = config.Get[string](conf, "_RESUME_SUBJECT", config.DefaultValue(defaultSubject[_RESUME_SUBJECT]))
		case "ThrottleSummary":
			subject = config.Get[string](conf, "_SUMMARY_SUBJECT", config.DefaultValue(defaultSubject[_SUMMARY_SUBJECT]))
		}
	}

	// run the subject through text template to allow variable subjects
	subjtmpl := template.New("subject")
	subjtmpl, err = subjtmpl.Parse(subject)
	if err == nil {
		var subjbuf bytes.Buffer
		err = subjtmpl.Execute(&subjbuf, values)
		if err == nil {
			subject = subjbuf.String()
		}
	}

	m.SetHeader("Subject", subject)

	tmpl := template.New("text")
	if conf.IsSet("_TEMPLATE_TEXT_FILE") {
		tmpl, err = tmpl.ParseFiles(config.Get[string](conf, "_TEMPLATE_TEXT_FILE"))
	} else if conf.IsSet("_TEMPLATE_TEXT") {
		tmpl, err = tmpl.Parse(config.Get[string](conf, "_TEMPLATE_TEXT"))
	} else {
		tmpl, err = tmpl.Parse(defTextTemplate)
	}

	if err != nil {
		log.Println(err)
		return 1
	}

	var html *htmltemplate.Template

	// conditionally set-up non-text templates
	if !config.Get[bool](conf, "_TEMPLATE_TEXT_ONLY") {
		html = htmltemplate.New("base")
		var contents string

		if conf.IsSet("_TEMPLATE_HTML_FILE") {
			if contents, err = readTemplateFile(config.Get[string](conf, "_TEMPLATE_HTML_FILE")); err != nil {
				log.Println(err)
				return 1
			}
		} else if conf.IsSet("_TEMPLATE_HTML") {
			contents = config.Get[string](conf, "_TEMPLATE_HTML")
		} else {
			contents = defHTMLTemplate
		}
		html, err = html.New("html").Parse(contents)
		if err != nil {
			log.Println(err)
			return 1
		}

		if conf.IsSet("_TEMPLATE_CSS_FILE") {
			if contents, err = readTemplateFile(config.Get[string](conf, "_TEMPLATE_CSS_FILE")); err != nil {
				log.Println(err)
				return 1
			}
		} else if conf.IsSet("_TEMPLATE_CSS") {
			contents = config.Get[string](conf, "_TEMPLATE_CSS")
		} else {
			contents = defCSSTemplate
		}
		html, err = html.New("css").Parse(contents)
		if err != nil {
			log.Println(err)
			return 1
		}

		// if _TEMPLATE_LOGO_FILE is defined, load that, else use embedded
		if logopath := config.Get[string](conf, "_TEMPLATE_LOGO_FILE"); logopath != "" {
			logofile, err := os.Open(logopath)
			if err != nil {
				log.Println(err)
				return 1
			}
			defer logofile.Close()
			m.EmbedReader("logo.png", logofile)
		} else {
			m.EmbedReadSeeker("logo.png", bytes.NewReader(logo))
		}
	}

	// now that we've removed meta params, execute the templates and add the output to
	// the email

	var output bytes.Buffer
	err = tmpl.Execute(&output, values)
	if err != nil {
		log.Println(err)
		return 1
	}
	m.SetBodyString("text/plain", output.String())

	if !config.Get[bool](conf, "_TEMPLATE_TEXT_ONLY") {
		var htmlBody bytes.Buffer
		err = html.ExecuteTemplate(&htmlBody, "html", values)
		if err != nil {
			log.Println(err)
			return 1
		}
		m.AddAlternativeString("text/html", htmlBody.String())
	}

	err = d.DialAndSend(m)
	if err != nil {
		log.Println(err)
		return 1
	}
	return 0
}

//export GoSendToMsTeamsChannel
func GoSendToMsTeamsChannel(n C.int, args **C.char) C.int {
	return C.int(protect("GoSendToMsTeamsChannel", func() int { return goSendToMsTeamsChannel(parseArgs(goArgs(n, args))) }))
}

var webhookHostRE = regexp.MustCompile(DefaultWebhookHostPattern)

// validWebhook returns true if s is the URL of a Microsoft Teams webhook,
// which must be HTTPS, with a host that is entirely one of those that
// Microsoft use, and without credentials in the URL that can be used to
// make the host look like something else
func validWebhook(s string) bool {
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "https" || u.User != nil {
		return false
	}
	if p := u.Port(); p != "" && p != "443" {
		return false
	}
	return webhookHostRE.MatchString(strings.ToLower(u.Hostname()))
}

// webhookValid is the check of the URL of each webhook, which tests can
// replace to send to a server of their own
var webhookValid = validWebhook

// webhookClient returns the client to send to webhooks, which tests can
// replace
var webhookClient = func(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

func goSendToMsTeamsChannel(conf *config.Config, values map[string]string) int {
	// Parse arguments
	// ---------------

	// Check validity of msTeams incoming webhooks
	// -------------------------------------------
	// Error if no webhooks defined
	if !conf.IsSet("_TO") || len(config.Get[string](conf, "_TO")) == 0 {
		log.Println("ERR: No MsTeams webhooks defined in _TO. Abort GoSendToMsTeamsChannel().")
		return 1
	}
	// Split webhooks, provided in _TO as a pipe ("|") separate list
	// and browse through them to check validity. Invalid webhooks are
	// ignored.
	var webhooks []string
	for i, s := range strings.Split(config.Get[string](conf, "_TO"), "|") {
		s = strings.TrimSpace(s)
		if !webhookValid(s) {
			log.Printf("WARN: msTeams weekhook #%d (%s) is not valid. Ignoring it.\n", i+1, s)
			continue
		}
		webhooks = append(webhooks, s)
	}
	// Error if no valid webhooks defined
	if len(webhooks) == 0 {
		log.Println("ERR: No valid msTeams webhooks defined in _TO. Abort GoSendToMsTeamsChannel().")
		return 1
	}

	// Attempt at having a compatibility with the base / default Geneos e-mail formatting
	var subject string
	var header, body string

	// Define the notification subject / title
	// ---------------------------------------
	subject = defaultMsTeamsSubject[_SUBJECT]
	if conf.IsSet("_SUBJECT") && len(config.Get[string](conf, "_SUBJECT")) != 0 {
		subject = config.Get[string](conf, "_SUBJECT", config.DefaultValue(defaultMsTeamsSubject[_SUBJECT]))
	} else if conf.IsSet("_ALERT") {
		switch config.Get[string](conf, "_ALERT_TYPE") {
		case "Alert":
			subject = config.Get[string](conf, "_ALERT_SUBJECT", config.DefaultValue(defaultMsTeamsSubject[_ALERT_SUBJECT]))
		case "Clear":
			subject = config.Get[string](conf, "_CLEAR_SUBJECT", config.DefaultValue(defaultMsTeamsSubject[_CLEAR_SUBJECT]))
		case "Suspend":
			subject = config.Get[string](conf, "_SUSPEND_SUBJECT", config.DefaultValue(defaultMsTeamsSubject[_SUSPEND_SUBJECT]))
		case "Resume":
			subject = config.Get[string](conf, "_RESUME_SUBJECT", config.DefaultValue(defaultMsTeamsSubject[_RESUME_SUBJECT]))
		case "ThrottleSummary":
			subject = config.Get[string](conf, "_SUMMARY_SUBJECT", config.DefaultValue(defaultMsTeamsSubject[_SUMMARY_SUBJECT]))
		default:
			subject = config.Get[string](conf, "_SUBJECT", config.DefaultValue(defaultMsTeamsSubject[_SUBJECT]))
		}
	}

	// Run to subject through text template to allow variable subject
	if subjtmpl, err := template.New("subject").Parse(subject); err == nil {
		var subjbuf bytes.Buffer
		if err = subjtmpl.Execute(&subjbuf, values); err == nil {
			subject = subjbuf.String()
		}
	}
	header = replArgs(subject, values)

	// Define the notification text / body
	// -----------------------------------
	var htmltmpl *htmltemplate.Template
	var texttmpl *template.Template
	var textOnly, useHtmlTmpl bool
	var htmlOutput, textOutput bytes.Buffer
	var contents string
	var err error
	textOnly = config.Get[bool](conf, "_TEMPLATE_TEXT_ONLY")
	useHtmlTmpl = false
	// Identify the template to use and parse it
	if conf.IsSet("_TEMPLATE_HTML_FILE") {
		// Use of HTML template file defined in _TEMPLATE_HTML_FILE
		useHtmlTmpl = true
		contents, err = readTemplateFile(config.Get[string](conf, "_TEMPLATE_HTML_FILE"))
		if err != nil {
			log.Println("ERR: Error reading HTML Template file defined in _TEMPLATE_HTML_FILE. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
		htmltmpl, err = htmltemplate.New("html").Parse(contents)
		if err != nil {
			log.Println("ERR: Error parsing template file defined in _TEMPLATE_HTML_FILE. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
	} else if conf.IsSet("_TEMPLATE_HTML") {
		// Use manually defined HTML template found in _TEMPLATE_HTML
		useHtmlTmpl = true
		htmltmpl, err = htmltemplate.New("html").Parse(config.Get[string](conf, "_TEMPLATE_HTML"))
		if err != nil {
			log.Println("ERR: Error executing html template in _TO. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
	} else if conf.IsSet("_TEMPLATE_TEXT_FILE") {
		// Use of text template file defined in _TEMPLATE_TEXT_FILE
		useHtmlTmpl = false
		texttmpl, err = template.ParseFiles(config.Get[string](conf, "_TEMPLATE_TEXT_FILE"))
		if err != nil {
			log.Println("ERR: Error parsing text template file defined in _TEMPLATE_TEXT_FILE. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
	} else if conf.IsSet("_TEMPLATE_TEXT") {
		// Use manually defined text template found in _TEMPLATE_TEXT
		useHtmlTmpl = false
		texttmpl, err = template.New("text").Parse(config.Get[string](conf, "_TEMPLATE_TEXT"))
		if err != nil {
			log.Println("ERR: Error parsing text template defined in _TEMPLATE_TEXT. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
	} else if conf.IsSet("_FORMAT") {
		// _FORMAT defined and interpreted as a html template
		if textOnly {
			useHtmlTmpl = false
			texttmpl, err = template.New("text").Parse(config.Get[string](conf, "_FORMAT"))
			if err != nil {
				log.Println("ERR: Error parsing text template in _TO. Abort GoSendToMsTeamsChannel().", err)
				return 1
			}
		} else {
			useHtmlTmpl = true
			htmltmpl, err = htmltemplate.New("html").Parse(config.Get[string](conf, "_FORMAT"))
			if err != nil {
				log.Println("ERR: Error parsing html template in _TO. Abort GoSendToMsTeamsChannel().", err)
				return 1
			}
		}
	} else if textOnly {
		// Use default text template file
		useHtmlTmpl = false
		texttmpl, err = template.New("text").Parse(defMsTeamsTextTemplate)
		if err != nil {
			log.Println("ERR: Error parsing default text template. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
	} else {
		// Use default HTML template file
		useHtmlTmpl = true
		contents = defMsTeamsHTMLTemplate
		htmltmpl, err = htmltemplate.New("html").Parse(contents)
		if err != nil {
			log.Println("ERR: Error persing default HTML template. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
	}
	// Execute the template & account for older/legacy inputs formats from Geneos
	if useHtmlTmpl {
		// Template used is HTML
		err = htmltmpl.ExecuteTemplate(&htmlOutput, "html", values)
		if err != nil {
			log.Println("ERR: Error executing HTML template. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
		body = replArgs(htmlOutput.String(), values)
	} else {
		// Template used is text
		err = texttmpl.Execute(&textOutput, values)
		if err != nil {
			log.Println("ERR: Error executing text template. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
		body = replArgs(textOutput.String(), values)
	}

	// Process MsTeams API call
	// ------------------------
	// Define POST data for REST API call
	var postData msTeamsBasicTextNotifPostData
	postData.Type = msTeamsMessageCard
	postData.Title = header
	postData.Text = body
	postData.ThemeColor = geneosThemecolor

	// Build JSON data
	jsonValue, err := json.Marshal(postData)
	if err != nil {
		log.Println("ERR: Cannot generate JSON data for msTeams API. Abort GoSendToMsTeamsChannel().", err)
		return 1
	}

	// Define timeout for RST API call
	clientTimeout := webhookTimeout(config.Get[string](conf, "_TIMEOUT"))

	// Call REST API for each target msTeams Webhook
	if sent := postWebhooks(webhookClient(clientTimeout), webhooks, jsonValue); sent == 0 {
		log.Println("ERR: GoSendToMsTeamsChannel() failed to send to any webhook.")
		return 1
	}
	log.Println("INFO: GoSendToMsTeamsChannel() completed.")
	return 0
} // End of GoSendToMsTeamsChannel()

// webhookTimeout returns the timeout for requests to webhooks from the
// value of `_TIMEOUT`, which is a number of milliseconds or a duration
// such as `5s`. It is the default if there is no value, or it is not
// valid, as a zero timeout is no timeout.
func webhookTimeout(value string) time.Duration {
	value = strings.TrimSpace(value)
	if ms, err := strconv.Atoi(value); err == nil && ms > 0 {
		return time.Duration(ms) * time.Millisecond
	}
	if d, err := time.ParseDuration(value); err == nil && d > 0 {
		return d
	}
	return DefaultMsTeamsTimeout * time.Millisecond
}

// postWebhooks posts the JSON to each of the webhooks and returns how
// many accepted it, which is the number that returned a successful
// status
func postWebhooks(client *http.Client, webhooks []string, jsonValue []byte) (sent int) {
	for _, webhook := range webhooks {
		// each request has its own body, as it is read to the end by the
		// first one that is sent
		request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, webhook, bytes.NewReader(jsonValue))
		if err != nil {
			log.Printf("ERR: Cannot create HTTP POST request to msTeams on URL %s. Continue. %v", webhook, err)
			continue
		}
		request.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(request)
		if err != nil {
			log.Printf("ERR: Cannot complete HTTP POST request to MsTeams (target %s). Continue. %v", webhook, err)
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			log.Printf("ERR: MsTeams (target %s) did not accept the message, return code %d\n", webhook, resp.StatusCode)
			continue
		}
		log.Printf("INFO: Message sent to %s, return code %d\n", webhook, resp.StatusCode)
		sent++
	}
	return
}

// readTemplateFile returns the contents of a template file, which is
// an error if it cannot be read or is empty
func readTemplateFile(path string) (contents string, err error) {
	b, err := os.ReadFile(config.ResolveHome(path))
	if err != nil {
		return "", fmt.Errorf("error reading %s: %w", path, err)
	}
	if len(b) == 0 {
		return "", fmt.Errorf("error reading %s: file is empty", path)
	}
	return string(b), nil
}

// goArgs returns the arguments passed by C as Go strings
func goArgs(n C.int, args **C.char) (goargs []string) {
	for _, s := range unsafe.Slice(args, n) {
		goargs = append(goargs, C.GoString(s))
	}
	return
}

// parseArgs parses the arguments, which are of the form `NAME=VALUE`,
// and returns the configuration for the arguments, with a value of empty
// string where there is no "=" or value, and the arguments as they are
// for the templates, as the configuration does not keep the case of the
// names and expands values, and the values from the alert must be used
// as they are.
func parseArgs(args []string) (conf *config.Config, values map[string]string) {
	conf = config.New()
	values = make(map[string]string, len(args))
	for _, s := range args {
		k, v, _ := strings.Cut(s, "=")
		config.Set(conf, k, v)
		values[k] = v
	}
	return
}
