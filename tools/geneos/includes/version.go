package includes

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/itrs-group/cordial"
)

var includeSourcePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
var includeVersionPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(\.(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

func embeddedIncludeVersion() string {
	return "itrs-" + cordial.VERSION
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
		if name[:index] != "config" && includeSourcePattern.MatchString(name[:index]) && includeVersionPattern.MatchString(name[index+1:]) {
			return name[:index], name[index+1:], nil
		}
		end = index
	}
	return "", "", fmt.Errorf("invalid include version %q: expected <source>-v<major>.<minor>.<patch> (or a legacy semantic version)", name)
}

func includePattern(version string) (string, error) {
	source, _, err := parseIncludeVersion(version)
	if err != nil {
		return "", err
	}
	return "files/" + source + ".*.xml", nil
}
