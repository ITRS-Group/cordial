package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/itrs-group/cordial/pkg/config"
)

func main() {
	var keyfile config.KeyFile

	flag.Var(&keyfile, "k", "path to keyfile")
	flag.Parse()

	if keyfile == "" {
		log.Fatal("no keyfile path given")
	}

	a, err := keyfile.Read()
	if err != nil {
		log.Fatal("cannot read keyfile:", err)
	}
	password := flag.Arg(0)
	if password == "" {
		log.Fatal("no encoded password to decode")
	}
	p, err := a.DecodeString(password)
	if err != nil {
		log.Fatal("decode of password filed:", err)
	}
	fmt.Println(p)
}
