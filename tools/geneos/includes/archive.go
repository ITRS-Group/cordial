package includes

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// limits on the size of include archives, to protect against corrupted
// or malicious files. The real include files are small XML files.
const (
	maxIncludeFileSize    = 32 << 20  // largest single file in an archive
	maxIncludeArchiveSize = 256 << 20 // total size of all files in an archive
	maxArchiveTrailer     = 1 << 20   // data allowed after the last tar entry
)

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
	if n, err := io.Copy(io.Discard, io.LimitReader(compressed, maxArchiveTrailer+1)); err != nil {
		return "", fmt.Errorf("invalid include archive: %w", err)
	} else if n > maxArchiveTrailer {
		return "", errors.New("invalid include archive: unexpected data after the last entry")
	}
	return version, nil
}

func readIncludeArchive(archive *tar.Reader, staging string) (string, error) {
	var version string
	var archiveRoot string
	var archiveRootVersion string
	var totalSize int64
	includeCount := 0
	seen := make(map[string]bool)
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
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
		if len(parts) == 1 && strings.HasPrefix(parts[0], "geneos-includes-") && header.Typeflag == tar.TypeDir {
			if archiveRoot != "" {
				return "", fmt.Errorf("include archive contains multiple root directories")
			}
			archiveRoot = parts[0]
			archiveRootVersion = strings.TrimPrefix(archiveRoot, "geneos-includes-")
			if !includeVersionPattern.MatchString(archiveRootVersion) {
				return "", fmt.Errorf("invalid include archive root %q: expected geneos-includes-<version>", archiveRoot)
			}
			continue
		}
		if archiveRoot == "" || parts[0] != archiveRoot || len(parts) < 2 {
			return "", fmt.Errorf("include archive entry %q must be inside a geneos-includes-<version> directory", header.Name)
		}
		parts = parts[1:]
		_, contentsVersion, versionErr := parseIncludeVersion(parts[0])
		isVersion := versionErr == nil
		if isVersion {
			if contentsVersion != archiveRootVersion {
				return "", fmt.Errorf("include archive root version %q does not match contents version %q", archiveRootVersion, parts[0])
			}
			if version != "" && version != parts[0] {
				return "", fmt.Errorf("include archive contains multiple versions")
			}
			version = parts[0]
		} else if parts[0] != "config" && parts[0] != "scripts" && parts[0] != "docs" {
			return "", fmt.Errorf("invalid include archive directory %q: expected <source>-v<version>, config, scripts, or docs", parts[0])
		}
		if header.Typeflag == tar.TypeDir && len(parts) == 1 {
			continue
		}
		if header.Typeflag != tar.TypeReg || len(parts) != 2 {
			return "", fmt.Errorf("unsupported include archive entry %q", header.Name)
		}
		pattern := path.Base(ConfigPattern)
		if isVersion {
			includeGlob, err := includePattern(parts[0])
			if err != nil {
				return "", err
			}
			pattern = path.Base(includeGlob)
		} else if parts[0] == "scripts" {
			pattern = "install-includes.sh"
		} else if parts[0] == "docs" {
			pattern = ""
			if path.Ext(parts[1]) != ".md" && path.Ext(parts[1]) != ".pdf" {
				return "", fmt.Errorf("unexpected include archive documentation file %q", header.Name)
			}
		}
		if pattern != "" {
			matched, _ := path.Match(pattern, parts[1])
			if !matched {
				return "", fmt.Errorf("unexpected include archive file %q", header.Name)
			}
		}
		if seen[parts[1]] {
			return "", fmt.Errorf("duplicate include archive file %q", header.Name)
		}
		seen[parts[1]] = true
		if parts[0] == "docs" {
			continue
		}
		if header.Size > maxIncludeFileSize {
			return "", fmt.Errorf("include archive file %q is too large (%d bytes, limit %d)", header.Name, header.Size, maxIncludeFileSize)
		}
		if totalSize += header.Size; totalSize > maxIncludeArchiveSize {
			return "", fmt.Errorf("include archive contents are too large (limit %d bytes)", maxIncludeArchiveSize)
		}
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
