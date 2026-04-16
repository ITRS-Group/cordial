/*
Copyright © 2024 ITRS Group

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

package cmd

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"
	"unicode"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/wneessen/go-mail"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/reporter"
)

//go:embed _docs/email.md
var emailCmdDescription string

var emailCmdSubject, emailCmdFrom, emailCmdTo, emailCmdCc, emailCmdBcc string
var emailCmdContents string

func init() {
	GDNACmd.AddCommand(emailCmd)

	emailCmd.Flags().StringVarP(&reportNames, "report", "r", "", reportNamesDescription)

	emailCmd.Flags().StringVar(&emailCmdContents, "contents", "", "Override configured email contents")
	emailCmd.Flags().StringVar(&emailCmdSubject, "subject", "", "Override configured email Subject")
	emailCmd.Flags().StringVar(&emailCmdFrom, "from", "", "Override configured email From")
	emailCmd.Flags().StringVar(&emailCmdTo, "to", "", "Override configured email To\n(comma separated, but remember to quote as one argument)")
	emailCmd.Flags().StringVar(&emailCmdCc, "cc", "", "Override configured email Cc\n(comma separated, but remember to quote as one argument)")
	emailCmd.Flags().StringVar(&emailCmdBcc, "bcc", "", "Override configured email Bcc\n(comma separated, but remember to quote as one argument)")

	emailCmd.Flags().SortFlags = false
}

var emailCmd = &cobra.Command{
	Use:   "email",
	Short: "Email reports",
	Long:  emailCmdDescription,
	Args:  cobra.ArbitraryArgs,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	SilenceUsage:          true,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		// Handle SIGINT (CTRL+C) gracefully.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()

		db, err := openDB(ctx, cf, "db.dsn", false)
		if err != nil {
			return
		}
		defer db.Close()

		if emailCmdContents != "" {
			cf.BindPFlag("email.contents", cmd.Flags().Lookup("contents"))
		}
		if emailCmdSubject != "" {
			cf.BindPFlag("email.subject", cmd.Flags().Lookup("subject"))
		}
		if emailCmdFrom != "" {
			cf.BindPFlag("email.from", cmd.Flags().Lookup("from"))
		}
		if emailCmdTo != "" {
			cf.BindPFlag("email.to", cmd.Flags().Lookup("to"))
		}
		if emailCmdCc != "" {
			cf.BindPFlag("email.cc", cmd.Flags().Lookup("cc"))
		}
		if emailCmdBcc != "" {
			cf.BindPFlag("email.bcc", cmd.Flags().Lookup("bcc"))
		}

		return doEmail(ctx, cf, db, reportNames)
	},
}

type emailData struct {
	Timestamp      time.Time
	CSV            *reporter.ToolkitReporter
	Table          *reporter.FormattedReporter
	XLSX           *reporter.XLSXReporter
	TextBodyPart   *bytes.Buffer
	HTMLBodyPart   *bytes.Buffer
	XLSXAttachment *bytes.Buffer
	HTMLAttachment *bytes.Buffer
}

// doEmail is called by the `email` command or in the scheduler from the
// start command to send email reports as per the configuration in the
// top-level `email` configuration section.
func doEmail(ctx context.Context, cf *config.Config, db *sql.DB, reports string) (err error) {
	log.Info().Msgf("running email report")

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Error().Err(err).Msg("cannot BEGIN transaction")
		return
	}
	defer tx.Rollback()

	if err = updateReportingDatabase(ctx, cf, tx, nil); err != nil {
		return
	}

	// create a "data" struct to pass to templates, text tables etc.

	var data emailData
	data.Timestamp = time.Now()

	// always build a multipart body
	data.HTMLBodyPart = &bytes.Buffer{}
	r, _ := reporter.NewReporter("html", data.HTMLBodyPart,
		reporter.HTMLPreamble(config.Get[string](cf, cf.Join("email", "html-preamble"))),
		reporter.HTMLPostscript(config.Get[string](cf, cf.Join("email", "html-postscript"))),
		reporter.Scramble(config.Get[bool](cf, cf.Join("email", "scramble"))),
		reporter.DataviewCSSClass("gdna-dataview"),
		reporter.HeadlineCSSClass("gdna-headlines"),
	)
	runReports(ctx, cf, tx, r, config.Get[string](cf, cf.Join("email", "body-reports")), -1)
	r.Render()
	r.Close()
	log.Debug().Msgf("text+HTML report complete, %d bytes", data.HTMLBodyPart.Len())

	data.TextBodyPart = &bytes.Buffer{}
	r, _ = reporter.NewReporter("table", data.TextBodyPart,
		reporter.Scramble(config.Get[bool](cf, cf.Join("email", "scramble"))),
	)
	runReports(ctx, cf, tx, r, config.Get[string](cf, cf.Join("email", "body-reports")), -1)
	r.Render()
	r.Close()
	log.Debug().Msgf("TEXT+html report complete, %d bytes", data.TextBodyPart.Len())

	var contents []string
	c := config.Get[any](cf, cf.Join("email", "contents"))
	switch c2 := c.(type) {
	case []any:
		for _, v := range c2 {
			contents = append(contents, fmt.Sprint(v))
		}
	case string:
		contents = strings.FieldsFunc(c2, func(r rune) bool { return unicode.IsSpace(r) || r == ',' })
	default:
		log.Fatal().Msgf("content type %T", c2)
	}
	for _, c := range contents {
		switch c {
		case "html":
			if data.HTMLAttachment != nil {
				log.Debug().Msg("HTML content already initialised")
				continue
			}
			data.HTMLAttachment = &bytes.Buffer{}
			r, _ := reporter.NewReporter("html", data.HTMLAttachment,
				reporter.HTMLPreamble(config.Get[string](cf, cf.Join("email", "html-preamble"))),
				reporter.HTMLPostscript(config.Get[string](cf, cf.Join("email", "html-postscript"))),
				reporter.Scramble(config.Get[bool](cf, cf.Join("email", "scramble"))),
				reporter.DataviewCSSClass("gdna-dataview"),
				reporter.HeadlineCSSClass("gdna-headlines"),
			)
			runReports(ctx, cf, tx, r, reports, -1)
			r.Render()
			r.Close()
			log.Debug().Msgf("HTML report complete, %d bytes", data.HTMLAttachment.Len())
		case "xlsx":
			data.XLSXAttachment = &bytes.Buffer{}
			r, _ := reporter.NewReporter("xlsx", data.XLSXAttachment,
				reporter.SummarySheetName(config.Get[string](cf, cf.Join("reports", "gdna-summary", "name"))),
				reporter.XLSXScramble(config.Get[bool](cf, cf.Join("email", "scramble"))),
				reporter.XLSXPassword(config.Get[*config.Plaintext](cf, cf.Join("xlsx", "password"))),
				reporter.DateFormat(config.Get[string](cf, cf.Join("xlsx", "formats", "datetime"), config.DefaultValue("yyyy-mm-ddThh:MM:ss"))),
				reporter.IntFormat(config.Get[int](cf, cf.Join("xlsx", "formats", "int"), config.DefaultValue(1))),
				reporter.PercentFormat(config.Get[int](cf, cf.Join("xlsx", "formats", "percent"), config.DefaultValue(9))),
				reporter.SeverityColours(
					config.Get[string](cf, cf.Join("xlsx", "conditional-formats", "undefined"), config.DefaultValue("BFBFBF")),
					config.Get[string](cf, cf.Join("xlsx", "conditional-formats", "ok"), config.DefaultValue("5BB25C")),
					config.Get[string](cf, cf.Join("xlsx", "conditional-formats", "warning"), config.DefaultValue("F9B057")),
					config.Get[string](cf, cf.Join("xlsx", "conditional-formats", "critical"), config.DefaultValue("FF5668")),
				),
				reporter.MinColumnWidth(config.Get[float64](cf, cf.Join("xlsx", "formats", "min-width"))),
				reporter.MaxColumnWidth(config.Get[float64](cf, cf.Join("xlsx", "formats", "max-width"))),
				reporter.XLSXHeadlines(config.Get[int](cf, cf.Join("xlsx", "headlines"))),
			)
			runReports(ctx, cf, tx, r, reports, -1)
			r.Render()
			r.Close()
			log.Debug().Msgf("XLSX report complete, %d bytes", data.XLSXAttachment.Len())
		default:
		}
	}

	// commit any updates to database even if email fails to send as the
	// data is a snapshot at the time and subsequent email reports will
	// updates the data to a more recent set anyway.
	err = tx.Commit()
	if err != nil {
		log.Error().Err(err).Msg("email report failed")
		return
	}

	err = sendMail(cf, data)
	if err != nil {
		log.Error().Err(err).Msg("email report failed")
		return
	}
	log.Info().Msg("email report complete")
	return
}

func emailConfToString(a any) string {
	switch v := a.(type) {
	case string:
		return v
	case []string:
		return strings.Join(v, ",")
	case []any:
		l := []string{}
		for _, n := range v {
			l = append(l, fmt.Sprint(n))
		}
		return strings.Join(l, ",")
	default:
		return ""
	}
}

func sendMail(cf *config.Config, data emailData) (err error) {
	m := mail.NewMsg()
	if err = m.From(cf.GetString("email.from")); err != nil {
		err = fmt.Errorf("%w: setting From", err)
		return
	}
	if err = m.ToFromString(emailConfToString(config.Get[any](cf, "email.to"))); err != nil {
		err = fmt.Errorf("%w: setting To", err)
		return
	}
	if len(config.Get[[]string](cf, "email.cc")) > 0 {
		if err = m.CcFromString(emailConfToString(config.Get[any](cf, "email.cc"))); err != nil {
			err = fmt.Errorf("%w: setting Cc", err)
			return
		}
	}
	if len(config.Get[[]string](cf, "email.bcc")) > 0 {
		if err = m.BccFromString(emailConfToString(config.Get[any](cf, "email.bcc"))); err != nil {
			err = fmt.Errorf("%w: setting Bcc", err)
			return
		}
	}
	m.Subject(cf.GetString("email.subject", config.DefaultValue("ITRS GDNA EMail Report")))

	// we either have a multipart body or text or html - but we have to
	// have something
	if data.TextBodyPart != nil {
		m.SetBodyString("text/plain", data.TextBodyPart.String())
		if data.HTMLBodyPart != nil {
			m.AddAlternativeString("text/html", data.HTMLBodyPart.String())
		}
	} else if data.HTMLBodyPart != nil {
		m.SetBodyString("text/html", data.HTMLBodyPart.String())
	} else {
		err = errors.New("no text or html body defined")
		return
	}

	lookupDateTime := map[string]string{
		"date":     data.Timestamp.Local().Format("20060102"),
		"time":     data.Timestamp.Local().Format("150405"),
		"datetime": data.Timestamp.Local().Format(time.RFC3339),
	}

	if data.XLSXAttachment != nil {
		m.AttachReader(cf.GetString("email.xlsx-name", config.LookupTable(lookupDateTime)), data.XLSXAttachment)
	}

	if data.HTMLAttachment != nil {
		m.AttachReader(cf.GetString("email.html-name", config.LookupTable(lookupDateTime)), data.HTMLAttachment)
	}

	// build smtp connection details
	var tlsPolicy mail.TLSPolicy

	switch strings.ToLower(cf.GetString("email.tls", config.DefaultValue("default"))) {
	case "force":
		tlsPolicy = mail.TLSMandatory
	case "none":
		tlsPolicy = mail.NoTLS
	default:
		tlsPolicy = mail.TLSOpportunistic
	}

	password := &config.Plaintext{}

	username := cf.GetString("email.username")
	server := cf.GetString("email.smtp-server", config.DefaultValue("localhost"))

	if username != "" {
		password = config.Get[*config.Plaintext](cf, "email.password")
	}

	if username == "" {
		creds := config.FindCreds(server,
			config.SetAppName("geneos"),
			config.SetConfigFile(cf.GetString("email.credentials-file")),
		)
		if creds != nil {
			username = creds.GetString("username")
			password = config.Get[*config.Plaintext](creds, "password", config.UseKeyfile(cf.GetString("email.key-file")))
		}
	}

	mailOpts := []mail.Option{
		mail.WithTLSPortPolicy(tlsPolicy),
		mail.WithTimeout(time.Duration(config.Get[int](cf, "_smtp_timeout", config.DefaultValue(10))) * time.Second),
	}

	if username != "" {
		mailOpts = append(mailOpts,
			mail.WithUsername(username),
			mail.WithPassword(password.String()),
			mail.WithSMTPAuth(mail.SMTPAuthLogin),
		)
	} else {
		mailOpts = append(mailOpts,
			mail.WithSMTPAuth(mail.SMTPAuthNoAuth),
		)
	}

	// override port policy if we are told to, but zero skips through
	// sometimes, so check that too
	if cf.IsSet("email.port") && config.Get[uint16](cf, cf.Join("email", "port")) != 0 {
		mailOpts = append(mailOpts, mail.WithPort(int(config.Get[uint16](cf, cf.Join("email", "port")))))
	}

	if config.Get[bool](cf, cf.Join("email", "tls-insecure")) {
		mailOpts = append(mailOpts, mail.WithTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		}))
	}

	d, err := mail.NewClient(server, mailOpts...)
	if err != nil {
		return
	}

	d.SetTLSPolicy(tlsPolicy)
	return d.DialAndSend(m)
}
