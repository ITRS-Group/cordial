/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"os"
	"sync"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/geneos/api"
)

var moby client.APIClient

func init() {

}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mobylogs",
	Short: "Feed docker logs to Geneos API Streams plugin",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		moby, err = client.New(client.FromEnv, client.FromEnv)

		ctx := context.Background()
		result, err := moby.ContainerList(ctx, client.ContainerListOptions{All: true})
		if err != nil {
			return
		}

		var wg sync.WaitGroup
		c, err := api.NewRESTClient("http://thinkpad:7136/v1")
		// , rest.HTTPClient(&http.Client{
		// 	Transport: &http.Transport{
		// 		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		// 	},
		// }))
		if err != nil {
			return
		}

		for _, i := range result.Items {
			r, err := moby.ContainerLogs(ctx, i.ID, client.ContainerLogsOptions{
				Follow:     true,
				Timestamps: true,
				ShowStderr: true,
				ShowStdout: true,
				Tail:       "50",
			})
			if err != nil {
				log.Error().Err(err).Msg("")
				continue
			}
			defer r.Close()
			log.Info().Msgf("container log for %s open", i.ID)
			wg.Add(1)
			go func(i container.Summary) {
				defer wg.Done()
				sout, err := api.OpenStream(c, "localhost", "streams", i.Names[0]+".stdout")
				if err != nil {
					log.Error().Err(err).Msg("")
					return
				}
				if !c.Healthy() {
					log.Error().Msg("stdout not connected to probe")
				}
				serr, err := api.OpenStream(c, "localhost", "streams", i.Names[0]+".stderr")
				if err != nil {
					log.Error().Err(err).Msg("")
					return
				}
				if !c.Healthy() {
					log.Error().Msg("stderr not connected to probe")
				}
				if _, err = stdcopy.StdCopy(sout, serr, r); err != nil {
					log.Error().Err(err).Msg("")
				}
			}(i)
		}
		wg.Wait()

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
