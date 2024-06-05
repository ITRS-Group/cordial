package main

import (
	"flag"
	"fmt"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/geneos/api"
)

func main() {
	// var wg sync.WaitGroup
	var interval time.Duration
	var (
		hostname                string
		port                    uint
		entityname, samplername string
	)

	flag.StringVar(&hostname, "h", "localhost", "Netprobe hostname")
	flag.UintVar(&port, "p", 7036, "Netprobe port number")
	flag.DurationVar(&interval, "t", 1*time.Second, "Global DoSample Interval in seconds (min 1)")
	flag.StringVar(&entityname, "e", "", "Default entity to connect")
	flag.StringVar(&samplername, "s", "", "Default sampler to connect")
	flag.Parse()

	if interval < 1*time.Second {
		log.Fatal().Msgf("supplied sample interval (%v) too short, minimum 1 second", interval)
	}

	// connect to netprobe
	// url := fmt.Sprintf("https://%s:%v/xmlrpc", hostname, port)
	u := &url.URL{Scheme: "https", Host: fmt.Sprintf("%s:%d", hostname, port), Path: "/xmlrpc"}

	p, err := api.NewXMLRPCClient(u.String(), api.InsecureSkipVerify())
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	s := api.NewSampler(p, "cpu", entityname, samplername)

	defer s.Close()
	s.SetInterval(interval)
	if err = s.Start(); err != nil {
		log.Fatal().Err(err).Msg("")
	}
	select {} // and wait
}
