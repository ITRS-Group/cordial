package geneos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
)

// openRemoteArchive locates and opens a remote software archive file
// using the Geneos download conventions. It returns the underlying
// filename and the archives as a http.Response object.
//
// GeneosOptions supported are PlatformID, UseNexus, UseSnapshots,
// Version, Username and Password. PlatformID and Version cannot be set
// at the same time.
func openRemoteArchive(ct *Component, options ...PackageOption) (filename string, resp *http.Response, err error) {
	var source string

	opts := evalOptions(options...)

	switch opts.downloadtype {
	case "nexus":
		source, resp, err = openRemoteNexusArchive(ct, opts)
		if err != nil {
			return
		}

	default:
		source, resp, err = openRemoteDefaultArchive(ct, opts)
		if err != nil {
			return
		}
	}

	// process both nexus and resources status codes below
	if resp.StatusCode > 299 {
		resp.Body.Close()
		switch resp.StatusCode {
		case 404:
			fmt.Printf("cannot find %s package that matches version %s\n", ct, opts.version)
			err = fs.ErrNotExist
		default:
			err = fmt.Errorf("cannot access %s package at %q version %s: %s", ct, source, opts.version, resp.Status)
		}
		return
	}

	filename, err = FilenameFromHTTPResp(resp, resp.Request.URL)
	if err != nil {
		return
	}

	log.Debug("download check for versions", slog.String("component", ct.String()), slog.String("version", opts.version), slog.String("filename", filename), slog.Int64("contentLength", resp.ContentLength))
	return
}

func openRemoteDefaultArchive(ct *Component, opts *packageOptions) (source string, resp *http.Response, err error) {
	// cannot fetch partial versions for OSes with platformID set - restriction on download search interface
	platform := getPlatformId(opts.platformId)

	cf := config.Global()

	baseurl := config.Get[string](cf, cf.Join("download", "url"))
	downloadURL, _ := url.Parse(baseurl)

	os := config.Get[string](opts.host.Config, "os")
	arch := osMap[config.Get[string](opts.host.Config, "arch")]

	v := url.Values{}

	if ct.DownloadParams == nil {
		v.Set("os", os)

		if opts.version != "latest" {
			if platform != "" {
				log.Error("cannot download specific version for this platform - please download manually", slog.String("platform", platform))
				err = ErrInvalidArgs
				return
			}
			v.Set("title", opts.version)
		} else if platform != "" {
			v.Set("title", "-"+platform+"-"+os+"-"+arch)
		} else {
			v.Set("title", os+"-"+arch)
		}
	} else {
		for _, param := range *ct.DownloadParams {
			if key, value, found := strings.Cut(param, "="); found {
				v.Set(key, value)
			}
		}
		if opts.version != "latest" {
			v.Set("title", opts.version)
		}
	}

	basepaths := strings.FieldsFunc(ct.DownloadBase.Default, func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	})

	timeout := config.Get[time.Duration](cf, cf.Join("download", "timeout"), config.DefaultValue(60*time.Second))
	client := &http.Client{
		Timeout: timeout,
	}

	for _, bp := range basepaths {
		var authReader io.Reader
		var authBody []byte

		// first try plain unauthenticated GET
		basepath, _ := url.Parse(bp)
		basepath.RawQuery = v.Encode()
		source = downloadURL.ResolveReference(basepath).String()

		log.Debug("source url", slog.String("url", source))

		var req *http.Request
		req, err = http.NewRequest("GET", source, nil)
		if err != nil {
			log.Error("source, trying next if configured", slog.Any("error", err))
			continue
		}

		// add any headers
		for _, h := range opts.headers {
			name, value, found := strings.Cut(h, "=")
			if found {
				req.Header.Add(name, value)
			}
		}

		req1 := req.Clone(req.Context())

		if resp, err = client.Do(req1); err != nil {
			log.Error("source, trying next if configured", slog.Any("error", err))
			continue
		}

		if resp.StatusCode < 300 {
			return
		}

		// if we get a not found status and we are looking for a
		// platform specific archive then also try without the platform
		// in case the release is not available for the platform, eg
		// webserver
		if resp.StatusCode == 404 && platform != "" {
			resp.Body.Close()
			if ct.DownloadParams == nil {
				v.Set("title", os+"-"+arch)
			} else {
				v.Del("title")
			}
			basepath.RawQuery = v.Encode()
			source = downloadURL.ResolveReference(basepath).String()

			log.Debug("platform download failed, retry source url", slog.String("url", source))
			req2 := req.Clone(req.Context())
			if resp, err = client.Do(req2); err != nil {
				log.Error("source, trying next if configured", slog.Any("error", err))
				continue
			}
			if resp.StatusCode < 300 {
				return
			}
		}

		resp.Body.Close()

		// if that fails, check for creds
		if opts.username == "" {
			creds := config.FindCreds(source, config.AppName(cordial.ExecutableName()))
			if creds != nil {
				opts.username = config.Get[string](creds, "username")
				opts.password = config.Get[config.Secret](creds, "password")
			}
		}

		if opts.username != "" {
			da := downloadauth{
				Username: opts.username,
				Password: string(opts.password),
			}
			authBody, err = json.Marshal(da)
			if err != nil {
				log.Error("source, trying next if configured", slog.Any("error", err))
				continue
			}
			// make a copy as bytes.NewBuffer() takes ownership
			ba := bytes.Clone(authBody)
			authReader = bytes.NewBuffer(ba)
		}

		if authReader == nil {
			log.Error("source requires authentication but no credentials found, trying next if configured")
			continue
		}
		log.Debug("retrying source with auth", slog.String("url", source))

		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			req, err = http.NewRequest("POST", source, authReader)
			if err != nil {
				log.Error("source, trying next if configured", slog.Any("error", err))
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			// add any headers
			for _, h := range opts.headers {
				name, value, found := strings.Cut(h, "=")
				if found {
					req.Header.Add(name, value)
				}
			}
			if resp, err = client.Do(req); err != nil {
				log.Error("source, trying next if configured", slog.Any("error", err))
				continue
			}
			if resp.StatusCode < 300 {
				return
			}
		}

		resp.Body.Close()

		if resp.StatusCode == 404 && platform != "" {
			if ct.DownloadParams == nil {
				v.Set("title", os+"-"+arch)
			} else {
				v.Del("title")
			}
			basepath.RawQuery = v.Encode()
			source = downloadURL.ResolveReference(basepath).String()

			log.Debug("trying source with auth", slog.String("url", source))
			req, err = http.NewRequest("POST", source, authReader)
			if err != nil {
				log.Error("source, trying next if configured", slog.Any("error", err))
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			// add any headers
			for _, h := range opts.headers {
				name, value, found := strings.Cut(h, "=")
				if found {
					req.Header.Add(name, value)
				}
			}
			if resp, err = client.Do(req); err != nil {
				log.Error("source, trying next if configured", slog.Any("error", err))
				continue
			}
			if resp.StatusCode < 300 {
				return
			}
		}
		log.Debug("source not found, trying next if configured", slog.String("url", source))
	}
	return
}

func openRemoteNexusArchive(ct *Component, opts *packageOptions) (source string, resp *http.Response, err error) {
	os := config.Get[string](opts.host.Config, "os")
	arch := osMap[config.Get[string](opts.host.Config, "arch")]

	platform := ""
	if opts.platformId != "" {
		s := strings.Split(opts.platformId, ":")
		if len(s) > 1 {
			platform = s[1]
		}
	}

	baseurl := config.Get[string](
		config.Global(),
		config.Join("download", "nexus", "url"),
		config.DefaultValue("https://nexus.itrsgroup.com/service/rest/v1/search/assets/download"),
	)
	downloadURL, _ := url.Parse(baseurl)

	v := url.Values{}
	v.Set("sort", "version")
	v.Set("repository", opts.downloadbase)

	if ct.DownloadParamsNexus == nil {
		v.Set("maven.groupId", "com.itrsgroup.geneos")
		v.Set("maven.extension", "tar.gz")
		if platform != "" {
			v.Set("maven.classifier", platform+"-"+os+"-"+arch)
		} else {
			v.Set("maven.classifier", os+"-"+arch)
		}
	} else {
		for _, param := range *ct.DownloadParamsNexus {
			if key, value, found := strings.Cut(param, "="); found {
				v.Set(key, value)
			}
		}
	}

	if opts.version != "latest" {
		v.Set("maven.baseVersion", opts.version)
	}

	// check for fallback creds
	if opts.username == "" {
		creds := config.FindCreds(baseurl, config.AppName(cordial.ExecutableName()))
		if creds != nil {
			opts.username = config.Get[string](creds, "username")
			opts.password = config.Get[config.Secret](creds, "password")
		}
	}

	timeout := config.Get[time.Duration](config.Global(), config.Join("download", "timeout"), config.DefaultValue(60*time.Second))
	client := &http.Client{
		Timeout: timeout,
	}

	var req *http.Request

	artifacts := strings.FieldsFunc(ct.DownloadBase.Nexus, func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	})

	for _, artifactId := range artifacts {
		v.Set("maven.artifactId", artifactId)
		downloadURL.RawQuery = v.Encode()
		source = downloadURL.String()
		log.Debug("nexus url", slog.String("url", source))
		if req, err = http.NewRequest("GET", source, nil); err != nil {
			return
		}
		// add any headers
		for _, h := range opts.headers {
			name, value, found := strings.Cut(h, "=")
			if found {
				req.Header.Add(name, value)
			}
		}
		if opts.username != "" {
			log.Debug("setting creds", slog.String("username", opts.username))
			req.SetBasicAuth(opts.username, string(opts.password))
		}
		if resp, err = client.Do(req); err != nil {
			log.Debug("request failed", slog.Any("error", err))
			return
		}
		log.Debug("response status", slog.String("status", resp.Status))
		if resp.StatusCode < 300 {
			return
		}
	}
	return
}
