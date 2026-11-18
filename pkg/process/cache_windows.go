package process

import (
	"errors"
	"fmt"
	"reflect"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/itrs-group/cordial/pkg/host"
)

// procCacheTTL is how long the cached process entries are used for
var procCacheTTL = 5 * time.Second

// scanProcesses builds a map of PID to process status information from a
// snapshot of the processes, and if the type T has PID, PPID and Children
// fields, it also builds child lists based on linked PID and PPID fields.
func scanProcesses[T any](h host.Host) (c map[int]T, ok bool) {
	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		err = fmt.Errorf("failed to create toolhelp32 snapshot: %w", err)
		return c, false
	}
	defer windows.CloseHandle(handle)

	c = make(map[int]T)

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err = windows.Process32First(handle, &pe); err != nil {
		err = fmt.Errorf("failed to get first process: %w", err)
		return c, false
	}

	for {
		if pe.ProcessID != 0 {
			pid := pe.ProcessID
			var pstatus T
			if pstatus, err = ProcessStatus[T](h, int(pid), false, false); err != nil {
				if !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
					err = fmt.Errorf("failed to get process status for pid %d: %w", pid, err)
				}
			} else {
				c[int(pid)] = pstatus
			}
		}
		if err = windows.Process32Next(handle, &pe); err != nil {
			break
		}
	}

	st := reflect.TypeFor[T]()
	if st.Kind() == reflect.Pointer {
		st = st.Elem()
	}
	if _, hasPID := st.FieldByName("PID"); hasPID {
		if _, hasPPID := st.FieldByName("PPID"); hasPPID {
			if _, hasChildren := st.FieldByName("Children"); hasChildren {
				for _, child := range c {
					childValue := reflect.ValueOf(child)
					if childValue.Kind() == reflect.Pointer {
						childValue = childValue.Elem()
					}
					pidField := childValue.FieldByName("PID")
					ppidField := childValue.FieldByName("PPID")
					if !pidField.IsValid() || !ppidField.IsValid() || pidField.Kind() < reflect.Int || pidField.Kind() > reflect.Int64 || ppidField.Kind() < reflect.Int || ppidField.Kind() > reflect.Int64 {
						continue
					}
					parent, ok := c[int(ppidField.Int())]
					if !ok {
						continue
					}
					parentValue := reflect.ValueOf(parent)
					if parentValue.Kind() == reflect.Pointer {
						parentValue = parentValue.Elem()
					}
					children := parentValue.FieldByName("Children")
					if children.IsValid() && children.CanSet() && children.Kind() == reflect.Slice && children.Type().Elem().Kind() == reflect.Int {
						children.Set(reflect.Append(children, reflect.ValueOf(int(pidField.Int()))))
						c[int(ppidField.Int())] = parent
					}
				}
			}
		}
	}

	return c, true
}
