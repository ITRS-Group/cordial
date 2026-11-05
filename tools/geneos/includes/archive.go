package includes

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var includeVersionPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(\.(0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

func stageIncludeArchive(source io.Reader, staging string) (string, error) {
	compressed, err := gzip.NewReader(source)
	if err != nil {
		return "", err
	}
	defer compressed.Close()
	if err := os.Mkdir(filepath.Join(staging, "files"), 0700); err != nil {
		return "", err
	}
	version, err := readIncludeArchive(tar.NewReader(compressed), staging)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(io.Discard, compressed); err != nil {
		return "", fmt.Errorf("invalid include archive: %w", err)
	}
	return version, nil
}

func readIncludeArchive(archive *tar.Reader, staging string) (string, error) {
	var version string
	includeCount := 0
	seen := make(map[string]bool)
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("invalid include archive: %w", err)
		}
		name := strings.TrimSuffix(header.Name, "/")
		if !fs.ValidPath(name) || strings.Contains(name, `\`) {
			return "", fmt.Errorf("invalid include archive path %q", header.Name)
		}
		parts := strings.Split(name, "/")
		isVersion := includeVersionPattern.MatchString(parts[0])
		if isVersion {
			if version != "" && version != parts[0] {
				return "", fmt.Errorf("include archive contains multiple versions")
			}
			version = parts[0]
		} else if parts[0] != "config" {
			return "", fmt.Errorf("invalid include archive directory %q: expected a v-prefixed semantic version or config", parts[0])
		}
		if header.Typeflag == tar.TypeDir && len(parts) == 1 {
			continue
		}
		if header.Typeflag != tar.TypeReg || len(parts) != 2 {
			return "", fmt.Errorf("unsupported include archive entry %q", header.Name)
		}
		pattern := path.Base(ConfigPattern)
		if isVersion {
			pattern = path.Base(IncludePattern)
		}
		if matched, _ := path.Match(pattern, parts[1]); !matched {
			return "", fmt.Errorf("unexpected include archive file %q", header.Name)
		}
		if seen[parts[1]] {
			return "", fmt.Errorf("duplicate include archive file %q", header.Name)
		}
		seen[parts[1]] = true
		destination, err := os.OpenFile(filepath.Join(staging, "files", parts[1]), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return "", err
		}
		_, copyErr := io.Copy(destination, archive)
		closeErr := destination.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		if isVersion {
			includeCount++
		}
	}
	if version == "" || includeCount == 0 {
		return "", fmt.Errorf("include archive contains no versioned include files")
	}
	return version, nil
}
