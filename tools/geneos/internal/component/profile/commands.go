/*
Copyright © 2026 ITRS Group

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

package profile

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	profileconfig "github.com/itrs-group/cordial/tools/geneos/internal/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
)

//go:embed profile.md
var profileDescription string

func init() {
	cmd.Cmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileExtractCmd)

	profileCmd.SetHelpFunc(profileHelp)
}

// profileCmd represents the profile component
var profileCmd = &cobra.Command{
	Use:                   "profile",
	Aliases:               []string{"profiles"},
	GroupID:               cmd.CommandGroupComponents,
	Short:                 "Profile",
	Long:                  profileDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Run:                   cmd.Cmd.HelpFunc(),
}

// profileHelp is the custom help function for the profile command. It
// checks if a profile name is provided as an argument and displays its
// description if available. Otherwise, it shows the general help text.
func profileHelp(command *cobra.Command, _ []string) {
	// if given a profile name as an argumnent, see if the profile
	// has a `description` field and display it as help text using
	// the same markdown renderer

	// pull out profile name from the command-line arguments
	args := os.Args[3:]

	if pf := cmd.Profiles; pf != nil && len(args) > 0 {
		// check if the profile exists and has a description
		// display the description as help text
		// (implementation would go here)
		name := args[0]
		components, found := config.Lookup[map[string]any](pf, pf.Join("profiles", name))
		if found {
			if desc, ok := components["_description"].(string); ok {
				command.Print(cmd.RenderHelpMarkdown(desc, command.OutOrStdout()))
			}
			return
		}
	}

	cmd.Cmd.HelpFunc()(command, nil)
}

//go:embed _docs/list.md
var profileListDescription string

var profileListCmdFormat string

var profileListCmd = &cobra.Command{
	Use:                   "list",
	Aliases:               []string{"ls"},
	Short:                 "List available profiles",
	Long:                  profileListDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	RunE: func(command *cobra.Command, args []string) error {
		if err := validateProfileListFormat(profileListCmdFormat); err != nil {
			return err
		}
		pf := cmd.Profiles
		profiles, found := config.Lookup[map[string]any](pf, "profiles")
		if !found {
			profiles = map[string]any{}
		}
		source := pf.ConfigFileUsed()
		if source == "" {
			source = "internal defaults"
		}
		return formatProfileList(os.Stdout, profiles, source, profileListCmdFormat)
	},
}

func init() {
	profileListCmd.Flags().StringVarP(&profileListCmdFormat, "format", "f", "column", "Specify the output format: column, csv, json, pretty, table or toolkit")
}

func validateProfileListFormat(format string) error {
	switch format {
	case "column", "csv", "json", "pretty", "table", "toolkit":
		return nil
	default:
		return fmt.Errorf("invalid format %q: must be one of column, csv, json, pretty, table or toolkit", format)
	}
}

func formatProfileList(w io.Writer, profiles map[string]any, source, format string) error {
	headings := []string{"Profile", "Description", "From"}
	result := make(responses.GeneralResponses, len(profiles))
	for _, name := range slices.Sorted(maps.Keys(profiles)) {
		fields, _ := profiles[name].(map[string]any)
		description := fmt.Sprint(fields["_short"])
		if fields["_short"] == nil {
			description = ""
		}
		result[name] = &responses.General{
			Value: map[string]string{
				"profile":     name,
				"description": description,
				"from":        source,
			},
			Dataview: &responses.Dataview{Table: [][]string{{name, description, source}}},
		}
	}
	if format == "pretty" {
		return result.Formatted(w, "json", headings, nil, responses.IndentJSON(true))
	}
	return result.Formatted(w, format, headings, nil)
}

//go:embed _docs/extract.md
var profileExtractDescription string

var profileExtractCmd = &cobra.Command{
	Use:                   "extract",
	Aliases:               []string{"update"},
	Short:                 "Extract the embedded profile configuration",
	Long:                  profileExtractDescription,
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	RunE: func(command *cobra.Command, _ []string) error {
		profilesPath, err := config.UserConfigPath(cordial.ExecutableName(), "profiles.yaml")
		if err != nil {
			return fmt.Errorf("resolve user profiles path: %w", err)
		}

		backup, changed, err := extractProfileConfig(profilesPath, profileconfig.EmbeddedDefaults(), time.Now())
		if err != nil {
			return err
		}
		if !changed {
			command.Printf("Embedded profile configuration already matches %s; no changes made.\n", profilesPath)
			return nil
		}
		if backup != "" {
			command.Printf("Backed up existing profile configuration to %s\n", backup)
		}
		command.Printf("Wrote embedded profile configuration to %s\n", profilesPath)
		return nil
	},
}

func extractProfileConfig(filename string, embedded []byte, now time.Time) (backup string, changed bool, err error) {
	dir := filepath.Dir(filename)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return "", false, fmt.Errorf("create user profiles directory: %w", err)
	}

	current, readErr := os.ReadFile(filename)
	if readErr == nil && bytes.Equal(current, embedded) {
		return "", false, nil
	}
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return "", false, fmt.Errorf("read user profiles file: %w", readErr)
	}

	temp, err := os.CreateTemp(dir, ".profiles-*.yaml")
	if err != nil {
		return "", false, fmt.Errorf("create temporary profiles file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err = temp.Write(embedded); err != nil {
		temp.Close()
		return "", false, fmt.Errorf("write temporary profiles file: %w", err)
	}
	if err = temp.Chmod(0o644); err != nil {
		temp.Close()
		return "", false, fmt.Errorf("set temporary profiles permissions: %w", err)
	}
	if err = temp.Close(); err != nil {
		return "", false, fmt.Errorf("close temporary profiles file: %w", err)
	}

	if readErr == nil {
		backup, err = profileBackupPath(filename, now)
		if err != nil {
			return "", false, err
		}
		if err = os.Rename(filename, backup); err != nil {
			return "", false, fmt.Errorf("back up existing profiles file: %w", err)
		}
	}

	if err = os.Rename(tempPath, filename); err != nil {
		installErr := fmt.Errorf("install embedded profiles file: %w", err)
		if backup != "" {
			if restoreErr := os.Rename(backup, filename); restoreErr != nil {
				return backup, false, errors.Join(installErr, fmt.Errorf("restore previous profiles file from %s: %w", backup, restoreErr))
			}
		}
		return "", false, installErr
	}

	return backup, true, nil
}

func profileBackupPath(filename string, now time.Time) (string, error) {
	ext := filepath.Ext(filename)
	prefix := strings.TrimSuffix(filename, ext)
	timestamp := now.Format("20060102150405")
	for suffix := 0; ; suffix++ {
		candidate := fmt.Sprintf("%s-%s%s%s", prefix, timestamp, backupSequence(suffix), ext)
		if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", fmt.Errorf("check profile backup path %s: %w", candidate, err)
		}
	}
}

func backupSequence(suffix int) string {
	if suffix == 0 {
		return ""
	}
	return fmt.Sprintf("-%d", suffix)
}
