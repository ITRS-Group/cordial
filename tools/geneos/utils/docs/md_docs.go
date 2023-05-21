// Copyright 2013-2023 The Cobra Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This is a customised copy of the md_docs.go source from the cobra
// project but with change to meet local cordial requirements.

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func printOptions(buf *bytes.Buffer, cmd *cobra.Command, name string) error {
	flags := cmd.NonInheritedFlags()
	flags.SetOutput(buf)
	if flags.HasAvailableFlags() {
		buf.WriteString("### Options\n\n```text\n")
		flags.PrintDefaults()
		buf.WriteString("```\n\n")
	}

	if !cmd.SilenceUsage {
		parentFlags := cmd.InheritedFlags()
		parentFlags.SetOutput(buf)
		if parentFlags.HasAvailableFlags() {
			buf.WriteString("### Options inherited from parent commands\n\n```text\n")
			parentFlags.PrintDefaults()
			buf.WriteString("```\n\n")
		}
	}
	return nil
}

// GenMarkdown creates markdown output.
func GenMarkdown(cmd *cobra.Command, w io.Writer) error {
	return GenMarkdownCustom(cmd, w, func(s string) string { return s })
}

// GenMarkdownCustom creates custom markdown output.
func GenMarkdownCustom(cmd *cobra.Command, w io.Writer, linkHandler func(string) string) error {
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()

	buf := new(bytes.Buffer)
	name := cmd.CommandPath()

	buf.WriteString("# `" + name + "`\n\n")
	buf.WriteString(cmd.Short + "\n\n")
	if cmd.Runnable() {
		buf.WriteString(fmt.Sprintf("```text\n%s\n```\n", cmd.UseLine()))
	}
	if hasSeeAlso(cmd) {
		children := cmd.Commands()
		sort.Sort(byName(children))

		groups := cmd.Groups()

		if len(groups) > 0 {
			for i, group := range groups {
				cmdHeader := false
				for _, child := range children {
					if child.GroupID != group.ID {
						continue
					}
					// if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
					// 	continue
					// }
					if !cmdHeader {
						buf.WriteString("## " + group.Title + "\n\n")
						cmdHeader = true
					}

					cname := name + " " + child.Name()
					link := cname + ".md"
					link = strings.ReplaceAll(link, " ", "_")
					buf.WriteString(fmt.Sprintf("* [`%s`](%s)\t - %s\n", cname, linkHandler(link), child.Short))
				}
				if i != len(groups)-1 {
					buf.WriteString("\n")
				}
			}
		} else {
			hadChildren := true
			for _, child := range children {
				if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
					continue
				}
				if !child.HasAvailableSubCommands() && hadChildren {
					buf.WriteString("\n## Commands\n\n")
					hadChildren = false
				}
				cname := name + " " + child.Name()
				link := cname + ".md"
				link = strings.ReplaceAll(link, " ", "_")
				buf.WriteString(fmt.Sprintf("* [`%s`](%s)\t - %s\n", cname, linkHandler(link), child.Short))
			}
		}
		buf.WriteString("\n")
	}

	if len(cmd.Long) > 0 {
		buf.WriteString(cmd.Long + "\n")
	}

	if err := printOptions(buf, cmd, name); err != nil {
		return err
	}

	if len(cmd.Example) > 0 {
		buf.WriteString("## Examples\n\n")
		buf.WriteString(fmt.Sprintf("```bash%s\n```\n\n", cmd.Example))
	}

	if hasSeeAlso(cmd) {
		buf.WriteString("## SEE ALSO\n\n")
		if cmd.HasParent() {
			parent := cmd.Parent()
			pname := parent.CommandPath()
			link := pname + ".md"
			link = strings.ReplaceAll(link, " ", "_")
			buf.WriteString(fmt.Sprintf("* [%s](%s)\t - %s\n", pname, linkHandler(link), parent.Short))
			cmd.VisitParents(func(c *cobra.Command) {
				if c.DisableAutoGenTag {
					cmd.DisableAutoGenTag = c.DisableAutoGenTag
				}
			})
		}
	}
	_, err := buf.WriteTo(w)
	return err
}

// GenMarkdownTree will generate a markdown page for this command and all
// descendants in the directory given. The header may be nil.
// This function may not work correctly if your command names have `-` in them.
// If you have `cmd` with two subcmds, `sub` and `sub-third`,
// and `sub` has a subcommand called `third`, it is undefined which
// help output will be in the file `cmd-sub-third.1`.
func GenMarkdownTree(cmd *cobra.Command, dir string) error {
	identity := func(s string) string { return s }
	emptyStr := func(s string) string { return "" }
	return GenMarkdownTreeCustom(cmd, dir, emptyStr, identity)
}

// GenMarkdownTreeCustom is the the same as GenMarkdownTree, but
// with custom filePrepender and linkHandler.
func GenMarkdownTreeCustom(cmd *cobra.Command, dir string, filePrepender, linkHandler func(string) string) error {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := GenMarkdownTreeCustom(c, dir, filePrepender, linkHandler); err != nil {
			return err
		}
	}

	basename := strings.ReplaceAll(cmd.CommandPath(), " ", "_") + ".md"
	filename := filepath.Join(dir, basename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.WriteString(f, filePrepender(filename)); err != nil {
		return err
	}
	if err := GenMarkdownCustom(cmd, f, linkHandler); err != nil {
		return err
	}
	return nil
}

// Test to see if we have a reason to print See Also information in docs
// Basically this is a test for a parent command or a subcommand which is
// both not deprecated and not the autogenerated help command.
func hasSeeAlso(cmd *cobra.Command) bool {
	if cmd.HasParent() {
		return true
	}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		return true
	}
	return false
}

// Temporary workaround for yaml lib generating incorrect yaml with long strings
// that do not contain \n.
func forceMultiLine(s string) string {
	if len(s) > 60 && !strings.Contains(s, "\n") {
		s = s + "\n"
	}
	return s
}

type byName []*cobra.Command

func (s byName) Len() int           { return len(s) }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byName) Less(i, j int) bool { return s[i].Name() < s[j].Name() }

type subsystemsThenNames []*cobra.Command

func (s subsystemsThenNames) Len() int      { return len(s) }
func (s subsystemsThenNames) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s subsystemsThenNames) Less(i, j int) bool {
	if s[i].HasAvailableSubCommands() && !s[j].HasAvailableSubCommands() {
		return true
	}
	if !s[i].HasAvailableSubCommands() && s[j].HasAvailableSubCommands() {
		return false
	}
	return s[i].Name() < s[j].Name()
}

// type groupsThenNames []*cobra.Command

// func (s groupsThenNames) Len() int      { return len(s) }
// func (s groupsThenNames) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
// func (s groupsThenNames) Less(i, j int) bool {
// 	if s[i].HasAvailableSubCommands() && !s[j].HasAvailableSubCommands() {
// 		return true
// 	}
// 	if !s[i].HasAvailableSubCommands() && s[j].HasAvailableSubCommands() {
// 		return false
// 	}
// 	return s[i].Name() < s[j].Name()
// }
