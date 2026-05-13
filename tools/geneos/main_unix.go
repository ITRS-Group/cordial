//go:build !windows

package main

import (
	"syscall"

	"github.com/rs/zerolog/log"
)

func startUp() {
	// set effective IDs to real IDs, to drop any elevated privileges,
	// for example when running as root on Linux/Unix
	if err := syscall.Seteuid(syscall.Getuid()); err != nil {
		log.Debug().Err(err).Msg("failed to set effective user ID")
	}
	if err := syscall.Setegid(syscall.Getgid()); err != nil {
		log.Debug().Err(err).Msg("failed to set effective group ID")
	}
}
