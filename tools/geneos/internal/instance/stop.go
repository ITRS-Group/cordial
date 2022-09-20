package instance

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func Stop(c geneos.Instance, force bool) (err error) {
	if !force {
		err = Signal(c, syscall.SIGTERM)
		if err == os.ErrProcessDone {
			return nil
		}

		if errors.Is(err, syscall.EPERM) {
			return nil
		}

		for i := 0; i < 10; i++ {
			time.Sleep(250 * time.Millisecond)
			err = Signal(c, syscall.SIGTERM)
			if err == os.ErrProcessDone {
				break
			}
		}

		if _, err = GetPID(c); err == os.ErrProcessDone {
			fmt.Printf("%s stopped\n", c)
			return nil
		}
	}

	if err = Signal(c, syscall.SIGKILL); err == os.ErrProcessDone {
		return nil
	}

	time.Sleep(250 * time.Millisecond)
	_, err = GetPID(c)
	if err == os.ErrProcessDone {
		fmt.Printf("%s killed\n", c)
		return nil
	}
	return
}
