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
	"fmt"
	"os/exec"
	"os/user"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/itrs-group/cordial/pkg/host"
	gprocess "github.com/shirou/gopsutil/v4/process"
	"golang.org/x/sys/windows"
)

// ProcessStatus for Windows fills in the pstats struct based on tags
// and field names.
func ProcessStatus[T any](h host.Host, pid int, getStat, getStatus bool) (pstats T, err error) {
	if h == nil {
		err = errors.New("host is nil")
		return
	}
	if !h.IsLocalhost() {
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
			ProcessStatusOpenFiles(h, pid, fv)

		case "OpenSockets":
			ProcessStatusOpenSockets(h, pid, fv)

		case "ListeningPorts":
			ProcessStatusListeningPorts(h, pid, fv)
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
	uid, gid := -1, -1

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
	if account, domain, _, lookupErr := sid.LookupAccount(""); lookupErr == nil {
		username := account
		if domain != "" {
			username = domain + "\\" + account
		}
		setProcessString(sv, "Username", username)
	}
	if primaryGroup, groupErr := token.GetTokenPrimaryGroup(); groupErr == nil {
		if group, domain, _, lookupErr := primaryGroup.PrimaryGroup.LookupAccount(""); lookupErr == nil {
			groupname := group
			if domain != "" {
				groupname = domain + "\\" + group
			}
			setProcessString(sv, "Groupname", groupname)
		}
	}

	var pbi windows.PROCESS_BASIC_INFORMATION
	var retLen uint32
	var argc int32

	err = windows.NtQueryInformationProcess(ph, windows.ProcessBasicInformation, unsafe.Pointer(&pbi), uint32(unsafe.Sizeof(pbi)), &retLen)
	if err != nil {
		err = fmt.Errorf("failed to query process information for pid %d, wanted %d got %d", pid, retLen, uint32(unsafe.Sizeof(pbi)))
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
		err = fmt.Errorf("failed to read process memory for pid %d: %w", pid, err)
		return
	}
	peb := (*windows.PEB)(unsafe.Pointer(&pebBuf[0]))

	ppp := peb.ProcessParameters
	if ppp == nil {
		err = fmt.Errorf("no process parameters for pid %d", pid)
		return
	}

	pp := &windows.RTL_USER_PROCESS_PARAMETERS{}
	ppSize := unsafe.Sizeof(*pp)
	ppBuf := make([]byte, ppSize)

	err = windows.ReadProcessMemory(ph, uintptr(unsafe.Pointer(ppp)), &ppBuf[0], ppSize, &numRead)
	if err != nil {
		err = fmt.Errorf("failed to read process parameters for pid %d: %w", pid, err)
		return
	}
	pp = (*windows.RTL_USER_PROCESS_PARAMETERS)(unsafe.Pointer(&ppBuf[0]))

	// Exe and Cmdline

	svExe := sv.FieldByName("Exe")
	if svExe.IsValid() && svExe.CanSet() && svExe.Kind() == reflect.String {
		exePath, err := readNTUnicodeString(ph, pp.ImagePathName)
		if err != nil {
			err = fmt.Errorf("failed to read executable path for pid %d: %w", pid, err)
		} else {
			svExe.SetString(exePath)
		}
	}

	var cmdLine []string
	if pp.CommandLine.Length > 0 && pp.CommandLine.Buffer != nil {
		cmdBuf := make([]byte, pp.CommandLine.Length)
		err = windows.ReadProcessMemory(ph, uintptr(unsafe.Pointer(pp.CommandLine.Buffer)), &cmdBuf[0], uintptr(pp.CommandLine.Length), nil)
		if err != nil {
			err = fmt.Errorf("failed to read command line for pid %d: %w", pid, err)
			return
		}

		argvw, conversionErr := windows.CommandLineToArgv((*uint16)(unsafe.Pointer(&cmdBuf[0])), &argc)
		if conversionErr != nil {
			err = fmt.Errorf("failed to convert command line to argv for pid %d: %w", pid, conversionErr)
			return
		}
		defer windows.LocalFree((windows.Handle)(unsafe.Pointer(argvw)))
		for i := range argc {
			cmdLine = append(cmdLine, windows.UTF16ToString(argvw[i][:]))
		}
	}

	svCmdline := sv.FieldByName("Cmdline")
	if svCmdline.IsValid() && svCmdline.CanSet() && svCmdline.Kind() == reflect.Slice && svCmdline.Type().Elem().Kind() == reflect.String {
		svCmdline.Set(reflect.ValueOf(cmdLine))
	}

	var createTime, exitTime, kernelTime, userTime windows.Filetime
	err = windows.GetProcessTimes(ph, &createTime, &exitTime, &kernelTime, &userTime)
	if err != nil {
		err = fmt.Errorf("failed to get process times for pid %d: %w", pid, err)
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
	fillWindowsProcessDetails(sv, pid)

	return
}

func fillWindowsProcessDetails(sv reflect.Value, pid int) {
	proc, err := gprocess.NewProcess(int32(pid))
	if err != nil {
		return
	}

	if ppid, err := proc.Ppid(); err == nil {
		setProcessInt(sv, "PPID", int64(ppid))
	}
	if username, err := proc.Username(); err == nil {
		setProcessString(sv, "Username", username)
	}
	if cwd, err := proc.Cwd(); err == nil {
		setProcessString(sv, "Cwd", cwd)
	}
	if threads, err := proc.NumThreads(); err == nil {
		setProcessInt(sv, "Threads", int64(threads))
	}
	if times, err := proc.Times(); err == nil {
		setProcessDuration(sv, "Utime", times.User)
		setProcessDuration(sv, "Stime", times.System)
	}
	if memory, err := proc.MemoryInfo(); err == nil {
		setProcessInt(sv, "VmRSS", int64(memory.RSS))
	}
	if affinity, err := processCPUAffinity(pid); err == nil {
		values := make([]string, len(affinity))
		for i, cpu := range affinity {
			values[i] = fmt.Sprint(cpu)
		}
		setProcessString(sv, "CpusAllowedList", strings.Join(values, ","))
	}
	fillProcessMemoryCounters(sv, pid)
	if running, err := proc.IsRunning(); err == nil {
		state := "stopped"
		if running {
			state = "running"
		}
		setProcessString(sv, "State", state)
	}
}

var (
	getProcessAffinityMaskProc = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetProcessAffinityMask")
	getProcessMemoryInfoProc   = windows.NewLazySystemDLL("psapi.dll").NewProc("GetProcessMemoryInfo")
)

type processMemoryCounters struct {
	Size                    uint32
	PageFaultCount          uint32
	PeakWorkingSetSize      uintptr
	WorkingSetSize          uintptr
	QuotaPeakPagedPoolUsage uintptr
	QuotaPagedPoolUsage     uintptr
	QuotaPeakNonPagedPool   uintptr
	QuotaNonPagedPoolUsage  uintptr
	PagefileUsage           uintptr
	PeakPagefileUsage       uintptr
	PrivateUsage            uintptr
}

func processCPUAffinity(pid int) (cpus []int, err error) {
	ph, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(ph)

	var processMask, systemMask uintptr
	result, _, callErr := getProcessAffinityMaskProc.Call(
		uintptr(ph),
		uintptr(unsafe.Pointer(&processMask)),
		uintptr(unsafe.Pointer(&systemMask)),
	)
	if result == 0 {
		if callErr != syscall.Errno(0) {
			return nil, callErr
		}
		return nil, syscall.EINVAL
	}
	for cpu := 0; cpu < runtime.NumCPU() && cpu < 8*int(unsafe.Sizeof(processMask)); cpu++ {
		if processMask&(uintptr(1)<<uint(cpu)) != 0 {
			cpus = append(cpus, cpu)
		}
	}
	return cpus, nil
}

func fillProcessMemoryCounters(sv reflect.Value, pid int) {
	ph, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(ph)

	counters := processMemoryCounters{Size: uint32(unsafe.Sizeof(processMemoryCounters{}))}
	result, _, _ := getProcessMemoryInfoProc.Call(
		uintptr(ph),
		uintptr(unsafe.Pointer(&counters)),
		uintptr(counters.Size),
	)
	if result != 0 {
		setProcessInt(sv, "VmRSS", int64(counters.WorkingSetSize))
		setProcessInt(sv, "VmHWM", int64(counters.PeakWorkingSetSize))
	}
}

func setProcessString(sv reflect.Value, name, value string) {
	fv := sv.FieldByName(name)
	if fv.IsValid() && fv.CanSet() && fv.Kind() == reflect.String {
		fv.SetString(value)
	}
}

func setProcessInt(sv reflect.Value, name string, value int64) {
	fv := sv.FieldByName(name)
	if !fv.IsValid() || !fv.CanSet() {
		return
	}
	switch fv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fv.SetInt(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value >= 0 {
			fv.SetUint(uint64(value))
		}
	}
}

func setProcessDuration(sv reflect.Value, name string, seconds float64) {
	fv := sv.FieldByName(name)
	if fv.IsValid() && fv.CanSet() && fv.Type() == reflect.TypeOf(time.Duration(0)) {
		fv.SetInt(int64(seconds * float64(time.Second)))
	}
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

// ProcessStatusOpenFiles TODO: Document its behavior, inputs, and effects.
func ProcessStatusOpenFiles(h host.Host, pid int, fv reflect.Value) {
	if !fv.IsValid() || !fv.CanSet() {
		return
	}
	files := OpenFiles(h, pid)
	switch fv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fv.SetInt(int64(len(files)))
	case reflect.Slice:
		if fv.Type().Elem() == reflect.TypeOf(ProcessFDs{}) {
			fv.Set(reflect.ValueOf(files))
		} else if fv.Type().Elem().Kind() == reflect.String {
			paths := make([]string, 0, len(files))
			for _, file := range files {
				paths = append(paths, file.Path)
			}
			fv.Set(reflect.ValueOf(paths))
		}
	case reflect.String:
		paths := make([]string, 0, len(files))
		for _, file := range files {
			paths = append(paths, file.Path)
		}
		if len(paths) == 0 {
			fv.SetString("NONE")
		} else {
			fv.SetString(strings.Join(paths, ","))
		}
	}
}

// ProcessStatusOpenSockets TODO: Document its behavior, inputs, and effects.
func ProcessStatusOpenSockets(h host.Host, pid int, fv reflect.Value) {
	if !fv.IsValid() || !fv.CanSet() || !h.IsLocalhost() {
		return
	}
	proc, err := gprocess.NewProcess(int32(pid))
	if err != nil {
		return
	}
	connections, err := proc.Connections()
	if err == nil && fv.Kind() >= reflect.Int && fv.Kind() <= reflect.Int64 {
		fv.SetInt(int64(len(connections)))
	}
}

// ProcessStatusListeningPorts TODO: Document its behavior, inputs, and effects.
func ProcessStatusListeningPorts(h host.Host, pid int, fv reflect.Value) {
	if !fv.IsValid() || !fv.CanSet() || !h.IsLocalhost() {
		return
	}
	proc, err := gprocess.NewProcess(int32(pid))
	if err != nil {
		return
	}
	connections, err := proc.Connections()
	if err != nil {
		return
	}
	ports := make([]int, 0)
	seen := make(map[int]struct{})
	for _, connection := range connections {
		if !strings.EqualFold(connection.Status, "LISTEN") {
			continue
		}
		port := int(connection.Laddr.Port)
		if _, ok := seen[port]; !ok {
			seen[port] = struct{}{}
			ports = append(ports, port)
		}
	}
	switch fv.Kind() {
	case reflect.Slice:
		if fv.Type().Elem().Kind() == reflect.Int {
			fv.Set(reflect.ValueOf(ports))
		} else if fv.Type().Elem().Kind() == reflect.String {
			values := make([]string, 0, len(ports))
			for _, port := range ports {
				values = append(values, fmt.Sprint(port))
			}
			fv.Set(reflect.ValueOf(values))
		}
	case reflect.String:
		values := make([]string, 0, len(ports))
		for _, port := range ports {
			values = append(values, fmt.Sprint(port))
		}
		if len(values) == 0 {
			fv.SetString("NONE")
		} else {
			fv.SetString(strings.Join(values, ","))
		}
	}
}

// GetUsername TODO: Document its behavior, inputs, and effects.
func GetUsername(uid int) (username string) {
	if uid < 0 {
		return "unknown"
	}
	return fmt.Sprint(uid)
}

// GetGroupname TODO: Document its behavior, inputs, and effects.
func GetGroupname(gid int) (groupname string) {
	if gid < 0 {
		return "unknown"
	}
	return fmt.Sprint(gid)
}

// GetLoginName TODO: Document its behavior, inputs, and effects.
func GetLoginName(pid int) (name string, err error) {
	if pid < 0 {
		return GetCurrentUsername()
	}
	proc, err := gprocess.NewProcess(int32(pid))
	if err != nil {
		return "", err
	}
	return proc.Username()
}

// GetCurrentUsername TODO: Document its behavior, inputs, and effects.
func GetCurrentUsername() (name string, err error) {
	currentUser, err := user.Current()
	if err != nil {
		return "unknown", err
	}
	return currentUser.Username, nil
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
