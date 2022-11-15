package utils

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	"github.com/itrs-group/cordial/pkg/config"
	"golang.org/x/term"
)

func GetIDs(username string) (uid, gid int, gids []int, err error) {
	uid, gid = math.MaxUint32, math.MaxUint32

	if username == "" {
		username = config.GetString("defaultuser")
	}

	u, err := user.Lookup(username)
	if err != nil {
		return -1, -1, nil, err
	}
	uid, err = strconv.Atoi(u.Uid)
	if err != nil {
		uid = -1
	}

	gid, err = strconv.Atoi(u.Gid)
	if err != nil {
		gid = -1
	}
	groups, _ := u.GroupIds()
	for _, g := range groups {
		gid, err := strconv.Atoi(g)
		if err != nil {
			gid = -1
		}
		gids = append(gids, gid)
	}
	return
}

func IsSuperuser() bool {
	if os.Geteuid() == 0 || os.Getuid() == 0 {
		return true
	}
	return false
}

// check if the current user can do "something" with the selected component
//
// just check if running as root or if a username is specified in the config
// that the current user matches.
//
// this does not however change the user to match anything, so starting a
// process still requires a seteuid type change
func CanControl(username string) bool {
	if IsSuperuser() {
		return true
	}

	if len(username) == 0 {
		// assume the caller with try to set-up the correct user
		return true
	}

	u, err := user.Lookup(username)
	if err != nil {
		// user not found, should fail
		return false
	}

	uid, _ := strconv.Atoi(u.Uid)
	if uid == os.Getuid() || uid == os.Geteuid() {
		// if uid != euid then child proc may fail because
		// of linux ld.so secure-execution discarding
		// envs like LD_LIBRARY_PATH, account for this?
		return true
	}

	uc, _ := user.Current()
	return username == uc.Username
}

func ReadPasswordPrompt(prompt ...string) []byte {
	if len(prompt) == 0 {
		fmt.Printf("Password: ")
	} else {
		fmt.Printf("%s: ", strings.Join(prompt, " "))
	}
	pw, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalln("Error getting password:", err)
	}
	fmt.Println()
	return bytes.TrimSpace(pw)
}

func ReadPasswordFile(path string) []byte {
	pw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalln("Error reading password from file:", err)
	}
	return bytes.TrimSpace(pw)
}
