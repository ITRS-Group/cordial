package main

import "C"
import (
	"bytes"
	_ "embed"
	htmltemplate "html/template"
	"io"
	"log"
	"os"
	"regexp"
	"text/template"

	"github.com/go-mail/mail/v2"
)

//go:embed text.gotmpl
var defTextTemplate string

//go:embed html.gotmpl
var defHTMLTemplate string

//go:embed css.gotmpl
var defCSSTemplate string

//go:embed logo.png
var logo []byte

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

	if _, ok := conf["_FORMAT"]; ok {
		format = conf["_FORMAT"]
		subject = getWithDefault("_SUBJECT", conf, defaultSubject[_SUBJECT])
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
			subject = getWithDefault("_SUBJECT", conf, defaultSubject[_SUBJECT])
		}
	} else {
		format = defaultFormat[_FORMAT]
		subject = defaultSubject[_SUBJECT]
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
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	if err = d.DialAndSend(m); err != nil {
		log.Println(err)
		return 1
	}
	return 0
}

// substitue placeholder of the form %(XXX) for the value of XXX or empty and
// return the result as a new string
func replArgs(format string, conf EMailConfig) string {
	re := regexp.MustCompile(`%\([^\)]*\)`)
	result := re.ReplaceAllStringFunc(format, func(key string) string {
		// strip containing "%(...)" - as we are here, the regexp must have matched OK
		// so no further check required
		key = key[2 : len(key)-1]
		return conf[key]
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
	subject := defaultSubject[_SUBJECT]

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
		default:
			subject = getWithDefault("_SUBJECT", conf, defaultSubject[_SUBJECT])
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
