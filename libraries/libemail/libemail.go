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
	"fmt"
	htmltemplate "html/template"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/go-mail/mail/v2"
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
const DefaultMsTeamsTimeout = 2000

var webhookValidators []string = []string{
	`^https:\/\/(.+\.webhook|outlook)\.office(365)?\.com`,
	`^https:\/\/.+\.azure\.com`,
	// New Power Platform / Power Automate Workflow
	`^https:\/\/.+\.environment\.api\.powerplatform\.com(:443)?`,
}

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

	d, err := dialServer(conf)
	if err != nil {
		log.Println(err)
		return 1
	}

	m, err := setupMail(conf)
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

	subject = getWithDefault("_SUBJECT", conf, defaultSubject[_SUBJECT])

	if _, ok := conf["_FORMAT"]; ok {
		format = conf["_FORMAT"]
	} else if _, ok = conf["_ALERT"]; ok {
		switch conf["_ALERT_TYPE"] {
		case "Alert":
			format = getWithDefault("_ALERT_FORMAT", conf, defaultFormat[_ALERT_FORMAT])
			subject = getWithDefault("_ALERT_SUBJECT", conf, defaultSubject[_ALERT_SUBJECT])
		case "Clear":
			format = getWithDefault("_CLEAR_FORMAT", conf, defaultFormat[_CLEAR_FORMAT])
			subject = getWithDefault("_CLEAR_SUBJECT", conf, defaultSubject[_CLEAR_SUBJECT])
		case "Suspend":
			format = getWithDefault("_SUSPEND_FORMAT", conf, defaultFormat[_SUSPEND_FORMAT])
			subject = getWithDefault("_SUSPEND_SUBJECT", conf, defaultSubject[_SUSPEND_SUBJECT])
		case "Resume":
			format = getWithDefault("_RESUME_FORMAT", conf, defaultFormat[_RESUME_FORMAT])
			subject = getWithDefault("_RESUME_SUBJECT", conf, defaultSubject[_RESUME_SUBJECT])
		case "ThrottleSummary":
			format = getWithDefault("_SUMMARY_FORMAT", conf, defaultFormat[_SUMMARY_FORMAT])
			subject = getWithDefault("_SUMMARY_SUBJECT", conf, defaultSubject[_SUMMARY_SUBJECT])
		default:
			format = defaultFormat[_FORMAT]
		}
	} else {
		format = defaultFormat[_FORMAT]
	}

	if !debug(conf) {
		keys := []string{"_FORMAT", "_ALERT_FORMAT", "_CLEAR_FORMAT", "_SUSPEND_FORMAT", "_RESUME_FORMAT", "_SUMMARY_FORMAT",
			"_SUBJECT", "_ALERT_SUBJECT", "_CLEAR_SUBJECT", "_SUSPEND_SUBJECT", "_RESUME_TEMPLATE", "_SUMMARY_TEMPLATE",
		}
		for _, key := range keys {
			delete(conf, key)
		}
	}

	body := replArgs(format, conf)
	m.SetHeader("Subject", replArgs(subject, conf))
	m.SetBody("text/plain", body)

	if err = d.DialAndSend(m); err != nil {
		log.Println(err)
		return 1
	}
	return 0
}

var replArgsRE = regexp.MustCompile(`%\([^\)]*\)`)

// substitute placeholder of the form %(XXX) for the value of XXX or empty and
// return the result as a new string
func replArgs(format string, conf EMailConfig) string {
	result := replArgsRE.ReplaceAllStringFunc(format, func(key string) string {
		// strip containing "%(...)" - as we are here, the regexp must have matched OK
		// so no further check required. No match returns empty string.
		return conf[key[2:len(key)-1]]
	})

	return result
}

//export GoSendMail
func GoSendMail(n C.int, args **C.char) C.int {
	conf := parseArgs(n, args)

	d, err := dialServer(conf)
	if err != nil {
		log.Println(err)
		return 1
	}

	m, err := setupMail(conf)
	if err != nil {
		log.Println(err)
		return 1
	}

	// The subject follows the same rules as the original SendMail function
	subject := getWithDefault("_SUBJECT", conf, defaultSubject[_SUBJECT])

	// there is a default template that contains embedded tests for which type of
	// alert, if any. This can be overridden with a template file or a template string
	//
	// first grab a suitable Subject if this is an Alert, overridden below if
	// a template file or string is specified
	if _, ok := conf["_ALERT"]; ok {
		switch conf["_ALERT_TYPE"] {
		case "Alert":
			subject = getWithDefault("_ALERT_SUBJECT", conf, defaultSubject[_ALERT_SUBJECT])
		case "Clear":
			subject = getWithDefault("_CLEAR_SUBJECT", conf, defaultSubject[_CLEAR_SUBJECT])
		case "Suspend":
			subject = getWithDefault("_SUSPEND_SUBJECT", conf, defaultSubject[_SUSPEND_SUBJECT])
		case "Resume":
			subject = getWithDefault("_RESUME_SUBJECT", conf, defaultSubject[_RESUME_SUBJECT])
		case "ThrottleSummary":
			subject = getWithDefault("_SUMMARY_SUBJECT", conf, defaultSubject[_SUMMARY_SUBJECT])
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

	m.SetHeader("Subject", replArgs(subject, conf))

	tmpl := template.New("text")
	if _, ok := conf["_TEMPLATE_TEXT_FILE"]; ok {
		tmpl, err = tmpl.ParseFiles(conf["_TEMPLATE_TEXT_FILE"])
	} else if _, ok := conf["_TEMPLATE_TEXT"]; ok {
		tmpl, err = tmpl.Parse(conf["_TEMPLATE_TEXT"])
	} else {
		tmpl, err = tmpl.Parse(defTextTemplate)
	}

	if err != nil {
		log.Println(err)
		return 1
	}

	// save a text only flag now so we can delete the key
	textOnly := false
	_, textOnly = conf["_TEMPLATE_TEXT_ONLY"]

	var html *htmltemplate.Template

	// conditionally set-up non-text templates
	if !textOnly {
		html = htmltemplate.New("base")
		var contents string

		if _, ok := conf["_TEMPLATE_HTML_FILE"]; ok {
			contents, err = readFileString(conf["_TEMPLATE_HTML_FILE"])
			if err != nil {
				log.Println(err)
				return 1
			}
		} else if _, ok := conf["_TEMPLATE_HTML"]; ok {
			contents = conf["_TEMPLATE_HTML"]
		} else {
			contents = defHTMLTemplate
		}
		html, err = html.New("html").Parse(contents)
		if err != nil {
			log.Println(err)
			return 1
		}

		if _, ok := conf["_TEMPLATE_CSS_FILE"]; ok {
			contents, err = readFileString(conf["_TEMPLATE_CSS_FILE"])
			if err != nil {
				log.Println(err)
				return 1
			}
		} else if _, ok := conf["_TEMPLATE_CSS"]; ok {
			contents = conf["_TEMPLATE_CSS"]
		} else {
			contents = defCSSTemplate
		}
		html, err = html.New("css").Parse(contents)
		if err != nil {
			log.Println(err)
			return 1
		}

		// if _TEMPLATE_LOGO_FILE is defined, load that, else use embedded
		if logopath, ok := conf["_TEMPLATE_LOGO_FILE"]; ok {
			logofile, err := os.Open(logopath)
			if err != nil {
				return 1
			}
			m.EmbedReader("logo.png", logofile)
		} else {
			// use var path and a default of the embedded logo
			m.Embed("logo.png", mail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(logo)
				return err
			}))
		}
	}

	if !debug(conf) {
		keys := []string{
			"_TEMPLATE_TEXT", "_TEMPLATE_TEXT_FILE",
			"_TEMPLATE_HTML", "_TEMPLATE_HTML_FILE",
			"_TEMPLATE_CSS", "_TEMPLATE_CSS_FILE",
			"_TEMPLATE_TEXT_ONLY", "_TEMPLATE_LOGO_FILE",
			"_SUBJECT", "_ALERT_SUBJECT", "_CLEAR_SUBJECT",
			"_SUSPEND_SUBJECT", "_RESUME_SUBJECT", "_SUMMARY_SUBJECT",
		}
		for _, key := range keys {
			delete(conf, key)
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
	m.SetBody("text/plain", output.String())

	if !textOnly {
		var htmlBody bytes.Buffer
		err = html.ExecuteTemplate(&htmlBody, "html", conf)
		if err != nil {
			log.Println(err)
			return 1
		}
		m.AddAlternative("text/html", htmlBody.String())
	}

	err = d.DialAndSend(m)
	if err != nil {
		log.Println(err)
		return 1
	}
	return 0
}

type cardElement struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Weight string `json:"weight,omitempty"`
	Size   string `json:"size,omitempty"`
	Wrap   bool   `json:"wrap,omitempty"`
}

//export GoSendToMsTeamsWorkflow
func GoSendToMsTeamsWorkflow(n C.int, args **C.char) C.int {
	var msTeamsWebhooksValidity = make(map[string]bool)
	var clientTimeout time.Duration

	conf := parseArgs(n, args)

	if _, ok := conf["_TO"]; !ok || len(conf["_TO"]) == 0 {
		log.Println("ERR: No MsTeams Workflow URLs defined.")
		return 1
	}

	msTeamsWebhooks := strings.Split(conf["_TO"], "|")
	validityCount := 0

	var regexes []*regexp.Regexp
	for _, r := range webhookValidators {
		regexes = append(regexes, regexp.MustCompile(r))
	}

	for _, s := range msTeamsWebhooks {
		trimmedAddr := strings.TrimSpace(s)
		isValid := false
		for _, r := range regexes {
			if r.MatchString(trimmedAddr) {
				isValid = true
				break
			}
		}
		msTeamsWebhooksValidity[trimmedAddr] = isValid
		if isValid {
			validityCount++
		}
	}

	if validityCount == 0 {
		log.Println("ERR: No valid msTeams webhooks defined in _TO. Abort GoSendToMsTeamsChannel()")
		return 1
	}

	var subject string
	subject = defaultMsTeamsSubject[_SUBJECT]
	if _, ok := conf["_SUBJECT"]; ok && len(conf["_SUBJECT"]) != 0 {
		subject = getWithDefault("_SUBJECT", conf, defaultMsTeamsSubject[_SUBJECT])
	} else if _, ok = conf["_ALERT"]; ok {
		switch conf["_ALERT_TYPE"] {
		case "Alert":
			subject = getWithDefault("_ALERT_SUBJECT", conf, defaultMsTeamsSubject[_ALERT_SUBJECT])
		case "Clear":
			subject = getWithDefault("_CLEAR_SUBJECT", conf, defaultMsTeamsSubject[_CLEAR_SUBJECT])
		case "Suspend":
			subject = getWithDefault("_SUSPEND_SUBJECT", conf, defaultMsTeamsSubject[_SUSPEND_SUBJECT])
		case "Resume":
			subject = getWithDefault("_RESUME_SUBJECT", conf, defaultMsTeamsSubject[_RESUME_SUBJECT])
		case "ThrottleSummary":
			subject = getWithDefault("_SUMMARY_SUBJECT", conf, defaultMsTeamsSubject[_SUMMARY_SUBJECT])
		default:
			subject = getWithDefault("_SUBJECT", conf, defaultMsTeamsSubject[_SUBJECT])
		}
	}

	subjtmpl := template.New("subject")
	if tmpl, err := subjtmpl.Parse(subject); err == nil {
		var subjbuf bytes.Buffer
		if err = tmpl.Execute(&subjbuf, conf); err == nil {
			subject = subjbuf.String()
		}
	}
	header := replArgs(subject, conf)

	var htmltmpl *htmltemplate.Template
	var texttmpl *template.Template
	var textOnly, useHtmlTmpl bool
	var htmlOutput, textOutput bytes.Buffer
	var contents string
	var body string
	var err error

	_, textOnly = conf["TEMPLATE_TEXT_ONLY"]

	if val, ok := conf["_TEMPLATE_HTML_FILE"]; ok && len(val) != 0 {
		useHtmlTmpl = true
		contents, err = readFileString(val)
		if err == nil {
			htmltmpl, err = htmltemplate.New("html").Parse(contents)
		}
	} else if val, ok := conf["_TEMPLATE_HTML"]; ok && len(val) != 0 {
		useHtmlTmpl = true
		htmltmpl, err = htmltemplate.New("html").Parse(val)
	} else if val, ok := conf["_TEMPLATE_TEXT_FILE"]; ok && len(val) != 0 {
		useHtmlTmpl = false
		texttmpl, err = template.ParseFiles(val)
	} else if val, ok := conf["_TEMPLATE_TEXT"]; ok && len(val) != 0 {
		useHtmlTmpl = false
		texttmpl, err = template.New("text").Parse(val)
	} else if val, ok := conf["_FORMAT"]; ok && len(val) != 0 {
		if textOnly {
			useHtmlTmpl = false
			texttmpl, err = template.New("text").Parse(val)
		} else {
			useHtmlTmpl = true
			htmltmpl, err = htmltemplate.New("html").Parse(val)
		}
	} else if textOnly {
		useHtmlTmpl = false
		texttmpl, err = template.New("text").Parse(defMsTeamsTextTemplate)
	} else {
		useHtmlTmpl = true
		htmltmpl, err = htmltemplate.New("html").Parse(defMsTeamsHTMLTemplate)
	}

	if err != nil {
		log.Println("ERR: Template parsing failed. Abort.", err)
		return 1
	}

	if useHtmlTmpl {
		err = htmltmpl.ExecuteTemplate(&htmlOutput, "html", conf)
		body = replArgs(htmlOutput.String(), conf)
	} else {
		err = texttmpl.Execute(&textOutput, conf)
		body = replArgs(textOutput.String(), conf)
	}

	if err != nil {
		log.Println("ERR: Template execution failed. Abort.", err)
		return 1
	}

	type adaptiveCardPayload struct {
		Type        string `json:"type"`
		Attachments []struct {
			ContentType string `json:"contentType"`
			Content     struct {
				Schema  string                   `json:"$schema"`
				Type    string                   `json:"type"`
				Version string                   `json:"version"`
				Body    []map[string]interface{} `json:"body"`
			} `json:"content"`
		} `json:"attachments"`
	}

	payload := adaptiveCardPayload{Type: "message"}
	payload.Attachments = make([]struct {
		ContentType string `json:"contentType"`
		Content     struct {
			Schema  string                   `json:"$schema"`
			Type    string                   `json:"type"`
			Version string                   `json:"version"`
			Body    []map[string]interface{} `json:"body"`
		} `json:"content"`
	}, 1)

	payload.Attachments[0].ContentType = "application/vnd.microsoft.card.adaptive"
	payload.Attachments[0].Content.Schema = "http://adaptivecards.io/schemas/adaptive-card.json"
	payload.Attachments[0].Content.Type = "AdaptiveCard"
	payload.Attachments[0].Content.Version = "1.5"

	headerBlock := map[string]interface{}{
		"type":   "TextBlock",
		"text":   header,
		"weight": "Bolder",
		"size":   "Medium",
		"wrap":   true,
	}

	var bodyBlocks []map[string]interface{}
	trimmedBody := strings.TrimSpace(body)

	if strings.HasPrefix(trimmedBody, "[") && strings.HasSuffix(trimmedBody, "]") {
		err := json.Unmarshal([]byte(trimmedBody), &bodyBlocks)
		if err != nil {
			log.Printf("WARN: Template looked like JSON but failed to parse: %v. Using as plain text.", err)
			bodyBlocks = []map[string]interface{}{
				{"type": "TextBlock", "text": body, "wrap": true},
			}
		}
	} else {
		bodyBlocks = []map[string]interface{}{
			{"type": "TextBlock", "text": body, "wrap": true},
		}
	}

	payload.Attachments[0].Content.Body = append([]map[string]interface{}{headerBlock}, bodyBlocks...)

	jsonValue, err := json.Marshal(payload)
	if err != nil {
		log.Println("ERR: Cannot generate JSON data for Workflow API.", err)
		return 1
	}

	timeoutStr := getWithDefault("_TIMEOUT", conf, fmt.Sprintf("%d", DefaultMsTeamsTimeout))
	if timeout, err := strconv.Atoi(timeoutStr); err != nil {
		clientTimeout = DefaultMsTeamsTimeout * time.Millisecond
	} else {
		clientTimeout = time.Duration(timeout) * time.Millisecond
	}

	client := &http.Client{Timeout: clientTimeout}
	for url, isValid := range msTeamsWebhooksValidity {
		if isValid {
			request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
			if err != nil {
				log.Printf("ERR: Request creation failed for %s: %v", url, err)
				continue
			}
			request.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(request)
			if err != nil {
				log.Printf("ERR: HTTP POST to Workflow failed (target %s): %v", url, err)
			} else {
				log.Printf("INFO: Message sent to Workflow %s, status %d\n", url, resp.StatusCode)
				resp.Body.Close()
			}
		}
	}
	log.Println("INFO: GoSendToMsTeamsChannel() (Workflow) completed.")
	return 0
}
