package includes

import (
	"embed"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"log/slog"

	"github.com/spf13/cobra"
	"golang.org/x/net/html/charset"

	geneosxml "github.com/itrs-group/cordial/pkg/geneos"
	"github.com/itrs-group/cordial/pkg/logger"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
)

var log = logger.Logger

//go:embed files/*
var IncludesFS embed.FS

//go:embed README.md
var includeCmdDescription string

func init() {
	cmd.Cmd.AddCommand(includesCmd)
}

var includesCmd = &cobra.Command{
	Use:          "includes",
	Aliases:      []string{"include"},
	GroupID:      cmd.CommandGroupSubsystems,
	Short:        "Manage includes",
	Long:         includeCmdDescription,
	SilenceUsage: true,
	Annotations: map[string]string{
		cmd.CmdGlobal:       "false",
		cmd.CmdRequireHome:  "true",
		cmd.CmdAuditCommand: "never",
		cmd.CmdAuditActions: "never",
	},
	DisableFlagParsing:    true,
	DisableFlagsInUseLine: true,
}

func getComment(src fs.File) (comment string, err error) {
	d := xml.NewDecoder(src)
	d.CharsetReader = charset.NewReaderLabel

	var g geneosxml.Gateway
	if err = d.Decode(&g); err != nil {
		line, column := d.InputPos()
		err = fmt.Errorf("%w: line: %d char: %d", err, line, column)
		log.Error("Failed to decode XML", slog.String("error", err.Error()))
		return "", err
	}

	// rewind
	_, err = src.(io.Seeker).Seek(0, io.SeekStart)
	if err != nil {
		log.Error("Failed to rewind source", slog.String("error", err.Error()))
		return "", err
	}

	comment = g.Comment
	log.Debug("Processing include file", slog.String("comment", comment))

	return comment, nil
}
