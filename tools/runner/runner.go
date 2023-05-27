package main

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/host"
	"github.com/itrs-group/cordial/pkg/process"
)

const xauthority = "/var/run/.Xauthority"

var programs = []process.Program{
	{
		Executable: "xauth",
		Args:       []string{"-f", xauthority, "add", ":0", ".", hexrand128()},
		Foreground: true,
	},
	{
		Executable: "chmod",
		Args:       []string{"+r", xauthority},
		Foreground: true,
	},
	{
		Executable: "easy-novnc",
		Username:   "novnc",
		ErrLog:     ".nonvc.log",
		Args:       "--no-url-password -a :6901 --novnc-params resize=remote --cert test.pem --key test.key",
	},

	{
		Executable: "/usr/bin/Xvnc",
		Username:   "xserver",
		ErrLog:     ".Xvnc.log",
		Args:       ":0 -SecurityTypes None -x509cert test.pem -x509key test.key -desktop TEST -auth " + xauthority,
	},
	{
		Executable: "xfce4-session",
		Username:   "geneos",
		ErrLog:     ".xfce4-session.log",
		Args:       "--disable-tcp",
		Env:        []string{"DISPLAY=:0", "XAUTHORITY=" + xauthority, "NO_AT_BRIDGE=1"},
		Restart:    true,
	},
}

func main() {
	cordial.LogInit("runner")
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	_, err := process.Batch(host.Localhost, programs)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	select {} // wait forever
}

func hexrand128() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
