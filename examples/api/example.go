package main

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/itrs-group/cordial/pkg/logger"
	"github.com/itrs-group/cordial/pkg/plugins"
	"github.com/itrs-group/cordial/pkg/streams"

	"github.com/itrs-group/cordial/examples/api/cpu"
	"github.com/itrs-group/cordial/examples/api/generic"
	"github.com/itrs-group/cordial/examples/api/memory"
	"github.com/itrs-group/cordial/examples/api/process"
)

func init() {
	// logger.EnableDebugLog()
}

var (
	log      = logger.Log
	logDebug = logger.Debug
	logError = logger.Error
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
		log.Fatalf("supplied sample interval (%v) too short, minimum 1 second", interval)
	}

	// connect to netprobe
	url := fmt.Sprintf("https://%s:%v/xmlrpc", hostname, port)
	p, err := plugins.Sampler(url, entityname, samplername)
	if err != nil {
		log.Fatal(err)
	}
	p.AllowUnverifiedCertificates()

	m, err := memory.New(p, "memory", "SYSTEM")
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()
	m.SetInterval(interval)
	if err = m.Start(&wg); err != nil {
		log.Fatal(err)
	}

	c, err := cpu.New(p, "cpu", "SYSTEM")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	c.SetInterval(interval)
	if err = c.Start(&wg); err != nil {
		log.Fatal(err)
	}

	pr, err := process.New(p, "processes", "SYSTEM")
	defer pr.Close()
	pr.SetInterval(10 * time.Second)
	pr.Start(&wg)

	g, err := generic.New(p, "example", "SYSTEM")
	defer g.Close()
	g.SetInterval(interval)
	g.Start(&wg)

	powerwall, err := NewPW(p, "PW Meters", "Powerwall")
	defer powerwall.Close()
	powerwall.SetInterval(interval)
	powerwall.Start(&wg)

	streamssampler := "streams"
	sp, err := streams.Sampler(fmt.Sprintf("https://%s:%v/xmlrpc", hostname, port), entityname, streamssampler)
	if err != nil {
		log.Fatal(err)
	}
	sp.AllowUnverifiedCertificates()

	wg.Add(1)
	sp.SetStreamName("teststream")
	go func() {
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for {
			<-tick.C
			// err := sp.WriteMessage("teststream", time.Now().String()+" this is a test")

			_, err := sp.WriteString(time.Now().String() + " this is a test")
			if err != nil {
				log.Fatal(err)
				break
			}
		}
		wg.Done()
	}()

	wg.Wait()
}
