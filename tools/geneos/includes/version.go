package includes

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	includecontent "github.com/itrs-group/cordial/pkg/includes"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

var includeSourcePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
var includeVersionPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(\.(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)
var holidayYearPattern = regexp.MustCompile(`^v[0-9]{4}\.`)

func embeddedIncludeVersion() string {
	return "itrs-" + includecontent.VERSION
}

func validCalendarVersion(version string) bool {
	date, err := time.Parse("v2006.01.02", version)
	return err == nil && date.Year() > 0 && date.Format("v2006.01.02") == version
}

func validIncludeVersion(source, version string) bool {
	if source == "holidays" && holidayYearPattern.MatchString(version) {
		return validCalendarVersion(version)
	}
	return includeVersionPattern.MatchString(version)
}

func latestIncludeVersion(h *geneos.Host, prefix string) (string, error) {
	entries, err := h.ReadDir(h.PathTo("gateway", "includes"))
	if err != nil {
		return "", fmt.Errorf("find installed includes for source %q: %w", prefix, err)
	}
	var latest, latestVersion string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		source, version, err := parseIncludeVersion(entry.Name())
		if err != nil || source != prefix {
			continue
		}
		if latest == "" || geneos.CompareVersion(version, latestVersion) > 0 {
			latest, latestVersion = entry.Name(), version
		}
	}
	if latest == "" {
		return "", fmt.Errorf("no installed include versions for source %q; install an archive first", prefix)
	}
	return latest, nil
}

func commandIncludePrefix(command *cobra.Command, allowAll bool) (string, error) {
	prefix, err := command.Flags().GetString("prefix")
	if err != nil {
		return "", err
	}
	if allowAll && prefix == "*" {
		return "", nil
	}
	if prefix == "config" || !includeSourcePattern.MatchString(prefix) {
		return "", fmt.Errorf("invalid include prefix %q: expected a source name beginning with a letter or digit and containing only letters, digits, dots, underscores or hyphens; config is reserved", prefix)
	}
	return prefix, nil
}

func includeVersionForPrefix(version, prefix string, explicit bool) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		if prefix == "itrs" {
			return embeddedIncludeVersion(), nil
		}
		return "", nil
	}
	if validIncludeVersion(prefix, "v"+strings.TrimPrefix(version, "v")) {
		if prefix == "itrs" && !explicit {
			return version, nil
		}
		return prefix + "-v" + strings.TrimPrefix(version, "v"), nil
	}
	if err := checkIncludePrefix(version, prefix, explicit); err != nil {
		return "", err
	}
	return version, nil
}

func checkIncludePrefix(version, prefix string, explicit bool) error {
	source, _, err := parseIncludeVersion(version)
	if err != nil {
		return err
	}
	if explicit && source != prefix {
		return fmt.Errorf("include source %q does not match --prefix %q", source, prefix)
	}
	return nil
}

// Legacy version directories belong to the ITRS source. Try source
// separators from the right without splitting semantic prerelease suffixes.
func parseIncludeVersion(name string) (source, version string, err error) {
	if includeVersionPattern.MatchString("v" + strings.TrimPrefix(name, "v")) {
		return "itrs", name, nil
	}
	for end := len(name); end > 0; {
		index := strings.LastIndex(name[:end], "-v")
		if index < 0 {
			break
		}
		if name[:index] != "config" && includeSourcePattern.MatchString(name[:index]) && validIncludeVersion(name[:index], name[index+1:]) {
			return name[:index], name[index+1:], nil
		}
		end = index
	}
	return "", "", fmt.Errorf("invalid include version %q: expected <source>-v<major>.<minor>.<patch>, holidays-vYYYY.MM.DD, or a legacy semantic version", name)
}

func includePattern(version string) (string, error) {
	source, _, err := parseIncludeVersion(version)
	if err != nil {
		return "", err
	}
	return "files/" + source + ".*.xml", nil
}
