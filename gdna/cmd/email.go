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
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"time"
	"unicode"

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
	Cmd.AddCommand(emailCmd)

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
	log.Info("running email report")

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("cannot BEGIN transaction", slog.Any("error", err))
		return
	}
	defer tx.Rollback()

	if err = updateReportingDatabase(ctx, cf, tx, nil); err != nil {
		return
	}

	// create a "data" struct to pass to templates, text tables etc.

	var data emailData
	data.Timestamp = time.Now()

	bodyReports := config.Get[string](cf, cf.Join("email", "body-reports"))
	scramble := config.Get[bool](cf, cf.Join("email", "scramble"))

	// always build a multipart body
	data.HTMLBodyPart = &bytes.Buffer{}
	if err = emailPart(ctx, cf, tx, "html", data.HTMLBodyPart, bodyReports,
		reporter.HTMLPreamble(config.Get[string](cf, cf.Join("email", "html-preamble"))),
		reporter.HTMLPostscript(config.Get[string](cf, cf.Join("email", "html-postscript"))),
		reporter.Scramble(scramble),
		reporter.DataviewCSSClass("gdna-dataview"),
		reporter.HeadlineCSSClass("gdna-headlines"),
	); err != nil {
		return fmt.Errorf("text+HTML report: %w", err)
	}
	log.Debug("text+HTML report complete", slog.Int("bytes", data.HTMLBodyPart.Len()))

	data.TextBodyPart = &bytes.Buffer{}
	if err = emailPart(ctx, cf, tx, "table", data.TextBodyPart, bodyReports,
		reporter.Scramble(scramble),
	); err != nil {
		return fmt.Errorf("TEXT+html report: %w", err)
	}
	log.Debug("TEXT+html report complete", slog.Int("bytes", data.TextBodyPart.Len()))

	contents, err := emailContents(config.Get[any](cf, cf.Join("email", "contents")))
	if err != nil {
		return err
	}
	for _, c := range contents {
		switch c {
		case "html":
			if data.HTMLAttachment != nil {
				log.Debug("HTML content already initialised")
				continue
			}
			data.HTMLAttachment = &bytes.Buffer{}
			if err = emailPart(ctx, cf, tx, "html", data.HTMLAttachment, reports,
				reporter.HTMLPreamble(config.Get[string](cf, cf.Join("email", "html-preamble"))),
				reporter.HTMLPostscript(config.Get[string](cf, cf.Join("email", "html-postscript"))),
				reporter.Scramble(scramble),
				reporter.DataviewCSSClass("gdna-dataview"),
				reporter.HeadlineCSSClass("gdna-headlines"),
			); err != nil {
				return fmt.Errorf("HTML report: %w", err)
			}
			log.Debug("HTML report complete", slog.Int("bytes", data.HTMLAttachment.Len()))
		case "xlsx":
			data.XLSXAttachment = &bytes.Buffer{}
			if err = emailPart(ctx, cf, tx, "xlsx", data.XLSXAttachment, reports, xlsxOptions(cf, scramble)...); err != nil {
				return fmt.Errorf("XLSX report: %w", err)
			}
			log.Debug("XLSX report complete", slog.Int("bytes", data.XLSXAttachment.Len()))
		default:
		}
	}

	// commit any updates to database even if email fails to send as the
	// data is a snapshot at the time and subsequent email reports will
	// updates the data to a more recent set anyway.
	err = tx.Commit()
	if err != nil {
		log.Error("email report failed", slog.Any("error", err))
		return
	}

	err = sendMail(cf, data)
	if err != nil {
		log.Error("email report failed", slog.Any("error", err))
		return
	}
	log.Info("email report complete")
	return
}

// emailPart creates a reporter of the given format that writes to w, runs
// the reports that match reports and renders the result. The error of any
// report that failed is returned, so that an email is not sent with
// output that is incomplete.
func emailPart(ctx context.Context, cf *config.Config, tx *sql.Tx, format string, w io.Writer, reports string, options ...any) (err error) {
	r, err := reporter.NewReporter(format, w, options...)
	if err != nil {
		return
	}
	if err = runReports(ctx, cf, tx, r, reports, -1); err != nil {
		return errors.Join(err, r.Close())
	}
	if err = r.Render(); err != nil {
		return errors.Join(err, r.Close())
	}
	return r.Close()
}

// emailContents returns the formats to attach to the email from the
// `email.contents` setting, which is a list or a string of formats
// separated by white space or commas
func emailContents(c any) (contents []string, err error) {
	switch c2 := c.(type) {
	case []any:
		for _, v := range c2 {
			contents = append(contents, fmt.Sprint(v))
		}
	case []string:
		contents = c2
	case string:
		contents = strings.FieldsFunc(c2, func(r rune) bool { return unicode.IsSpace(r) || r == ',' })
	default:
		err = fmt.Errorf("email.contents must be a list or a string, not %T", c)
	}
	return
}

// smtpTimeout returns the timeout for the connection to the SMTP
// server, which is the documented `email.timeout`. The undocumented
// `_smtp_timeout`, in seconds, is only used, as it was before, if it is
// set.
func smtpTimeout(cf *config.Config) (timeout time.Duration) {
	if cf.IsSet("_smtp_timeout") {
		timeout = time.Duration(config.Get[int](cf, "_smtp_timeout")) * time.Second
	} else {
		timeout = config.Get[time.Duration](cf, cf.Join("email", "timeout"))
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
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
	if err = m.From(config.Get[string](cf, "email.from")); err != nil {
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
	m.Subject(config.Get[string](cf, "email.subject", config.DefaultValue("ITRS GDNA EMail Report")))

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
		m.AttachReader(config.Get[string](cf, "email.xlsx-name", config.LookupTable(lookupDateTime)), data.XLSXAttachment)
	}

	if data.HTMLAttachment != nil {
		m.AttachReader(config.Get[string](cf, "email.html-name", config.LookupTable(lookupDateTime)), data.HTMLAttachment)
	}

	// build smtp connection details
	var tlsPolicy mail.TLSPolicy

	switch strings.ToLower(config.Get[string](cf, "email.tls", config.DefaultValue("default"))) {
	case "force":
		tlsPolicy = mail.TLSMandatory
	case "none":
		tlsPolicy = mail.NoTLS
	default:
		tlsPolicy = mail.TLSOpportunistic
	}

	var password config.Secret

	username := config.Get[string](cf, "email.username")
	server := config.Get[string](cf, "email.smtp-server", config.DefaultValue("localhost"))

	if username != "" {
		password = config.Get[config.Secret](cf, "email.password")
		defer clear(password)
	}

	if username == "" {
		creds := config.FindCreds(server,
			config.AppName("geneos"),
			config.FilePath(config.Get[string](cf, "email.credentials-file")),
		)
		if creds != nil {
			username = config.Get[string](creds, "username")
			password = config.Get[config.Secret](creds, "password", config.UseKeyfile(config.Get[string](cf, "email.key-file")))
			defer clear(password)
		}
	}

	mailOpts := []mail.Option{
		mail.WithTLSPortPolicy(tlsPolicy),
		mail.WithTimeout(smtpTimeout(cf)),
	}

	if username != "" {
		mailOpts = append(mailOpts,
			mail.WithUsername(username),
			mail.WithPassword(string(password)),
			mail.WithSMTPAuth(mail.SMTPAuthLogin),
		)
	} else {
		mailOpts = append(mailOpts,
			mail.WithSMTPAuth(mail.SMTPAuthNoAuth),
		)
	}

	// override port policy if we are told to, but zero skips through
	// sometimes, so check that too
	if p, ok := config.Lookup[int](cf, cf.Join("email", "port")); ok && p != 0 {
		mailOpts = append(mailOpts, mail.WithPort(p))
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
