//go:build !windows

package main

import (
	"log/slog"
	"syscall"
)

func startUp() {
	// set effective IDs to real IDs, to drop any elevated privileges,
	// for example when running as root on Linux/Unix
	if err := syscall.Seteuid(syscall.Getuid()); err != nil {
		log.Debug("failed to set effective user ID", slog.Any("error", err))
	}
	if err := syscall.Setegid(syscall.Getgid()); err != nil {
		log.Debug("failed to set effective group ID", slog.Any("error", err))
	}
}
