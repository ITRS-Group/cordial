package geneos

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

type gatewayReadOptions struct {
	insecureSkipVerify bool
	timeout            time.Duration
}

// GatewayReadOption configures ReadGateway.
type GatewayReadOption func(*gatewayReadOptions)

// WithGatewayInsecureSkipVerify disables TLS certificate verification for an
// HTTPS source. Use it only when the server certificate cannot be validated
// through the normal trust store.
func WithGatewayInsecureSkipVerify() GatewayReadOption {
	return func(options *gatewayReadOptions) {
		options.insecureSkipVerify = true
	}
}

// WithGatewayTimeout sets the total HTTP request timeout. Zero leaves the
// request without a timeout; negative values are rejected.
func WithGatewayTimeout(timeout time.Duration) GatewayReadOption {
	return func(options *gatewayReadOptions) {
		options.timeout = timeout
	}
}

// ReadGateway reads Gateway XML from a local path, a file URL, or an HTTP(S) URL.
func ReadGateway(source string, options ...GatewayReadOption) (*Gateway, error) {
	return ReadGatewayContext(context.Background(), source, options...)
}

// ReadGatewayContext reads Gateway XML from a local path, a file URL, or an
// HTTP(S) URL. The context cancels HTTP requests and is checked before local
// file operations and XML decoding.
func ReadGatewayContext(ctx context.Context, source string, options ...GatewayReadOption) (*Gateway, error) {
	if ctx == nil {
		return nil, errors.New("gateway context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if source == "" {
		return nil, errors.New("gateway source is empty")
	}
	readOptions := &gatewayReadOptions{}
	for _, option := range options {
		if option != nil {
			option(readOptions)
		}
	}

	parsed, err := url.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("parse gateway source %q: %w", source, err)
	}
	if readOptions.timeout < 0 {
		return nil, errors.New("gateway HTTP timeout cannot be negative")
	}
	if readOptions.insecureSkipVerify && parsed.Scheme != "https" {
		return nil, errors.New("insecure TLS option requires an HTTPS gateway source")
	}

	var input io.ReadCloser
	switch parsed.Scheme {
	case "http", "https":
		client := &http.Client{Transport: http.DefaultTransport, Timeout: readOptions.timeout}
		if readOptions.insecureSkipVerify {
			transport, ok := http.DefaultTransport.(*http.Transport)
			if !ok {
				return nil, errors.New("cannot configure insecure TLS with a custom default transport")
			}
			transport = transport.Clone()
			tlsConfig := transport.TLSClientConfig
			if tlsConfig == nil {
				tlsConfig = &tls.Config{}
			} else {
				tlsConfig = tlsConfig.Clone()
			}
			tlsConfig.InsecureSkipVerify = true // Explicitly requested by the caller.
			transport.TLSClientConfig = tlsConfig
			client.Transport = transport
			defer transport.CloseIdleConnections()
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return nil, fmt.Errorf("create request for gateway XML %q: %w", source, err)
		}
		response, err := client.Do(request)
		if err != nil {
			return nil, fmt.Errorf("get gateway XML from %q: %w", source, err)
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			response.Body.Close()
			return nil, fmt.Errorf("get gateway XML from %q: HTTP %s", source, response.Status)
		}
		input = response.Body
	case "file":
		if parsed.Host != "" && !strings.EqualFold(parsed.Host, "localhost") {
			return nil, fmt.Errorf("file URL host %q is not local", parsed.Host)
		}
		if parsed.Path == "" {
			return nil, errors.New("file URL has an empty path")
		}
		input, err = os.Open(filepath.FromSlash(parsed.Path))
		if err != nil {
			return nil, fmt.Errorf("open gateway XML %q: %w", source, err)
		}
	case "":
		input, err = os.Open(source)
		if err != nil {
			return nil, fmt.Errorf("open gateway XML %q: %w", source, err)
		}
	default:
		return nil, fmt.Errorf("unsupported gateway source scheme %q", parsed.Scheme)
	}
	defer input.Close()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	decoder := xml.NewDecoder(input)
	decoder.CharsetReader = charset.NewReaderLabel
	var gateway Gateway
	if err := decoder.Decode(&gateway); err != nil {
		line, column := decoder.InputPos()
		return nil, fmt.Errorf("decode gateway XML from %q at line %d, column %d: %w", source, line, column, err)
	}
	return &gateway, nil
}

// DefaultGatewayHistoryLimit is the number of numbered Gateway history files
// retained by WriteGateway when Backup is requested and
// WithGatewayHistoryLimit is omitted or called with zero.
// This matches the Gateway binary's default history retention.
const DefaultGatewayHistoryLimit = 10

type gatewayWriteOptions struct {
	backup       bool
	overwrite    bool
	historyLimit int
}

// GatewayWriteOption configures WriteGateway.
type GatewayWriteOption func(*gatewayWriteOptions)

// WithGatewayBackup enables numbered history backups when replacing a file.
func WithGatewayBackup() GatewayWriteOption {
	return func(options *gatewayWriteOptions) {
		options.backup = true
	}
}

// WithGatewayOverwrite permits replacing an existing file without a backup.
func WithGatewayOverwrite() GatewayWriteOption {
	return func(options *gatewayWriteOptions) {
		options.overwrite = true
	}
}

// WithGatewayHistoryLimit sets how many numbered backups are retained.
// A zero limit uses DefaultGatewayHistoryLimit.
func WithGatewayHistoryLimit(limit int) GatewayWriteOption {
	return func(options *gatewayWriteOptions) {
		options.historyLimit = limit
	}
}

func evalGatewayWriteOptions(options ...GatewayWriteOption) (*gatewayWriteOptions, error) {
	result := &gatewayWriteOptions{}
	for _, option := range options {
		if option != nil {
			option(result)
		}
	}
	if result.backup && result.overwrite {
		return nil, errors.New("gateway write cannot request both backup and overwrite")
	}
	if result.historyLimit < 0 {
		return nil, errors.New("gateway history limit cannot be negative")
	}
	return result, nil
}

// WriteGateway serializes gateway to destination. Backup rotation mirrors the
// Gateway binary: older numbered history files shift upward and the current
// file becomes destination_0001.history, with the oldest copy pruned at the
// configured limit. WithGatewayOverwrite replaces an existing file without a
// backup. It returns the new backup path, if created.
func WriteGateway(destination string, gateway *Gateway, options ...GatewayWriteOption) (backupPath string, err error) {
	if destination == "" {
		return "", errors.New("gateway destination is empty")
	}
	if gateway == nil {
		return "", errors.New("gateway is nil")
	}
	writeOptions, err := evalGatewayWriteOptions(options...)
	if err != nil {
		return "", err
	}

	data, err := xml.MarshalIndent(gateway, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal gateway XML: %w", err)
	}
	data = append(data, '\n')

	lockPath := destination + ".lck"
	if err := os.Mkdir(lockPath, 0o700); err != nil {
		return "", fmt.Errorf("lock gateway destination %q: %w", destination, err)
	}
	defer os.Remove(lockPath)

	info, err := os.Lstat(destination)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("stat gateway destination %q: %w", destination, err)
	}
	if err == nil {
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("gateway destination %q is not a regular file", destination)
		}
		if !writeOptions.backup && !writeOptions.overwrite {
			return "", fmt.Errorf("gateway destination %q exists; choose backup or overwrite: %w", destination, fs.ErrExist)
		}
	}

	directory := filepath.Dir(destination)
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(destination)+".tmp-*")
	if err != nil {
		return "", fmt.Errorf("create temporary gateway file in %q: %w", directory, err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	mode := fs.FileMode(0o600)
	if info != nil {
		mode = info.Mode().Perm()
	}
	writeErr := func() (err error) {
		defer func() { err = errors.Join(err, temporary.Close()) }()
		if err := temporary.Chmod(mode); err != nil {
			return err
		}
		if _, err := temporary.Write(data); err != nil {
			return err
		}
		return temporary.Sync()
	}()
	if writeErr != nil {
		return "", fmt.Errorf("write temporary gateway file: %w", writeErr)
	}

	if info != nil && writeOptions.backup {
		historyLimit := writeOptions.historyLimit
		if historyLimit == 0 {
			historyLimit = DefaultGatewayHistoryLimit
		}
		if err := rotateGatewayHistory(destination, historyLimit); err != nil {
			return "", err
		}
		backupPath = gatewayHistoryPath(destination, 1)
		if err := os.Rename(destination, backupPath); err != nil {
			return "", fmt.Errorf("backup existing gateway file to %q: %w", backupPath, err)
		}
	}

	if err := os.Rename(temporaryPath, destination); err != nil {
		if backupPath != "" {
			if restoreErr := os.Rename(backupPath, destination); restoreErr != nil {
				return backupPath, errors.Join(fmt.Errorf("install gateway file: %w", err), fmt.Errorf("restore backup %q: %w", backupPath, restoreErr))
			}
		}
		return "", fmt.Errorf("install gateway file %q: %w", destination, err)
	}
	return backupPath, nil
}

func rotateGatewayHistory(destination string, limit int) error {
	oldest := gatewayHistoryPath(destination, limit)
	if err := os.Remove(oldest); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove oldest gateway history %q: %w", oldest, err)
	}
	for sequence := limit - 1; sequence >= 1; sequence-- {
		from := gatewayHistoryPath(destination, sequence)
		if _, err := os.Lstat(from); errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return fmt.Errorf("stat gateway history %q: %w", from, err)
		}
		to := gatewayHistoryPath(destination, sequence+1)
		if err := os.Rename(from, to); err != nil {
			return fmt.Errorf("shift gateway history from %q to %q: %w", from, to, err)
		}
	}
	return nil
}

func gatewayHistoryPath(destination string, sequence int) string {
	return fmt.Sprintf("%s_%04d.history", destination, sequence)
}
