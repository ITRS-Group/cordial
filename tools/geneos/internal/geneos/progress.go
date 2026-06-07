package geneos

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

func getbar(console *os.File, name string, size int64) (bar *progressbar.ProgressBar, isterm bool) {
	isterm = term.IsTerminal(int(console.Fd()))

	out := io.Discard
	if isterm {
		out = console
	}

	if size < 1 {
		bar = progressbar.DefaultSilent(size)
		return
	}

	bar = progressbar.NewOptions64(
		size,
		progressbar.OptionSetDescription(name),
		progressbar.OptionSetWriter(out),
		progressbar.OptionShowBytes(true),
		// progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionUseIECUnits(true),
	)
	return
}
