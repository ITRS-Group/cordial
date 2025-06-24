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

// FileOwner is only available on Linux localhost
type FileOwner struct {
	Uid int
	Gid int
}

func (h *Host) GetFileOwner(info fs.FileInfo) (s FileOwner) {
	s.Uid = -1
	s.Gid = -1
	if h.GetString("name") != LOCALHOST {
		if sys := info.Sys(); sys != nil {
			s.Uid = int(info.Sys().(*sftp.FileStat).UID)
			s.Gid = int(info.Sys().(*sftp.FileStat).GID)
		}
	}
	return
}

func GetUsername(s FileOwner) (username string) {
	return
}

func GetGroupname(s FileOwner) (groupname string) {
	return
}
