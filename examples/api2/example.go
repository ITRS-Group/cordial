package main

import (
	"flag"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/examples/api2/cpu"
	"github.com/itrs-group/cordial/pkg/geneos/api"
)

func main() {
	var wg sync.WaitGroup
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

	s, err := api.NewSampler(p, entityname, samplername)

	c, err := cpu.New(s, "cpu", "SYSTEM")
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	defer c.Close()
	c.Interval = interval
	if err = c.Start(&wg); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	// powerwall, err := NewPW(p, "PW Meters", "Powerwall")
	// defer powerwall.Close()
	// powerwall.SetInterval(interval)
	// powerwall.Start(&wg)

	wg.Add(1)
	go func() {
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for {
			<-tick.C
			fmt.Fprintln(sp, time.Now().String(), "this is a test")
			if err != nil {
				log.Fatal().Err(err).Msg("")
				break
			}
		}
		wg.Done()
	}()

	wg.Wait()
}
