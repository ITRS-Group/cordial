package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/itrs-group/cordial/pkg/geneos/api"
	"github.com/itrs-group/cordial/pkg/logger"
)

var log = logger.Logger

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
		log.Error("supplied sample interval too short, minimum 1 second", slog.Duration("interval", interval))
	}

	// connect to netprobe
	// url := fmt.Sprintf("https://%s:%v/xmlrpc", hostname, port)
	u := &url.URL{Scheme: "https", Host: fmt.Sprintf("%s:%d", hostname, port), Path: "/xmlrpc"}

	p, err := api.NewXMLRPCClient(u.String(), api.InsecureSkipVerify())
	if err != nil {
		log.Error("error creating XMLRPC client", slog.Any("error", err))
		os.Exit(1)
	}

	s := api.NewSampler(p, "cpu", entityname, samplername)

	defer s.Close()
	s.SetInterval(interval)
	if err = s.Start(); err != nil {
		log.Error("error starting sampler", slog.Any("error", err))
		os.Exit(1)
	}
	select {} // and wait
}
