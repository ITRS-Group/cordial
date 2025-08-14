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

package process

import (
	"errors"
	"os/exec"
	"syscall"
	"time"
	"unsafe"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
)

func prepareCmd(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
		}
	} else {
		cmd.SysProcAttr.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS
	}
}

// user and group are sids
func setCredentials(cmd *exec.Cmd, user, group any) {
	// not implemented
}

func setCredentialsFromUsername(cmd *exec.Cmd, username string) (err error) {
	// not implemented
	return
}

func getLocalProcCache() (c procCache, ok bool, resetcache bool) {
	procCacheMutex.Lock()
	defer procCacheMutex.Unlock()

	if !resetcache {
		if c, ok = procCacheMap[nil]; ok {
			if time.Since(c.LastUpdate) < procCacheTTL {
				return
			}
		}
	}

	c.Entries = map[int]ProcessInfo{}

	h, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		log.Error().Err(err).Msg("failed to create toolhelp32 snapshot")
		return c, false
	}
	defer windows.CloseHandle(h)

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err = windows.Process32First(h, &pe); err != nil {
		log.Error().Err(err).Msg("failed to get first process")
		return c, false
	}

	for {
		// var pbi windows.PROCESS_BASIC_INFORMATION
		// var retLen uint32
		// var argc int32

		pid := uint32(pe.ProcessID)
		exe := windows.UTF16ToString(pe.ExeFile[:])

		if err = windows.Process32Next(h, &pe); err != nil {
			// done
			break
		}
		if pe.ProcessID == 0 {
			continue
		}

		cmdLine, _, err := getProcessInfo(pid)
		if err != nil {
			continue
		}

		c.Entries[int(pid)] = ProcessInfo{
			PID:     int(pid),
			Exe:     exe,
			Cmdline: cmdLine,
		}
	}

	c.LastUpdate = time.Now()
	procCacheMap[nil] = c

	return c, true
}

func getProcessInfo(pid uint32) (cmdLine []string, creationTime time.Time, err error) {
	var pbi windows.PROCESS_BASIC_INFORMATION
	var retLen uint32
	var argc int32

	ph, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, pid)
	if err != nil {
		return
	}
	defer windows.CloseHandle(ph)

	err = windows.NtQueryInformationProcess(ph, windows.ProcessBasicInformation, unsafe.Pointer(&pbi), uint32(unsafe.Sizeof(pbi)), &retLen)
	if err != nil {
		log.Debug().Err(err).Msgf("failed to query process information for pid %d, wanted %d got %d", pid, retLen, uint32(unsafe.Sizeof(pbi)))
		return
	}

	pebSize := unsafe.Sizeof(windows.PEB{})
	pebBuf := make([]byte, pebSize)

	err = windows.ReadProcessMemory(ph, uintptr(unsafe.Pointer(pbi.PebBaseAddress)), &pebBuf[0], pebSize, nil)
	if err != nil {
		log.Debug().Err(err).Msgf("failed to read process memory for pid %d", pid)
		return
	}
	peb := (*windows.PEB)(unsafe.Pointer(&pebBuf[0]))

	ppp := peb.ProcessParameters
	if ppp == nil {
		log.Debug().Msgf("no process parameters for pid %d", pid)
		err = errors.New("no process parameters")
		return
	}

	pp := &windows.RTL_USER_PROCESS_PARAMETERS{}
	ppSize := unsafe.Sizeof(*pp)
	ppBuf := make([]byte, ppSize)
	err = windows.ReadProcessMemory(ph, uintptr(unsafe.Pointer(ppp)), &ppBuf[0], ppSize, nil)
	if err != nil {
		log.Debug().Err(err).Msgf("failed to read process parameters for pid %d	", pid)
		return
	}
	pp = (*windows.RTL_USER_PROCESS_PARAMETERS)(unsafe.Pointer(&ppBuf[0]))

	if pp.CommandLine.Length == 0 {
		log.Debug().Msgf("no command line for pid %d", pid)
		err = errors.New("no command line")
		return
	}

	cmdBuf := make([]byte, pp.CommandLine.Length)
	err = windows.ReadProcessMemory(ph, uintptr(unsafe.Pointer(pp.CommandLine.Buffer)), &cmdBuf[0], uintptr(pp.CommandLine.Length), nil)
	if err != nil {
		log.Debug().Err(err).Msgf("failed to read command line for pid %d", pid)
		return
	}

	argvw, err := windows.CommandLineToArgv((*uint16)(unsafe.Pointer(&cmdBuf[0])), &argc)
	if err != nil {
		log.Debug().Err(err).Msgf("failed to convert command line to argv for pid	%d", pid)
		return
	}
	defer windows.LocalFree((windows.Handle)(unsafe.Pointer(argvw)))

	args := make([]string, argc)
	for i := range args {
		cmdLine = append(cmdLine, windows.UTF16ToString(argvw[i][:]))
	}

	var createTime, exitTime, kernelTime, userTime windows.Filetime
	err = windows.GetProcessTimes(ph, &createTime, &exitTime, &kernelTime, &userTime)
	if err != nil {
		log.Debug().Err(err).Msgf("failed to get process times for pid %d", pid)
		return
	}
	// convert to time.Time
	creationTime = time.Unix(0, int64(createTime.Nanoseconds()))

	return
}

func getProcessUser(ph windows.Handle) (uid, gid int, err error) {
	var token windows.Token
	err = windows.OpenProcessToken(ph, windows.TOKEN_QUERY, &token)
	if err != nil {
		return 0, 0, err
	}
	defer token.Close()

	var size uint32
	err = windows.GetTokenInformation(token, windows.TokenUser, nil, 0, &size)
	if err != nil && err != windows.ERROR_INSUFFICIENT_BUFFER {
		return 0, 0, err
	}

	buf := make([]byte, size)
	err = windows.GetTokenInformation(token, windows.TokenUser, &buf[0], size, &size)
	if err != nil {
		return 0, 0, err
	}

	userInfo := (*windows.Tokenuser)(unsafe.Pointer(&buf[0]))

	sid := userInfo.User.Sid
	subAuthCount := int(sid.SubAuthorityCount())
	if subAuthCount < 2 {
		return 0, 0, nil
	}
	uid = int(sid.SubAuthority(uint32(subAuthCount - 1)))
	gid = int(sid.SubAuthority(uint32(subAuthCount - 2)))

	return uid, gid, nil
}
