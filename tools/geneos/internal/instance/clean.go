package instance

import (
	"os"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

func Clean(c geneos.Instance, options ...geneos.GeneosOptions) (err error) {
	var stopped bool

	opts := geneos.EvalOptions(options...)

	cleanlist := config.GetString(c.Type().CleanList)
	purgelist := config.GetString(c.Type().PurgeList)

	if !opts.Restart() {
		if cleanlist != "" {
			if err = RemovePaths(c, cleanlist); err == nil {
				log.Debug().Msgf("%s cleaned", c)
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
	log.Debug().Msgf("%s fully cleaned", c)
	if stopped {
		err = Start(c)
	}
	return
}
