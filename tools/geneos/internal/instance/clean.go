package instance

import (
	"os"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/spf13/viper"
)

func Clean(c geneos.Instance, options ...geneos.GeneosOptions) (err error) {
	var stopped bool

	opts := geneos.EvalOptions(options...)

	cleanlist := viper.GetString(c.Type().CleanList)
	purgelist := viper.GetString(c.Type().PurgeList)

	if !opts.Restart() {
		if cleanlist != "" {
			if err = RemovePaths(c, cleanlist); err == nil {
				logDebug.Println(c, "cleaned")
			}
		}
		return
	}

	if _, err = GetPID(c); err == os.ErrProcessDone {
		stopped = false
	} else if err = Stop(c, false); err != nil {
		return
	} else {
		stopped = true
	}

	if cleanlist != "" {
		if err = RemovePaths(c, cleanlist); err != nil {
			return
		}
	}
	if purgelist != "" {
		if err = RemovePaths(c, purgelist); err != nil {
			return
		}
	}
	logDebug.Println(c, "fully cleaned")
	if stopped {
		err = Start(c)
	}
	return
}
