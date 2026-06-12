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

package geneos

import (
	"fmt"
	"log/slog"
	"path"
	"regexp"
	"strings"

	"github.com/itrs-group/cordial/pkg/host"
)

// CleanRelativePath returns a cleaned version of relative path p. If
// the cleaning results in an absolute path or one that tries to ascend
// the tree then return an error
//
// TODO: look at os.Root instead
func CleanRelativePath(p string) (clean string, err error) {
	clean = path.Clean(p)
	if path.IsAbs(clean) || strings.HasPrefix(clean, "../") {
		log.Debug("path must be relative and descending only", slog.String("path", clean))
		return "", host.ErrInvalidArgs
	}

	return
}

// split an package archive name into type and version
//
// geneos-gateway-7.1.0-20240828.194610-12-linux-x64.tar.gz
var archiveRE = regexp.MustCompile(`^geneos-(?<component>[\w-]+)-(?<version>[\d\-\.]+)(-(?<platform>\w+))?[\.-](?<os>linux|windows).*?\.(?<suffix>[\w\.]+)$`)

// FilenameToComponentVersion transforms an archive filename and returns
// the component and version or an error if the file format is not
// recognised
func FilenameToComponentVersion(oct *Component, filename string) (ct *Component, version, platform, suffix string, err error) {
	var re *regexp.Regexp
	var parts []string

	for nct := range oct.OrList() {
		re = archiveRE
		if nct.DownloadNameRegexp != nil {
			re = nct.DownloadNameRegexp
		}
		parts = re.FindStringSubmatch(filename)
		if len(parts) > 0 {
			break
		}
	}

	if len(parts) == 0 {
		err = fmt.Errorf("%q: filename not in expected format: %w", filename, ErrInvalidArgs)
		return
	}
	versionIndex := re.SubexpIndex("version")
	componentIndex := re.SubexpIndex("component")
	// osIndex := re.SubexpIndex("os") // unused
	suffixIndex := re.SubexpIndex("suffix")

	if versionIndex == -1 || componentIndex == -1 || suffixIndex == -1 || len(parts) < versionIndex+1 {
		err = fmt.Errorf("%q: filename not in expected format: %w", filename, ErrInvalidArgs)
		return
	}
	version = parts[versionIndex]
	// replace '-' prefix of recognised platform suffixes with '+' so work with semver as metadata
	for _, m := range platformSuffixList {
		version = strings.ReplaceAll(version, "-"+m, "+"+m)
	}

	ct = ParseComponent(parts[componentIndex])
	platformIndex := re.SubexpIndex("platform")
	if platformIndex != -1 && len(parts) > platformIndex {
		platform = parts[platformIndex]
	}

	suffix = parts[suffixIndex]
	return
}

var anchoredVersRE = regexp.MustCompile(`^(\d+(\.\d+){0,2})$`)

func matchVersion(v string) bool {
	return anchoredVersRE.MatchString(v)
}

func OverrideToComponentVersion(override string) (ct *Component, version string, err error) {
	t, v, found := strings.Cut(override, ":")
	if !found {
		err = fmt.Errorf("type/version override must be in the form TYPE:VERSION (%w)", ErrInvalidArgs)
		return
	}
	ct = ParseComponent(t)
	if ct == nil {
		err = fmt.Errorf("invalid component type %q (%w)", t, ErrInvalidArgs)
		return
	}
	version = v
	if !matchVersion(version) {
		err = fmt.Errorf("invalid version %q (%w)", v, ErrInvalidArgs)
		return
	}
	return
}

func getPlatformId(value string) (id string) {
	s := strings.Split(value, ":")
	if len(s) > 1 {
		id = s[1]
	}
	return
}
