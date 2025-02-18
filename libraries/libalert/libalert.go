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
	_ "embed"
	"encoding/json"
	htmltemplate "html/template"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"
	"unsafe"

	"github.com/go-mail/mail/v2"

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
const geneosThemecolorRed = "#FF0000"
const geneosThemecolorAmber = "#FFA500"
const geneosThemecolorGreen = "#00FF00"
const DefaultWebhookURLValidationPattern = `^https:\/\/(?:.*\.webhook|outlook)\.office(?:365)?\.com`
const DefaultMsTeamsTimeout = 2000
const DefaultSeverity = ""

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
	conf := parseArgs(n, args)

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

	subject = conf.GetString("_SUBJECT", config.Default(defaultSubject[_SUBJECT]))

	if conf.IsSet("_FORMAT") {
		format = conf.GetString("_FORMAT")
	} else if conf.IsSet("_ALERT") {
		switch conf.GetString("_ALERT_TYPE") {
		case "Alert":
			format = conf.GetString("_ALERT_FORMAT", config.Default(defaultFormat[_ALERT_FORMAT]))
			subject = conf.GetString("_ALERT_SUBJECT", config.Default(defaultSubject[_ALERT_SUBJECT]))
		case "Clear":
			format = conf.GetString("_CLEAR_FORMAT", config.Default(defaultFormat[_CLEAR_FORMAT]))
			subject = conf.GetString("_CLEAR_SUBJECT", config.Default(defaultSubject[_CLEAR_SUBJECT]))
		case "Suspend":
			format = conf.GetString("_SUSPEND_FORMAT", config.Default(defaultFormat[_SUSPEND_FORMAT]))
			subject = conf.GetString("_SUSPEND_SUBJECT", config.Default(defaultSubject[_SUSPEND_SUBJECT]))
		case "Resume":
			format = conf.GetString("_RESUME_FORMAT", config.Default(defaultFormat[_RESUME_FORMAT]))
			subject = conf.GetString("_RESUME_SUBJECT", config.Default(defaultSubject[_RESUME_SUBJECT]))
		case "ThrottleSummary":
			format = conf.GetString("_SUMMARY_FORMAT", config.Default(defaultFormat[_SUMMARY_FORMAT]))
			subject = conf.GetString("_SUMMARY_SUBJECT", config.Default(defaultSubject[_SUMMARY_SUBJECT]))
		default:
			format = defaultFormat[_FORMAT]
		}
	} else {
		format = defaultFormat[_FORMAT]
	}

	body := replArgs(format, conf)
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
// return the result as a new string
func replArgs(format string, conf *config.Config) string {
	result := replArgsRE.ReplaceAllStringFunc(format, func(key string) string {
		// strip containing "%(...)" - as we are here, the regexp must have matched OK
		// so no further check required. No match returns empty string.
		return conf.GetString(key[2 : len(key)-1])
	})

	return result
}

//export GoSendMail
func GoSendMail(n C.int, args **C.char) C.int {
	conf := parseArgs(n, args)

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
	subject := conf.GetString("_SUBJECT", config.Default(defaultSubject[_SUBJECT]))

	// there is a default template that contains embedded tests for which type of
	// alert, if any. This can be overridden with a template file or a template string
	//
	// first grab a suitable Subject if this is an Alert, overridden below if
	// a template file or string is specified
	if conf.IsSet("_ALERT") {
		switch conf.GetString("_ALERT_TYPE") {
		case "Alert":
			subject = conf.GetString("_ALERT_SUBJECT", config.Default(defaultSubject[_ALERT_SUBJECT]))
		case "Clear":
			subject = conf.GetString("_CLEAR_SUBJECT", config.Default(defaultSubject[_CLEAR_SUBJECT]))
		case "Suspend":
			subject = conf.GetString("_SUSPEND_SUBJECT", config.Default(defaultSubject[_SUSPEND_SUBJECT]))
		case "Resume":
			subject = conf.GetString("_RESUME_SUBJECT", config.Default(defaultSubject[_RESUME_SUBJECT]))
		case "ThrottleSummary":
			subject = conf.GetString("_SUMMARY_SUBJECT", config.Default(defaultSubject[_SUMMARY_SUBJECT]))
		}
	}

	// run the subject through text template to allow variable subjects
	subjtmpl := template.New("subject")
	subjtmpl, err = subjtmpl.Parse(subject)
	if err == nil {
		var subjbuf bytes.Buffer
		err = subjtmpl.Execute(&subjbuf, conf)
		if err == nil {
			subject = subjbuf.String()
		}
	}

	m.SetHeader("Subject", subject)

	tmpl := template.New("text")
	if conf.IsSet("_TEMPLATE_TEXT_FILE") {
		tmpl, err = tmpl.ParseFiles(conf.GetString("_TEMPLATE_TEXT_FILE"))
	} else if conf.IsSet("_TEMPLATE_TEXT") {
		tmpl, err = tmpl.Parse(conf.GetString("_TEMPLATE_TEXT"))
	} else {
		tmpl, err = tmpl.Parse(defTextTemplate)
	}

	if err != nil {
		log.Println(err)
		return 1
	}

	var html *htmltemplate.Template

	// conditionally set-up non-text templates
	if !conf.GetBool("_TEMPLATE_TEXT_ONLY") {
		html = htmltemplate.New("base")
		var contents string

		if conf.IsSet("_TEMPLATE_HTML_FILE") {
			contents = config.ExpandString("${file:" + conf.GetString("_TEMPLATE_HTML_FILE") + "}")
			if contents == "" {
				log.Println("error reading", conf.GetString("_TEMPLATE_HTML_FILE"))
				return 1
			}
		} else if conf.IsSet("_TEMPLATE_HTML") {
			contents = conf.GetString("_TEMPLATE_HTML")
		} else {
			contents = defHTMLTemplate
		}
		html, err = html.New("html").Parse(contents)
		if err != nil {
			log.Println(err)
			return 1
		}

		if conf.IsSet("_TEMPLATE_CSS_FILE") {
			contents = config.ExpandString("${file:" + conf.GetString("_TEMPLATE_CSS_FILE") + "}")
			if contents == "" {
				log.Println("error reading", conf.GetString("_TEMPLATE_CSS_FILE"))
				return 1
			}
		} else if conf.IsSet("_TEMPLATE_CSS") {
			contents = conf.GetString("_TEMPLATE_CSS")
		} else {
			contents = defCSSTemplate
		}
		html, err = html.New("css").Parse(contents)
		if err != nil {
			log.Println(err)
			return 1
		}

		// if _TEMPLATE_LOGO_FILE is defined, load that, else use embedded
		if logopath := conf.GetString("_TEMPLATE_LOGO_FILE"); logopath != "" {
			logofile, err := os.Open(logopath)
			if err != nil {
				return 1
			}
			m.EmbedReader("logo.png", logofile)
		} else {
			// use var path and a default of the embedded logo
			mail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(logo)
				return err
			})
			m.EmbedReadSeeker("logo.png", bytes.NewReader(logo))
		}
	}

	// now that we've removed meta params, execute the templates and add the output to
	// the email

	var output bytes.Buffer
	err = tmpl.Execute(&output, conf)
	if err != nil {
		log.Println(err)
		return 1
	}
	m.SetBodyString("text/plain", output.String())

	if !conf.GetBool("_TEMPLATE_TEXT_ONLY") {
		var htmlBody bytes.Buffer
		err = html.ExecuteTemplate(&htmlBody, "html", conf)
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
	var msTeamsWebhooksValidity = make(map[string]bool)
	var clientTimeout time.Duration

	// Parse arguments
	// ---------------
	conf := parseArgs(n, args)

	// Check validity of msTeams incoming webhooks
	// -------------------------------------------
	// Error if no webhooks defined
	if conf.IsSet("_TO") || len(conf.GetString("_TO")) == 0 {
		log.Println("ERR: No MsTeams webhooks defined in _TO. Abort GoSendToMsTeamsChannel().")
		return 1
	}
	// Split webhooks, provided in _TO as a pipe ("|") separate list
	msTeamsWebhooks := strings.Split(conf.GetString("_TO"), "|")
	validityWebhooksCount := 0
	// Browse through webhooks & check validity usng a regex match.
	// Invalid webhooks are ignored.
	regex, err := regexp.Compile(DefaultWebhookURLValidationPattern)
	if err != nil {
		log.Println("ERR: Cannot compile regex to validate msTeams webhooks. Abort GoSendToMsTeamsChannel().")
		return 1
	}
	for i, s := range msTeamsWebhooks {
		if match := regex.MatchString(s); !match {
			log.Printf("WARN: msTeams weekhook #%d (%s) is not valid. Ignoring it.\n", i+1, s)
			msTeamsWebhooksValidity[strings.TrimSpace(s)] = false
		} else {
			msTeamsWebhooksValidity[strings.TrimSpace(s)] = true
			validityWebhooksCount++
		}
	}
	// Error if no valid webhooks defined
	if validityWebhooksCount == 0 {
		log.Println("ERR: No valid msTeams webhooks defined in _TO. Abort GoSendToMsTeamsChannel().")
		return 1
	}

	// Attempt at having a compatibility with the base / default Geneos e-mail formatting
	var subject string
	var header, body string

	// Define the notification subject / title
	// ---------------------------------------
	subject = defaultMsTeamsSubject[_SUBJECT]
	if conf.IsSet("_SUBJECT") && len(conf.GetString("_SUBJECT")) != 0 {
		subject = conf.GetString("_SUBJECT", config.Default(defaultMsTeamsSubject[_SUBJECT]))
	} else if conf.IsSet("_ALERT") {
		switch conf.GetString("_ALERT_TYPE") {
		case "Alert":
			subject = conf.GetString("_ALERT_SUBJECT", config.Default(defaultMsTeamsSubject[_ALERT_SUBJECT]))
		case "Clear":
			subject = conf.GetString("_CLEAR_SUBJECT", config.Default(defaultMsTeamsSubject[_CLEAR_SUBJECT]))
		case "Suspend":
			subject = conf.GetString("_SUSPEND_SUBJECT", config.Default(defaultMsTeamsSubject[_SUSPEND_SUBJECT]))
		case "Resume":
			subject = conf.GetString("_RESUME_SUBJECT", config.Default(defaultMsTeamsSubject[_RESUME_SUBJECT]))
		case "ThrottleSummary":
			subject = conf.GetString("_SUMMARY_SUBJECT", config.Default(defaultMsTeamsSubject[_SUMMARY_SUBJECT]))
		default:
			subject = conf.GetString("_SUBJECT", config.Default(defaultMsTeamsSubject[_SUBJECT]))
		}
	}

	// Run to subject through text template to allow variable subject
	subjtmpl := template.New("subject")
	subjtmpl, err = subjtmpl.Parse(subject)
	if err == nil {
		var subjbuf bytes.Buffer
		err = subjtmpl.Execute(&subjbuf, conf)
		if err == nil {
			subject = subjbuf.String()
		}
	}
	header = replArgs(subject, conf)

	// Define the notification text / body
	// -----------------------------------
	var htmltmpl *htmltemplate.Template
	var texttmpl *template.Template
	var textOnly, useHtmlTmpl bool
	var htmlOutput, textOutput bytes.Buffer
	var contents string
	textOnly = conf.GetBool("_TEMPLATE_TEXT_ONLY")
	useHtmlTmpl = false
	// Identify the template to use and parse it
	if conf.IsSet("_TEMPLATE_HTML_FILE") {
		// Use of HTML template file defined in _TEMPLATE_HTML_FILE
		useHtmlTmpl = true
		contents = config.ExpandString("${file:" + conf.GetString("_TEMPLATE_HTML_FILE") + "}")
		if contents == "" {
			log.Println("ERR: Error reading HTML Template file defined in _TEMPLATE_HTML_FILE. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
		htmltmpl, err = htmltemplate.New("html").Parse(contents)
		if err != nil {
			log.Panicln("ERR: Error parsing template file defined in _TEMPLATE_HTML_FILE. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
	} else if conf.IsSet("_TEMPLATE_HTML") {
		// Use manually defined HTML template found in _TEMPLATE_HTML
		useHtmlTmpl = true
		htmltmpl, err = htmltemplate.New("html").Parse(conf.GetString("_TEMPLATE_HTML"))
		if err != nil {
			log.Println("ERR: Error executing html template in _TO. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
	} else if conf.IsSet("_TEMPLATE_TEXT_FILE") {
		// Use of text template file defined in _TEMPLATE_TEXT_FILE
		useHtmlTmpl = false
		texttmpl, err = template.ParseFiles(conf.GetString("_TEMPLATE_TEXT_FILE"))
		if err != nil {
			log.Println("ERR: Error parsing text template file defined in _TEMPLATE_TEXT_FILE. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
	} else if conf.IsSet("_TEMPLATE_TEXT") {
		// Use manually defined text template found in _TEMPLATE_TEXT
		useHtmlTmpl = false
		texttmpl, err = template.New("text").Parse(conf.GetString("_TEMPLATE_TEXT"))
		if err != nil {
			log.Println("ERR: Error parsing text template defined in _TEMPLATE_TEXT. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
	} else if conf.IsSet("_FORMAT") {
		// _FORMAT defined and interpreted as a html template
		if textOnly {
			useHtmlTmpl = false
			texttmpl, err = template.New("text").Parse(conf.GetString("_FORMAT"))
			if err != nil {
				log.Println("ERR: Error parsing text template in _TO. Abort GoSendToMsTeamsChannel().", err)
				return 1
			}
		} else {
			useHtmlTmpl = true
			htmltmpl, err = htmltemplate.New("html").Parse(conf.GetString("_FORMAT"))
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
		err = htmltmpl.ExecuteTemplate(&htmlOutput, "html", conf)
		if err != nil {
			log.Println("ERR: Error executing HTML template. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
		body = replArgs(htmlOutput.String(), conf)
	} else {
		// Template used is text
		err = texttmpl.Execute(&textOutput, conf)
		if err != nil {
			log.Println("ERR: Error executing text template. Abort GoSendToMsTeamsChannel().", err)
			return 1
		}
		body = replArgs(textOutput.String(), conf)
	}

	// Process MsTeams API call
	// ------------------------
	// Define POST data for REST API call
	var postData msTeamsBasicTextNotifPostData
	postData.Type = msTeamsMessageCard
	postData.Title = header
	postData.Text = body
	switch severity := strings.ToLower(conf.GetString("_SEVERITY", config.Default(DefaultSeverity))); severity {
	case "ok":
		postData.ThemeColor = geneosThemecolorGreen
	case "warning":
		postData.ThemeColor = geneosThemecolorAmber
	case "critical":
		postData.ThemeColor = geneosThemecolorRed
	default:
		postData.ThemeColor = geneosThemecolor
	}

	// Build JSON data
	jsonValue, err := json.Marshal(postData)
	if err != nil {
		log.Println("ERR: Cannot generate JSON data for msTeams API. Abort GoSendToMsTeamsChannel().", err)
		return 1
	}
	jsonBody := bytes.NewReader(jsonValue)

	// Define timeout for RST API call
	clientTimeout = time.Duration(conf.GetInt("_TIMEOUT", config.Default(DefaultMsTeamsTimeout))) * time.Millisecond

	// Call REST API for each target msTeams Webhook
	client := &http.Client{
		Timeout: clientTimeout,
	}
	for k, v := range msTeamsWebhooksValidity {
		if v {
			// Webhook is valid, proceed with REST API call / HTTP POST command
			request, err := http.NewRequest("POST", k, jsonBody)
			if err != nil {
				log.Printf("ERR: Cannot create HTTP POST request to msTeams on URL %s. Continue. %v", k, err)
				continue
			}
			request.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(request)
			if err != nil {
				log.Printf("ERR: Cannot complete HTTP POST request to MsTeams (target %s). Continue. %v", k, err)
			} else {
				log.Printf("INFO: Message sent to %s, return code %d\n", k, resp.StatusCode)
			}
		}
	}
	log.Println("INFO: GoSendToMsTeamsChannel() completed.")
	return 0
} // End of GoSendToMsTeamsChannel()

// parse the C args - "n" of them - and return a config map
// - a value of empty string where there is no "=" or value
func parseArgs(n C.int, args **C.char) (conf *config.Config) {
	// unsafe.Slice() requires Go 1.17+
	for _, s := range unsafe.Slice(args, n) {
		t := strings.SplitN(C.GoString(s), "=", 2)
		if len(t) > 1 {
			conf.Set(t[0], t[1])
		} else {
			conf.Set(t[0], "")
		}
	}
	return conf
}
