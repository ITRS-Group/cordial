/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"os"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial/pkg/geneos/api"
)

var docker client.APIClient

func init() {

}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dockerlogs",
	Short: "Feed docker logs to Geneos API Streams plugin",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		docker, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

		ctx := context.Background()
		allContainers, err := docker.ContainerList(ctx, types.ContainerListOptions{})
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

		for _, i := range allContainers {
			r, err := docker.ContainerLogs(ctx, i.ID, types.ContainerLogsOptions{
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
			go func(i types.Container) {
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
