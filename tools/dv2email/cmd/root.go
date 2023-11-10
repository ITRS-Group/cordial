/*
Copyright © 2023 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package cmd

import (
	_ "embed"
	"errors"
	"fmt"
	htemplate "html/template"
	"io"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	ttemplate "text/template"

	"github.com/aymerick/douceur/inliner"
	"github.com/go-mail/mail/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/commands"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/email"
	"github.com/itrs-group/cordial/pkg/xpath"
)

var cfgFile string
var execname string
var debug, quiet bool
var inlineCSS bool

var entityArg, samplerArg, typeArg, dataviewArg string
var toArg, ccArg, bccArg string

func init() {
	cobra.OnInitialize(initConfig)

	config.DefaultKeyDelimiter("::")

	DV2EMAILCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	DV2EMAILCmd.PersistentFlags().BoolVarP(&inlineCSS, "inline-css", "i", true, "inline CSS for better mail client support")
	DV2EMAILCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "config file (default is $HOME/.config/geneos/dv2email.yaml)")

	// how to remove the help flag help text from the help output! Sigh...
	DV2EMAILCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	DV2EMAILCmd.PersistentFlags().MarkHidden("help")

	DV2EMAILCmd.Flags().StringVarP(&entityArg, "entity", "E", "", "entity name")
	DV2EMAILCmd.Flags().StringVarP(&samplerArg, "sampler", "S", "", "sampler name")
	DV2EMAILCmd.Flags().StringVarP(&typeArg, "type", "T", "", "type name")
	DV2EMAILCmd.Flags().StringVarP(&dataviewArg, "dataview", "D", "", "dataview name")

	DV2EMAILCmd.Flags().StringVarP(&toArg, "to", "t", "", "To as comma-separated emails")
	DV2EMAILCmd.Flags().StringVarP(&ccArg, "cc", "c", "", "Cc as comma-separated emails")
	DV2EMAILCmd.Flags().StringVarP(&bccArg, "bcc", "b", "", "Bcc as comma-separated emails")

	DV2EMAILCmd.Flags().SortFlags = false

	execname = cordial.ExecutableName()
	cordial.LogInit(execname)
}

var cf *config.Config

func initConfig() {
	var err error
	if quiet {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	} else if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	cf, err = config.Load(execname,
		config.SetAppName("geneos"),
		config.SetConfigFile(cfgFile),
		config.MergeSettings(),
		config.SetFileExtension("yaml"),
	)
	if err != nil {
		cfg := config.Path(execname,
			config.SetAppName("geneos"),
			config.SetConfigFile(cfgFile),
			config.MergeSettings(),
			config.SetFileExtension("yaml"),
		)
		log.Fatal().Err(err).Msgf("loading from file: %s", cfg)
	}
}

type dv2emailData struct {
	Dataviews []*commands.Dataview
	Env       map[string]string
}

//go:embed dv2email-text.gotmpl
var textDefaultTemplate string

//go:embed dv2email-html.gotmpl
var htmlDefaultTemplate string

//go:embed _docs/root.md
var DV2EMAILCmdDescription string

// DV2EMAILCmd represents the base command when called without any subcommands
var DV2EMAILCmd = &cobra.Command{
	Use:                "dv2email",
	Short:              "Email a Dataview following Geneos Action/Effect conventions",
	Long:               DV2EMAILCmdDescription,
	DisableAutoGenTag:  true,
	DisableSuggestions: true,
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		gwcf := cf.Sub("gateway")
		gwcf.SetDefault("allow-insecure", true)

		u := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", gwcf.GetString("host", config.Default("localhost")), gwcf.GetInt("port", config.Default(7038))),
		}

		gwcf.SetDefault("use-tls", true)
		if gwcf.GetBool("use-tls") {
			u.Scheme = "https"
		}

		password := &config.Plaintext{}

		username := gwcf.GetString("username")
		gateway := gwcf.GetString("name")

		if username != "" {
			password = gwcf.GetPassword("password")
		}

		if username == "" {
			var creds *config.Config
			if gateway != "" {
				creds = config.FindCreds("gateway:"+gateway, config.SetAppName("geneos"))
			} else {
				creds = config.FindCreds("gateway", config.SetAppName("geneos"))
			}
			if creds != nil {
				username = creds.GetString("username")
				password = creds.GetPassword("password")
			}
		}

		gw, err := commands.DialGateway(u,
			commands.SetBasicAuth(username, password),
			commands.AllowInsecureCertificates(gwcf.GetBool("allow-insecure")),
		)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		htmlTemplate := cf.GetString("html-template", config.Default(htmlDefaultTemplate))
		ht, err := htemplate.New("dataview").Parse(htmlTemplate)
		if err != nil {
			return
		}

		textTemplate := cf.GetString("text-template", config.Default(textDefaultTemplate))
		tt, err := ttemplate.New("dataview").Parse(textTemplate)
		if err != nil {
			return
		}

		tmplData := dv2emailData{
			Dataviews: []*commands.Dataview{},
			Env:       make(map[string]string, len(os.Environ())),
		}

		// import all environment vars into both the template data and
		// also the config structure (config.WithEnvs doesn't work for
		// empty prefixes)
		for _, e := range os.Environ() {
			n := strings.SplitN(e, "=", 2)
			tmplData.Env[n[0]] = n[1]
			cf.Set(n[0], n[1])
		}

		varpath := cf.GetString("_variablepath")
		if varpath == "" {
			varpath = "//managedEntity"
			if entityArg != "" {
				varpath += fmt.Sprintf("[(@name=%q)]", entityArg)
			}
			varpath += "/sampler"
			if samplerArg != "" {
				varpath += fmt.Sprintf("[(@name=%q)][(@type=%q)]", samplerArg, typeArg)
			}
			varpath += "/dataview"
			if dataviewArg != "" {
				varpath += fmt.Sprintf("[(@name=%q)]", dataviewArg)
			}
		}
		dv, err := xpath.Parse(varpath)
		dv = dv.ResolveTo(&xpath.Dataview{})

		if err != nil {
			return
		}

		dataviews, err := gw.Match(dv, 0)
		if err != nil {
			return
		}

		if len(dataviews) == 0 {
			return errors.New("no matching dataviews found")
		}

		em := config.New()
		// set default from yaml file, can be overridden from Geneos

		// creds can come from `geneos` credentials for the mail server
		// domain

		epassword := &config.Plaintext{}

		emcf := cf.Sub("email")

		eusername := emcf.GetString("username")
		smtpserver := emcf.GetString("smtp", config.Default("localhost"))
		smtptls := emcf.GetString("use-tls", config.Default("default"))

		if eusername != "" {
			epassword = emcf.GetPassword("password")
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
		em.SetDefault("_smtp_port", emcf.GetInt("port", config.Default(25)))
		em.SetDefault("_from", emcf.GetString("from"))
		em.SetDefault("_to", emcf.GetString("to"))
		em.SetDefault("_cc", emcf.GetString("cc"))
		em.SetDefault("_bcc", emcf.GetString("bcc"))
		em.SetDefault("_subject", emcf.GetString("subject", config.Default("Geneos Alert")))

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

		for _, dataview := range dataviews {
			data, err := gw.Snapshot(dataview, commands.Scope{Value: true, Severity: true})
			if err != nil {
				log.Error().Err(err).Msg("")
				continue
			}

			// filter here

			headlines := match(data.Name, "headline-filter", "__headlines", em)
			if len(headlines) > 0 {
				nh := map[string]commands.DataItem{}
				for _, h := range headlines {
					h = strings.TrimSpace(h)
					for oh, headline := range data.Headlines {
						if ok, err := path.Match(h, oh); err == nil && ok {
							nh[oh] = headline
						}
					}
				}
				data.Headlines = nh
			}

			// the first column is either from `first-column` in config
			// (matched against the dataview name) or from the
			// environment variable _FIRSTCOLUMN or `rowname` and is
			// always the first column.
			var rowname string
			defaultRowName := match(data.Name, "first-column", "_firstcolumn", em)
			if len(defaultRowName) > 0 {
				rowname = defaultRowName[0]
			} else {
				rowname = em.GetString("_firstcolumn", config.Default("rowname"))
			}
			// set the default, may be overridden below but then reset
			// to the same value
			data.Columns[0] = rowname

			cols := match(data.Name, "column-filter", "__columns", em)
			if len(cols) > 0 {
				nc := []string{rowname}
				for _, c := range cols {
					c = strings.TrimSpace(c)
					for _, oc := range data.Columns {
						if oc == rowname {
							continue
						}
						if ok, err := path.Match(c, oc); err == nil && ok {
							nc = append(nc, oc)
						}
					}
				}

				data.Columns = nc
			}

			rows := match(data.Name, "row-filter", "__rows", em)
			if len(rows) > 0 {
				nr := map[string]map[string]commands.DataItem{}
				for _, r := range rows {
					r = strings.TrimSpace(r)
					for rowname, row := range data.Table {
						if ok, err := path.Match(r, rowname); err == nil && ok {
							nr[rowname] = row
						}
					}
				}
				data.Table = nr
			}

			// default sort rows
			sort.Strings(data.Rows)

			asc := true
			matches := matchForName(data.Name, "row-order")
			if len(matches) > 0 {
				m := matches[0]
				switch {
				case strings.HasSuffix(m, "-"):
					asc = false
					m = m[:len(m)-1]
				case strings.HasSuffix(m, "+"):
					m = m[:len(m)-1]
					fallthrough
				default:
					asc = true
				}

				// if the row-order is for a column that is used as the
				// rowname (decided above in Column ordering) then sort
				// the data.Rows slice directly based on value and
				// not a cell in the row
				if m == "rowname" || m == data.Columns[0] {
					sort.Slice(data.Rows, func(i, j int) bool {
						if asc {
							return data.Rows[i] < data.Rows[j]
						} else {
							return data.Rows[j] < data.Rows[i]
						}
					})
				} else {
					sort.Slice(data.Rows, func(i, j int) bool {
						r := data.Rows
						a := data.Table[r[i]][m].Value
						af, _ := strconv.ParseFloat(a, 64)
						b := data.Table[r[j]][m].Value
						bf, _ := strconv.ParseFloat(b, 64)
						if a == b {
							if asc {
								return a < b
							} else {
								return a > b
							}
						}
						if asc {
							return af < bf
						}
						return bf < af
					})
				}
			}

			tmplData.Dataviews = append(tmplData.Dataviews, data)

			if err != nil {
				log.Error().Err(err).Msg("")
				continue
			}

		}

		d, err := email.Dial(em)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		m, err := email.Envelope(em)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}
		m.SetHeader("Subject", em.GetString("_subject"))

		m.SetBodyWriter("text/plain", func(w io.Writer) error {
			return tt.Execute(w, tmplData)
		})

		if inlineCSS {
			var body strings.Builder
			err = ht.Execute(&body, tmplData)
			if err != nil {
				return
			}
			var inlined string
			inlined, err = inliner.Inline(body.String())
			if err != nil {
				return
			}
			m.AddAlternative("text/html", inlined)
		} else {
			m.AddAlternativeWriter("text/html", func(w io.Writer) error {
				return ht.Execute(w, tmplData)
			})
		}

		for name, path := range cf.GetStringMapString("images") {
			if _, err := os.Stat(path); err != nil {
				log.Error().Err(err).Msg("skipping")
				continue
			}
			m.Embed(path, mail.Rename(name))
		}

		err = d.DialAndSend(m)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		return
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	cordial.RenderHelpAsMD(DV2EMAILCmd)

	err := DV2EMAILCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// match will return either a fixed list of matches from a comma
// separated env (if set in config instance em) or it will search for a
// section in the global configuration instance for `confkey` and return
// the values for the longest matching key (using globbing rules)
func match(name, confkey, env string, em *config.Config) (matches []string) {
	if em.IsSet(env) {
		matches = strings.Split(em.GetString(env), ",")
	} else {
		matches = matchForName(name, confkey)
	}
	return
}

// matchForName returns a the first slice of values of the member of confkey that
// matches name using globbing rules, longest match wins. e.g. for
//
// confkey:
//
//	col*: this, that, other
//	columnFullName: only, these
//
// and if name is 'columnFullName' then [ 'only', 'these' ] is returned.
func matchForName(name, confkey string) (matches []string) {
	checks := cf.GetStringMapStringSlice(confkey)
	keys := []string{}
	for k := range checks {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})
	for _, m := range keys {
		if ok, _ := path.Match(m, name); ok {
			matches = checks[m]
			break
		}
	}
	return
}
