/*
Copyright © 2024 ITRS Group

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

package cmd

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func openSource(ctx context.Context, source string) (io.ReadCloser, error) {
	if strings.HasPrefix(source, "~/") {
		var home string
		home, err := config.UserHomeDir()
		if err != nil {
			return nil, err
		}
		source = filepath.Join(home, strings.TrimPrefix(source, "~/"))
	}
	u, err := url.Parse(source)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "https":
		log.Trace().Msgf("reading data from %s", source)
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: cf.GetBool("gdna.licd-skip-verify")},
		}
		client := &http.Client{Transport: otelhttp.NewTransport(tr), Timeout: cf.GetDuration("gdna.licd-timeout")}
		u = u.JoinPath(DetailsPath)
		req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode > 299 {
			resp.Body.Close()
			return nil, fmt.Errorf("server returned %s", resp.Status)
		}
		return resp.Body, nil
	case "http":
		log.Trace().Msgf("reading data from %s", source)
		client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport), Timeout: cf.GetDuration("gdna.licd-timeout")}
		u = u.JoinPath(DetailsPath)
		req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode > 299 {
			resp.Body.Close()
			return nil, fmt.Errorf("server returned %s", resp.Status)
		}
		return resp.Body, nil
	default:
		log.Trace().Msgf("reading data from file '%s'", source)

		if strings.HasPrefix(source, "~/") {
			home, _ := config.UserHomeDir()
			source = path.Join(home, strings.TrimPrefix(source, "~/"))
		}

		var s os.FileInfo
		s, err = os.Stat(source)
		if err != nil {
			return nil, err
		}
		if s.IsDir() {
			return nil, os.ErrInvalid // geneos.ErrIsADirectory
		}
		log.Trace().Msgf("reading from %s (modtime %s)", source, s.ModTime().UTC().Format(time.RFC3339))
		source, _ = filepath.Abs(source)
		source = filepath.ToSlash(source)
		return os.Open(source)
	}
}

// readLicdReport tries to read the three CSV sections from the latest
// (lexically) licd report file from path into memory. this format is
// not documented, and this is based on sample files. returns 3 CSV
// readers, one per section, the basename of the source file and the
// modtime as well as any error
func readLicdReports(ctx context.Context, cf *config.Config, tx *sql.Tx, source string,
	fn func(context.Context, *config.Config, *sql.Tx, *csv.Reader, string, string, string, time.Time) error) (sources []string, err error) {
	source = config.ExpandHome(source)
	matches, err := filepath.Glob(source)
	if err != nil {
		return
	}

	if len(matches) == 0 {
		log.Info().Msgf("no matches found for %s", source)
		return
	}

	// assume files include date times and they are ordered lexically
	slices.Sort(matches)

	// source = slices.Max(matches)
	for _, source := range matches {
		var err error
		st, err := os.Stat(source)
		if err != nil {
			log.Error().Err(err).Msg("")
			// updateSources(ctx, cf, tx, "licd:"+source, "licd", source, false, time.Now(), err)
			continue
		}
		sourceTimestamp := st.ModTime()
		s, _, c, err := readLicdReport(source)
		if err != nil {
			log.Error().Err(err).Msg("")
			// updateSources(ctx, cf, tx, "licd:"+source, "licd", source, false, time.Now(), err)
			continue
		}

		// pull out symbolic source name from summary
		var expiry, licenceName, sourceName string
		for {
			r, err := s.Read()
			if err != nil && errors.Is(err, io.EOF) {
				break
			}
			switch r[0] {
			case "expiry":
				expiry = r[1]
			case "licenceName":
				licenceName = r[1]
			default:
				continue
			}
		}

		// need both
		if expiry == "" || licenceName == "" {
			continue
		}

		t, err := time.Parse("02 January 2006", expiry)
		if err != nil {
			log.Error().Err(err).Msgf("cannot parse %s", expiry)
		}

		sourceName = "licd:" + licenceName + "_" + t.Format(time.DateOnly)
		sources = append(sources, sourceName)

		log.Debug().Msgf("processing licd report file %s using label %s", source, sourceName)
		if err = fn(ctx, cf, tx, c, sourceName, "licd", source, sourceTimestamp); err != nil {
			updateSources(ctx, cf, tx, sourceName, "licd", source, false, sourceTimestamp, err)
			log.Error().Err(err).Msgf("cannot process licd report file %s", source)
		}
	}
	return
}

func readLicdReport(source string) (summary, tokenUsage, details *csv.Reader, err error) {
	r, err := os.Open(source)
	if err != nil {
		return
	}

	// first identify sections and create SectionReaders, then create new CSV readers
	b, err := io.ReadAll(r)
	if err != nil {
		return
	}
	r.Close()

	var sections []*csv.Reader

	for _, s := range []string{"samplingStatus", "Group", "Req Number"} {
		var start, end int

		start = bytes.Index(b, []byte(s))
		if start == -1 {
			err = fmt.Errorf("cannot find section starting '%s' in %s", s, source)
			return
		}
		b = b[start:]
		end = bytes.Index(b, []byte{0})
		if end == -1 {
			err = fmt.Errorf("cannot locate end of section '%s' in %s", s, source)
			return
		}
		sections = append(sections, csv.NewReader(bytes.NewBuffer(b[:end])))
		b = b[end:]
	}

	if len(sections) != 3 {
		err = fmt.Errorf("%d sections found in %s, not the expected 3", len(sections), source)
		return
	}

	summary = sections[0]
	tokenUsage = sections[1]
	details = sections[2]

	return
}
