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
	"fmt"
	"html/template"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/commands"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/email"
	"github.com/itrs-group/cordial/pkg/xpath"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dv2html",
	Short: "Email a Dataview following Geneos Action/Effect conventions",
	Long: strings.ReplaceAll(`
Email a Dataview following Geneos Action/Effect conventions.

When called without a sub-command and no arguments the program
processes environment variables setup as per Geneos Action/Effect
conventions and constructs an HTML Email of the dataview the data
item is from.

Settings for the Gateway REST connection and defaults for the EMail
gateway can be located in dv2html.yaml (either in the working
directory or in the user's .config/dv2html directory)
	`, "|", "`"),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		cf := config.GetConfig()
		u := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", cf.GetString("host"), cf.GetInt("port")),
		}
		cf.SetDefault("use-tls", true)
		if cf.GetBool("use-tls") {
			u.Scheme = "https"
		}
		gw, err := commands.DialGateway(u,
			commands.SetBasicAuth(cf.GetString("username"), cf.GetString("password")),
			commands.AllowInsecureCertificates(cf.GetBool("allow-insecure")))
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		htmlTemplate := htmlDefaultTemplate
		if h := cf.GetString("html-template"); h != "" {
			htmlTemplate = h
		}

		t, err := template.New("dataview").Parse(htmlTemplate)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		cf.SetDefault("css-data", cssData)

		tmplData := templateData{
			CSSURL:    cf.GetString("css-url"),
			CSSDATA:   template.CSS(cf.GetString("css-data")),
			Dataviews: []*commands.Dataview{},
			Env:       make(map[string]string, len(os.Environ())),
		}

		for _, e := range os.Environ() {
			n := strings.SplitN(e, "=", 2)
			tmplData.Env[n[0]] = n[1]
		}

		dv, err := xpath.Parse(cf.GetString("_variablepath"))
		dv = dv.ResolveTo(&xpath.Dataview{})

		if err != nil {
			log.Error().Err(err).Msg("")
			return
		}

		paths, err := gw.Match(dv, 0)
		if err != nil {
			log.Error().Err(err).Msg("")
			return
		}

		if len(paths) == 0 {
			log.Fatal().Msg("no matching dataviews found")
		}

		em := config.New()
		// set default from yaml file, can be overridden from Geneos
		em.SetDefault("_smtp_username", cf.GetString("email.username"))
		em.SetDefault("_smtp_password", cf.GetString("email.password", config.RawString()))
		em.SetDefault("_smtp_server", cf.GetString("email.smtp", config.Default("localhost")))
		em.SetDefault("_smtp_port", cf.GetInt("email.port", config.Default(25)))
		em.SetDefault("_from", cf.GetString("email.from"))
		em.SetDefault("_to", cf.GetString("email.to"))
		em.SetDefault("_subject", cf.GetString("email.subject", config.Default("Geneos Alert")))

		for _, e := range os.Environ() {
			n := strings.SplitN(e, "=", 2)
			em.Set(n[0], n[1])
		}

		for _, x := range paths {
			data, err := gw.Snapshot(x, commands.Scope{Value: true, Severity: true})
			if err != nil {
				log.Error().Err(err).Msg("")
				continue
			}

			tmplData.Dataviews = append(tmplData.Dataviews, data)

			// filter here
			if em.IsSet("__headlines") {
				nh := map[string]commands.DataItem{}
				headlines := strings.Split(em.GetString("__headlines"), ",")
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

			if em.IsSet("__columns") {
				nc := []string{"rowname"}
				cols := strings.Split(em.GetString("__columns"), ",")
				for _, c := range cols {
					c = strings.TrimSpace(c)
					for _, oc := range data.Columns {
						if oc == "rowname" {
							continue
						}
						if ok, err := path.Match(c, oc); err == nil && ok {
							nc = append(nc, oc)
						}
					}
				}
				data.Columns = nc
			}

			if em.IsSet("__rows") {
				nr := map[string]map[string]commands.DataItem{}
				rows := strings.Split(em.GetString("__rows"), ",")
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
		m.SetBodyWriter("text/html", func(w io.Writer) error {
			return t.Execute(w, tmplData)
		})

		err = d.DialAndSend(m)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		return
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var cfgFile string
var execname string
var debug, quiet bool

func init() {
	debug = true
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "config file (default is $HOME/.config/geneos/dv2html.yaml)")

	// how to remove the help flag help text from the help output! Sigh...
	rootCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	rootCmd.PersistentFlags().MarkHidden("help")

	execname = filepath.Base(os.Args[0])
	cordial.LogInit(execname)
}

func initConfig() {
	if quiet {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	} else if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
		viper.AutomaticEnv() // read in environment variables that match
		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err == nil {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	} else {
		cf, err := config.LoadConfig(execname,
			config.SetConfigFile(cfgFile),
			config.Global(),
			config.MergeSettings(),
		)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		replacer := strings.NewReplacer(".", "_")
		cf.SetEnvKeyReplacer(replacer)
		cf.AutomaticEnv()
	}
}
