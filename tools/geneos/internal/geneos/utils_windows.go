/*
Copyright © 2022 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package geneos

import (
	"io/fs"

	"github.com/pkg/sftp"
)

func (h *Host) GetFileOwner(info fs.FileInfo) (uid, gid int) {
	uid = -1
	gid = -1
	if h.GetString("name") != LOCALHOST {
		if sys := info.Sys(); sys != nil {
			uid = int(info.Sys().(*sftp.FileStat).UID)
			gid = int(info.Sys().(*sftp.FileStat).GID)
		}
	}
	return
}

func GetUsername(uid int) (username string) {
	return
}

func GetGroupname(gid int) (groupname string) {
	return
}
