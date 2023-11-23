package cmd

import (
	"os"
	"slices"
	"strings"
	"time"

	"github.com/go-mail/mail/v2"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/email"
	"github.com/rs/zerolog/log"
)

func setupEmail(toArg, ccArg, bccArg string) (em *config.Config) {
	em = config.New()
	// set default from yaml file, can be overridden from Geneos as
	// environment variables

	// creds can come from `geneos` credentials for the mail server
	// domain

	epassword := &config.Plaintext{}

	eusername := cf.GetString("email.username")
	smtpserver := cf.GetString("email.smtp")
	smtptls := cf.GetString("email.use-tls")

	if eusername != "" {
		epassword = cf.GetPassword("email.password")
	}

	if eusername == "" {
		creds := config.FindCreds(smtpserver, config.SetAppName("geneos"))
		if creds != nil {
			eusername = creds.GetString("username")
			epassword = creds.GetPassword("password")
		}
	}

	em.SetDefault("_smtp_username", eusername)
	em.SetDefault("_smtp_password", epassword.String())
	em.SetDefault("_smtp_server", smtpserver)
	em.SetDefault("_smtp_tls", smtptls)
	em.SetDefault("_smtp_port", cf.GetInt("email.port"))
	em.SetDefault("_from", cf.GetString("email.from"))
	em.SetDefault("_to", cf.GetString("email.to"))
	em.SetDefault("_cc", cf.GetString("email.cc"))
	em.SetDefault("_bcc", cf.GetString("email.bcc"))
	em.SetDefault("_subject", cf.GetString("email.subject"))

	for _, e := range os.Environ() {
		n := strings.SplitN(e, "=", 2)
		em.Set(n[0], n[1])
	}

	// override with args
	if toArg != "" {
		em.Set("_to", toArg)
	}
	if ccArg != "" {
		em.Set("_cc", ccArg)
	}
	if bccArg != "" {
		em.Set("_bcc", bccArg)
	}

	return
}

func sendEmail(em *config.Config, data DV2EMailData, inlineCSS bool) (err error) {
	d, err := email.Dial(em)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	run := time.Now()

	m, err := email.Envelope(em)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	m.SetHeader("Subject", em.GetString("_subject"))

	// attachments here

	// if not a multipart/alternative then always attach a plain
	// text part as the main body
	textString, err := createTextTemplate(cf, data, cf.GetString("text.template"))
	if err != nil {
		return
	}
	m.SetBody("text/plain", textString)

	if slices.Contains(cf.GetStringSlice("email.contents"), "text+html") {
		htmlString, err := createHTML(cf, data, cf.GetString("html.template"), inlineCSS)
		if err != nil {
			return err
		}
		m.AddAlternative("text/html", htmlString)
	}

	if slices.Contains(cf.GetStringSlice("email.contents"), "texttable") {
		files, err := buildTextTableFiles(cf, data, run)
		if err != nil {
			return err
		}
		for _, file := range files {
			m.AttachReader(file.name, file.content)
		}
	}

	if slices.Contains(cf.GetStringSlice("email.contents"), "html") {
		files, err := buildHTMLFiles(cf, data, run, inlineCSS)
		if err != nil {
			return err
		}
		for _, file := range files {
			m.AttachReader(file.name, file.content)
		}
	}

	if slices.Contains(cf.GetStringSlice("email.contents"), "xlsx") {
		files, err := buildXLSXFiles(cf, data, run)
		if err != nil {
			return err
		}

		for _, file := range files {
			m.AttachReader(file.name, file.content)
		}
	}

	if slices.Contains(cf.GetStringSlice("email.contents"), "images") {
		for name, path := range cf.GetStringMapString("images") {
			if _, err := os.Stat(path); err != nil {
				log.Error().Err(err).Msg("skipping")
				continue
			}
			m.Embed(path, mail.Rename(name), mail.SetHeader(map[string][]string{
				"X-Attachment-Id": {name},
			}))
		}
	}

	// send
	return d.DialAndSend(m)
}
