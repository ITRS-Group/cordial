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
	"net/url"
	"os"
	"path"
	"slices"
	"sort"
	"strings"

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
var execname = cordial.ExecutableName()
var debug, quiet bool
var inlineCSS bool

var entityArg, samplerArg, typeArg, dataviewArg string
var toArg, ccArg, bccArg string

func init() {
	cobra.OnInitialize(initConfig)

	// execname = cordial.ExecutableName()
	cordial.LogInit(execname)

	DV2EMAILCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable extra debug output")
	DV2EMAILCmd.PersistentFlags().BoolVarP(&inlineCSS, "inline-css", "i", true, "inline CSS for better mail client support")
	DV2EMAILCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "config file (default is $HOME/.config/geneos/dv2email.yaml)")

	// how to remove the help flag help text from the help output! Sigh...
	DV2EMAILCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	DV2EMAILCmd.PersistentFlags().MarkHidden("help")

	DV2EMAILCmd.Flags().StringVarP(&entityArg, "entity", "E", "", "entity name, ignored if _VARIBLEPATH set in environment")
	DV2EMAILCmd.Flags().StringVarP(&samplerArg, "sampler", "S", "", "sampler name, ignored if _VARIBLEPATH set in environment")
	DV2EMAILCmd.Flags().StringVarP(&typeArg, "type", "T", "", "type name, ignored if _VARIBLEPATH set in environment")
	DV2EMAILCmd.Flags().StringVarP(&dataviewArg, "dataview", "D", "", "dataview name, ignored if _VARIBLEPATH set in environment")

	DV2EMAILCmd.Flags().StringVarP(&toArg, "to", "t", "", "To as comma-separated emails")
	DV2EMAILCmd.Flags().StringVarP(&ccArg, "cc", "c", "", "Cc as comma-separated emails")
	DV2EMAILCmd.Flags().StringVarP(&bccArg, "bcc", "b", "", "Bcc as comma-separated emails")

	DV2EMAILCmd.Flags().SortFlags = false
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

	config.DefaultKeyDelimiter("::")

	opts := []config.FileOptions{
		config.SetAppName("geneos"),
		config.SetConfigFile(cfgFile),
		config.MergeSettings(),
		config.SetFileExtension("yaml"),
		config.WithDefaults(defaults, "yaml"),
	}

	cf, err = config.Load(execname, opts...)
	if err != nil {
		log.Fatal().Err(err).Msgf("loading from file %s", config.Path(execname, opts...))
	}
}

type dv2emailData struct {
	Dataviews []*commands.Dataview
	Env       map[string]string
}

//go:embed dv2email.defaults.yaml
var defaults []byte

//go:embed _docs/root.md
var DV2EMAILCmdDescription string

// DV2EMAILCmd represents the base command when called without any subcommands
var DV2EMAILCmd = &cobra.Command{
	Use:          "dv2email",
	Short:        "Email a Dataview following Geneos Action/Effect conventions",
	Long:         DV2EMAILCmdDescription,
	SilenceUsage: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Version:               cordial.VERSION,
	DisableAutoGenTag:     true,
	DisableSuggestions:    true,
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, _ []string) (err error) {
		// cf.SetDefault("gateway::allow-insecure", true)
		u := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", cf.GetString("gateway::host", config.Default("localhost")), cf.GetInt("gateway::port", config.Default(7038))),
		}

		cf.SetDefault("use-tls", true)
		if cf.GetBool("use-tls") {
			u.Scheme = "https"
		}

		password := &config.Plaintext{}

		username := cf.GetString("gateway::username")
		gateway := cf.GetString("gateway::name")

		if username != "" {
			password = cf.GetPassword("gateway::password")
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
			commands.AllowInsecureCertificates(cf.GetBool("gateway::allow-insecure")),
		)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		data := dv2emailData{
			Dataviews: []*commands.Dataview{},
			Env:       make(map[string]string, len(os.Environ())),
		}

		// import all environment variables into both the template data
		// and also the config structure (config.WithEnvs doesn't work
		// for empty prefixes)
		for _, e := range os.Environ() {
			n := strings.SplitN(e, "=", 2)
			data.Env[n[0]] = n[1]
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
		if err != nil {
			return
		}
		dv = dv.ResolveTo(&xpath.Dataview{})

		dataviews, err := gw.Match(dv, 0)
		if err != nil {
			return
		}

		if len(dataviews) == 0 {
			return errors.New("no matching dataviews found")
		}

		em := config.New()
		// set default from yaml file, can be overridden from Geneos as
		// environment variables

		// creds can come from `geneos` credentials for the mail server
		// domain

		epassword := &config.Plaintext{}

		eusername := cf.GetString("email::username")
		smtpserver := cf.GetString("email::smtp", config.Default("localhost"))
		smtptls := cf.GetString("email::use-tls", config.Default("default"))

		if eusername != "" {
			epassword = cf.GetPassword("email::password")
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
		em.SetDefault("_smtp_port", cf.GetInt("email::port", config.Default(25)))
		em.SetDefault("_from", cf.GetString("email::from"))
		em.SetDefault("_to", cf.GetString("email::to"))
		em.SetDefault("_cc", cf.GetString("email::cc"))
		em.SetDefault("_bcc", cf.GetString("email::bcc"))
		em.SetDefault("_subject", cf.GetString("email::subject", config.Default("Geneos Alert")))

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

		for _, d := range dataviews {
			dataview, err := getDataview(d, gw, em)
			if err != nil {
				log.Error().Err(err).Msg("")
				continue
			}

			data.Dataviews = append(data.Dataviews, dataview)
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

		// attachments here

		if slices.Contains(cf.GetStringSlice("email::contents"), "text+html") {
			var textString, htmlString string
			textString, err = createTextTemplate(cf, data, cf.GetString("attachments::text::template"))
			m.SetBody("text/plain", textString)

			htmlString, err = createHTML(cf, data, cf.GetString("attachments::html::template"), inlineCSS)
			m.AddAlternative("text/html", htmlString)
		}

		if slices.Contains(cf.GetStringSlice("email::contents"), "xlsx") {
			buf, err := createXLSX(cf, data)
			if err != nil {
				log.Error().Err(err).Msg("xlsx")
			}

			m.AttachReader(cf.GetString("attachments::xlsx::filename", config.LookupTable(dv.LookupValues())), buf)
		}

		if slices.Contains(cf.GetStringSlice("email::contents"), "images") {
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
