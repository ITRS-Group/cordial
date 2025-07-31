package geneos

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func (h *Host) SetWindowsReleaseEnv(osinfo map[string]string) (err error) {
	v := windows.RtlGetVersion()

	osinfo["name"] = "windows"
	osinfo["pretty_name"] = "windows"
	osinfo["version"] = fmt.Sprintf("%d.%d", v.MajorVersion, v.MinorVersion)
	osinfo["version_id"] = fmt.Sprintf("Windows version: %d.%d (build %d)\n", v.MajorVersion, v.MinorVersion, v.BuildNumber)
	osinfo["build_id"] = fmt.Sprint(v.BuildNumber)

	return
}
