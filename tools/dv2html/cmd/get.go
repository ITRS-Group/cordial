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
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/itrs-group/cordial/pkg/commands"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/xpath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var getCmdGatewayURL, getCmdGatewayHost, getCmdGatewayUsername, getCmdGatewayPassword, getCmdHTML, getCmdCSS string
var getCmdHeadlinesFilter, getCmdColumnFilter, getCmdRowFilter, getCmdOutput string

var getCmdGatewayPort, getCmdMaxDataviews int

var getCmdUseTLS, getCmdAllowInsecure bool

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().StringVar(&getCmdGatewayURL, "url", "", "URL to access Gateway")
	getCmd.Flags().StringVarP(&getCmdGatewayHost, "host", "H", "localhost", "hostname or IP of gateway")
	getCmd.Flags().IntVarP(&getCmdGatewayPort, "port", "P", 7039, "port of gateway")
	getCmd.Flags().BoolVarP(&getCmdUseTLS, "tls", "t", false, "enable TLS")
	getCmd.Flags().BoolVarP(&getCmdAllowInsecure, "insecure", "k", false, "allow unverified TLS server certs")

	getCmd.Flags().IntVarP(&getCmdMaxDataviews, "max", "m", 0, "Maximum matching Dataviews to pass to template. Default 0, meaning unlimited.")

	getCmd.Flags().StringVarP(&getCmdGatewayUsername, "username", "u", "", "Gateway Username")
	getCmd.Flags().StringVarP(&getCmdGatewayPassword, "password", "p", "", "Gateway Password")

	getCmd.Flags().StringVarP(&getCmdOutput, "output", "o", "", "output file. default stdout")

	getCmd.Flags().StringVar(&getCmdHeadlinesFilter, "headlines", "", "comma separated, ordered list of headlines to include. empty means all")
	getCmd.Flags().StringVar(&getCmdColumnFilter, "columns", "", "comma separated, ordered list of columns to include. empty means all")
	getCmd.Flags().StringVar(&getCmdRowFilter, "rows", "", "comma separated, ordered list of rows to include. empty means all")

	getCmd.Flags().StringVar(&getCmdHTML, "html", "", "path to Go HTML template")
	getCmd.Flags().StringVar(&getCmdCSS, "css", "", "path to css file, file or URL")

	getCmd.MarkFlagsMutuallyExclusive("url", "host")
}

// default html and css templates
//
//

// //go:embed dv2html.css
// var cssData template.CSS

//go:embed dv2html.gotmpl
var htmlDefaultTemplate string

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get [flags] XPATH...",
	Short: "Get a Dataview",
	Long: strings.ReplaceAll(`
Get a Dataview from a Gateway and convert to HTML using a template and CSS.
`, "|", "`"),
	Run: func(cmd *cobra.Command, args []string) {
		var u *url.URL
		var err error
		cf := config.GetConfig()

		if getCmdGatewayURL != "" {
			u, err = url.Parse(getCmdGatewayURL)
			if err != nil {
				log.Fatal().Err(err).Msg("")
			}
		} else {
			cf.SetDefault("host", getCmdGatewayHost)
			cf.SetDefault("port", getCmdGatewayPort)
			u = &url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("%s:%d", cf.GetString("host"), cf.GetInt("port")),
			}
			cf.SetDefault("use-tls", getCmdUseTLS)
			if cf.GetBool("use-tls") {
				u.Scheme = "https"
			}
		}

		cf.SetDefault("username", getCmdGatewayUsername)
		cf.SetDefault("password", getCmdGatewayPassword)

		gw, err := commands.DialGateway(u, commands.SetBasicAuth(cf.GetString("username"), cf.GetString("password")), commands.AllowInsecureCertificates(cf.GetBool("allow-insecure")))
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		cf.SetDefault("css-url", getCmdCSS)
		cf.SetDefault("html-template", getCmdHTML)

		htmlTemplate := htmlDefaultTemplate
		if h := cf.GetString("html-template"); h != "" {
			htmlTemplate = h
		}

		t, err := template.New("dataview").Parse(htmlTemplate)
		if err != nil {
			log.Fatal().Err(err).Msg("")
		}

		// cf.SetDefault("css-data", cssData)

		tmplData := dv2htmlData{
			CSSURL:    cf.GetString("css-url"),
			CSSDATA:   template.CSS(cf.GetString("css-data")),
			Dataviews: []*commands.Dataview{},
			Env:       make(map[string]string, len(os.Environ())),
		}

		for _, e := range os.Environ() {
			n := strings.SplitN(e, "=", 2)
			tmplData.Env[n[0]] = n[1]
		}

		// get the variable XPath from environment if not on command line
		if len(args) == 0 && cf.IsSet("_variablepath") {
			args = []string{cf.GetString("_variablepath")}
		}

		for _, d := range args {
			dv, err := xpath.Parse(d)
			dv = dv.ResolveTo(&xpath.Dataview{})

			if err != nil {
				log.Error().Err(err).Msg("")
				continue
			}

			paths, err := gw.Match(dv, 0)
			if err != nil {
				log.Error().Err(err).Msg("")
				continue
			}

			if len(paths) == 0 {
				log.Fatal().Msg("no matching dataviews found")
			}

			if getCmdMaxDataviews > 0 && len(paths) > getCmdMaxDataviews {
				paths = paths[:getCmdMaxDataviews]
			}

			for _, x := range paths {
				data, err := gw.Snapshot(x, commands.Scope{Value: true, Severity: true})
				if err != nil {
					log.Error().Err(err).Msg("")
					continue
				}

				tmplData.Dataviews = append(tmplData.Dataviews, data)

				// filter here
				if getCmdHeadlinesFilter != "" {
					nh := map[string]commands.DataItem{}
					headlines := strings.Split(getCmdHeadlinesFilter, ",")
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

				if getCmdColumnFilter != "" {
					nc := []string{"rowname"}
					cols := strings.Split(getCmdColumnFilter, ",")
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

				if getCmdRowFilter != "" {
					nr := map[string]map[string]commands.DataItem{}
					rows := strings.Split(getCmdRowFilter, ",")
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
		}

		out := os.Stdout
		if getCmdOutput != "" {
			out, err = os.Create(getCmdOutput)
			if err != nil {
				log.Fatal().Err(err).Msg("")
			}
			defer out.Close()
		}

		if err = t.Execute(out, tmplData); err != nil {
			log.Fatal().Err(err).Msg("")
		}
	},
}
