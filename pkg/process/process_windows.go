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
	"reflect"
	"syscall"
	"time"
	"unsafe"

	"github.com/itrs-group/cordial/pkg/host"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
)

// ProcessStatus for Windows fills in the pstats struct based on tags
// and field names.
func ProcessStatus[T any](h host.Host, pid int, getStat, getStatus bool) (pstats T, err error) {
	if h == nil {
		err = errors.New("host is nil")
		return
	}
	if !h.IsLocal() {
		err = errors.New("host is not local")
		return
	}

	if reflect.TypeOf(pstats).Kind() != reflect.Pointer || reflect.TypeOf(pstats).Elem().Kind() != reflect.Struct {
		err = errors.New("pstats must be a pointer to a struct")
		return
	}

	st := reflect.TypeOf(pstats).Elem()

	if reflect.ValueOf(pstats).IsNil() {
		pstats = reflect.New(st).Interface().(T)
	}

	sv := reflect.ValueOf(pstats).Elem()

	for i := 0; i < st.NumField(); i++ {
		ft := st.Field(i)
		fv := sv.Field(i)

		if lookup, ok := ft.Tag.Lookup("cache"); ok && lookup == "lazy" {
			// skip filling this field for now, it will be filled on demand when requested
			continue

			// if not lazy or no cache tag, then fill it now drop
			// through to fill it below
		}

		switch ft.Name {
		case "OpenFiles":
			ProcessStatusOpenFiles(h, uint32(pid), fv)

		case "OpenSockets":
			ProcessStatusOpenSockets(h, uint32(pid), fv)

		case "ListeningPorts":
			ProcessStatusListeningPorts(h, uint32(pid), fv)
		}
	}

	ph, err := windows.OpenProcess(
		windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ,
		false,
		uint32(pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(ph)

	var token windows.Token
	err = windows.OpenProcessToken(ph, windows.TOKEN_QUERY, &token)
	if err != nil {
		return
	}
	defer token.Close()

	var size uint32
	err = windows.GetTokenInformation(token, windows.TokenUser, nil, 0, &size)
	if err != nil && err != windows.ERROR_INSUFFICIENT_BUFFER {
		return
	}

	buf := make([]byte, size)
	err = windows.GetTokenInformation(token, windows.TokenUser, &buf[0], size, &size)
	if err != nil {
		return
	}

	userInfo := (*windows.Tokenuser)(unsafe.Pointer(&buf[0]))

	sid := userInfo.User.Sid
	subAuthCount := int(sid.SubAuthorityCount())
	if subAuthCount < 2 {
		return
	}

	// "UID" and "GID"
	uid := int(sid.SubAuthority(uint32(subAuthCount - 1)))
	gid := int(sid.SubAuthority(uint32(subAuthCount - 2)))

	svUID := sv.FieldByName("UID")
	if svUID.IsValid() && svUID.CanSet() {
		k := svUID.Kind()
		if k == reflect.Int || k == reflect.Int64 || k == reflect.Int32 || k == reflect.Int16 || k == reflect.Int8 {
			svUID.SetInt(int64(uid))
		}
	}

	svGID := sv.FieldByName("GID")
	if svGID.IsValid() && svGID.CanSet() {
		k := svGID.Kind()
		if k == reflect.Int || k == reflect.Int64 || k == reflect.Int32 || k == reflect.Int16 || k == reflect.Int8 {
			svGID.SetInt(int64(gid))
		}
	}

	var pbi windows.PROCESS_BASIC_INFORMATION
	var retLen uint32
	var argc int32

	err = windows.NtQueryInformationProcess(ph, windows.ProcessBasicInformation, unsafe.Pointer(&pbi), uint32(unsafe.Sizeof(pbi)), &retLen)
	if err != nil {
		log.Debug().Err(err).Msgf("failed to query process information for pid %d, wanted %d got %d", pid, retLen, uint32(unsafe.Sizeof(pbi)))
		return
	}

	// "PID" and "PPID"
	svPID := sv.FieldByName("PID")
	if svPID.IsValid() && svPID.CanSet() {
		k := svPID.Kind()
		if k == reflect.Int || k == reflect.Int64 || k == reflect.Int32 || k == reflect.Int16 || k == reflect.Int8 {
			svPID.SetInt(int64(pid))
		}
	}

	svPPID := sv.FieldByName("PPID")
	if svPPID.IsValid() && svPPID.CanSet() {
		k := svPPID.Kind()
		if k == reflect.Int || k == reflect.Int64 || k == reflect.Int32 || k == reflect.Int16 || k == reflect.Int8 {
			svPPID.SetInt(int64(pbi.InheritedFromUniqueProcessId))
		}
	}

	pebSize := unsafe.Sizeof(windows.PEB{})
	pebBuf := make([]byte, pebSize)
	var numRead uintptr

	err = windows.ReadProcessMemory(ph, uintptr(unsafe.Pointer(pbi.PebBaseAddress)), &pebBuf[0], pebSize, &numRead)
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

	err = windows.ReadProcessMemory(ph, uintptr(unsafe.Pointer(ppp)), &ppBuf[0], ppSize, &numRead)
	if err != nil {
		log.Debug().Err(err).Msgf("failed to read process parameters for pid %d	", pid)
		return
	}
	pp = (*windows.RTL_USER_PROCESS_PARAMETERS)(unsafe.Pointer(&ppBuf[0]))

	// Exe and Cmdline

	svExe := sv.FieldByName("Exe")
	if svExe.IsValid() && svExe.CanSet() && svExe.Kind() == reflect.String {
		exePath, err := readNTUnicodeString(ph, pp.ImagePathName)
		if err != nil {
			log.Debug().Err(err).Msgf("failed to read executable path for pid %d", pid)
		} else {
			svExe.SetString(exePath)
		}
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

	var cmdLine []string
	for i := range argc {
		cmdLine = append(cmdLine, windows.UTF16ToString(argvw[i][:]))
	}

	svCmdline := sv.FieldByName("Cmdline")
	if svCmdline.IsValid() && svCmdline.CanSet() && svCmdline.Kind() == reflect.Slice && svCmdline.Type().Elem().Kind() == reflect.String {
		svCmdline.Set(reflect.ValueOf(cmdLine))
	}

	var createTime, exitTime, kernelTime, userTime windows.Filetime
	err = windows.GetProcessTimes(ph, &createTime, &exitTime, &kernelTime, &userTime)
	if err != nil {
		log.Debug().Err(err).Msgf("failed to get process times for pid %d", pid)
		return
	}

	// "StartTime"

	// convert to time.Time
	startTime := time.Unix(0, int64(createTime.Nanoseconds()))
	svStartTime := sv.FieldByName("StartTime")
	if svStartTime.IsValid() && svStartTime.CanSet() {
		k := svStartTime.Kind()
		if k == reflect.Struct && svStartTime.Type() == reflect.TypeOf(time.Time{}) {
			svStartTime.Set(reflect.ValueOf(startTime))
		} else if k == reflect.Pointer && svStartTime.Type().Elem() == reflect.TypeOf(time.Time{}) {
			svStartTime.Set(reflect.ValueOf(&startTime))
		}
	}

	return
}

func readNTUnicodeString(ph windows.Handle, us windows.NTUnicodeString) (s string, err error) {
	if us.Length == 0 {
		return "", nil
	}

	buf := make([]uint16, us.MaximumLength/2)
	var numRead uintptr
	err = windows.ReadProcessMemory(ph, uintptr(unsafe.Pointer(us.Buffer)), (*byte)(unsafe.Pointer(&buf[0])), uintptr(us.MaximumLength), &numRead)
	if err != nil {
		return "", err
	}

	return windows.UTF16ToString(buf[:numRead/2]), nil
}

func ProcessStatusOpenFiles(h host.Host, pid uint32, fv reflect.Value) {
	return
}

func ProcessStatusOpenSockets(h host.Host, pid uint32, fv reflect.Value) {
	return
}

func ProcessStatusListeningPorts(h host.Host, pid uint32, fv reflect.Value) {
	return
}

func GetUsername(uid int) (username string) {
	return
}

func GetGroupname(gid int) (groupname string) {
	return
}

func prepareCmd(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
		}
	} else {
		cmd.SysProcAttr.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS
	}
}

// func getWindowsProcCache(resetcache bool) (c procCache, ok bool) {
// 	procCacheMutex.Lock()
// 	defer procCacheMutex.Unlock()

// 	if !resetcache {
// 		if c, ok = procCacheMap[nil].(procCache); ok {
// 			if time.Since(c.LastUpdate) < procCacheTTL {
// 				return
// 			}
// 		}
// 	}

// 	c.Entries = map[int]*ProcessInfo{}

// 	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
// 	if err != nil {
// 		log.Error().Err(err).Msg("failed to create toolhelp32 snapshot")
// 		return c, false
// 	}
// 	defer windows.CloseHandle(handle)

// 	var pe windows.ProcessEntry32
// 	pe.Size = uint32(unsafe.Sizeof(pe))
// 	if err = windows.Process32First(handle, &pe); err != nil {
// 		log.Error().Err(err).Msg("failed to get first process")
// 		return c, false
// 	}

// 	for {
// 		if err = windows.Process32Next(handle, &pe); err != nil {
// 			// done
// 			break
// 		}
// 		if pe.ProcessID == 0 {
// 			continue
// 		}

// 		pid := pe.ProcessID

// 		cmdLine, StartTime, err := getProcessInfo(pid)
// 		if err != nil {
// 			continue
// 		}

// 		uid, gid, err := getProcessUser(windows.Handle(pid))
// 		if err != nil {
// 			log.Debug().Err(err).Msgf("failed to get process user for pid %d", pid)
// 			continue
// 		}

// 		c.Entries[int(pid)] = &ProcessInfo{
// 			PID:       int(pid),
// 			PPID:      int(pe.ParentProcessID),
// 			Exe:       windows.UTF16ToString(pe.ExeFile[:]),
// 			Cmdline:   cmdLine,
// 			StartTime: StartTime,
// 			UID:       uid,
// 			GID:       gid,
// 			Username:  GetUsername(uid),
// 			Groupname: GetGroupname(gid),
// 			Children:  []int{},
// 			GIDs:      []string{},
// 		}
// 	}

// 	// build child lists
// 	for _, p := range c.Entries {
// 		if parent, ok := c.Entries[p.PPID]; ok {
// 			parent.Children = append(parent.Children, p.PID)
// 			c.Entries[p.PPID] = parent
// 		}
// 	}

// 	c.LastUpdate = time.Now()
// 	procCacheMap[nil] = c

// 	return c, true
// }

// getProcessIngo fills in the pstats struct for a given pid.
func getProcessInfo[T any](pid uint32, pstats T) (err error) {

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
