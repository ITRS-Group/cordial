package utils

import (
	"math"
	"os"
	"os/user"
	"strconv"

	"github.com/itrs-group/cordial/pkg/config"
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
