package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/geneos/plugins"
	"github.com/itrs-group/cordial/pkg/geneos/streams"

	"github.com/itrs-group/cordial/examples/api/cpu"
	"github.com/itrs-group/cordial/examples/api/generic"
	"github.com/itrs-group/cordial/examples/api/memory"
	"github.com/itrs-group/cordial/examples/api/process"
)

var log = cordial.Logger

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
		log.Error("supplied sample interval too short, minimum 1 second", slog.Duration("interval", interval))
		os.Exit(1)
	}

	// connect to netprobe
	// url := fmt.Sprintf("https://%s:%v/xmlrpc", hostname, port)
	u := &url.URL{Scheme: "https", Host: fmt.Sprintf("%s:%d", hostname, port), Path: "/xmlrpc"}
	p, err := plugins.Open(u, entityname, samplername)
	if err != nil {
		log.Error("error opening plugin connection", slog.Any("error", err))
		os.Exit(1)
	}
	p.InsecureSkipVerify()

	m, err := memory.New(p, "memory", "SYSTEM")
	if err != nil {
		log.Error("error creating memory sampler", slog.Any("error", err))
		os.Exit(1)
	}
	defer m.Close()
	m.SetInterval(interval)
	if err = m.Start(&wg); err != nil {
		log.Error("error starting memory sampler", slog.Any("error", err))
		os.Exit(1)
	}

	c, err := cpu.New(p, "cpu", "SYSTEM")
	if err != nil {
		log.Error("error creating CPU sampler", slog.Any("error", err))
		os.Exit(1)
	}
	defer c.Close()
	c.SetInterval(interval)
	if err = c.Start(&wg); err != nil {
		log.Error("error starting CPU sampler", slog.Any("error", err))
		os.Exit(1)
	}

	pr, err := process.New(p, "processes", "SYSTEM")
	defer pr.Close()
	pr.SetInterval(10 * time.Second)
	pr.Start(&wg)

	g, err := generic.New(p, "example", "SYSTEM")
	defer g.Close()
	g.SetInterval(interval)
	g.Start(&wg)

	// powerwall, err := NewPW(p, "PW Meters", "Powerwall")
	// defer powerwall.Close()
	// powerwall.SetInterval(interval)
	// powerwall.Start(&wg)

	streamssampler := "streams"
	sp, err := streams.Open(u, entityname, streamssampler, "teststream")
	if err != nil {
		log.Error("error opening streams connection", slog.Any("error", err))
		os.Exit(1)
	}
	sp.InsecureSkipVerify()

	wg.Go(func() {
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for {
			<-tick.C
			fmt.Fprintln(sp, time.Now().String(), "this is a test")
			if err != nil {
				log.Error("error writing to stream", slog.Any("error", err))
				break
			}
		}
	})

	wg.Wait()
}
