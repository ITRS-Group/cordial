/*
Copyright © 2023 ITRS Group

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
	"net/http"
	"os"
	"path"
	"strings"

	zlog "github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
)

// DefaultUserKeyfile is the path to the user's key file as a
// config.Keyfile type
var DefaultUserKeyfile = config.KeyFile(
	config.Path("keyfile",
		config.AppName(cordial.ExecutableName()),
		config.Format("aes"),
		config.SkipWorkingDir(),
	),
)

// KeyFilePath returns the absolute path to either the given keyfile or
// a shared keyfile with the CRC of keycrc, if keyfile is not set. If ct
// is nil then the first matching keyfile from all components is
// returned. If h is ALL then only localhost is checked.
func (ct *Component) KeyFilePath(h *Host, keyfile config.KeyFile, keycrc string) (path string, err error) {
	if keyfile != "" {
		return h.Abs(string(keyfile))
	}

	if keycrc == "" {
		return "", ErrNotExist
	}

	return ct.Shared(h, "keyfiles", keycrc+".aes"), nil
}

// ReadKeyValues returns a *config.KeyValues in kv containing the key
// values from the source. `source` can be a path to a file, a `-` for
// STDIN (in which case an optional prompt is output) or a remote URL.
func ReadKeyValues(source string, prompt ...string) (kv *config.KeyValues, err error) {
	switch {
	case source == "":
		err = ErrInvalidArgs
		return
	case source == "-":
		// STDIN, prefix with prompt if given
		if len(prompt) > 0 && prompt[0] != "" {
			fmt.Println(prompt[0])
		}
		kv, err = config.ReadKeyValues(os.Stdin)
		if err != nil {
			return kv, err
		}
	case strings.HasPrefix(string(source), "https://"), strings.HasPrefix(string(source), "http://"):
		// remote
		resp, err := http.Get(string(source))
		if err != nil {
			return kv, err
		}
		defer resp.Body.Close()
		kv, err = config.ReadKeyValues(resp.Body)
		if err != nil {
			return kv, err
		}
	default:
		// local file, read and write to new locations
		keyfile := config.KeyFile(config.ResolveHome(source))
		_, err = keyfile.ReadCRC(host.Localhost)
		if err != nil {
			return
		}
		kv, err = keyfile.Read(host.Localhost)
		if err != nil {
			return kv, err
		}

	}

	return
}

// ImportSharedKey writes the contents of source to a shared keyfile on
// host h, component type ct. Host can be `ALL` and ct can be nil, in
// which case they are treated as wildcards. keyfile can be a local file
// ("~/" relative to user home), a remote URL or "-" for STDIN.
func ImportSharedKey(h *Host, ct *Component, source string, prompt ...string) (paths []string, crc uint32, err error) {
	switch {
	case source == "":
		err = ErrInvalidArgs
		return
	case source == "-":
		// STDIN, prefix with prompt if given
		if len(prompt) > 0 {
			fmt.Println(prompt[0])
		}
		kv, err := config.ReadKeyValues(os.Stdin)
		if err != nil {
			return paths, crc, err
		}
		return WriteSharedKeyValues(h, ct, kv)

	case strings.HasPrefix(string(source), "https://"), strings.HasPrefix(string(source), "http://"):
		// remote
		resp, err := http.Get(string(source))
		if err != nil {
			return paths, crc, err
		}
		defer resp.Body.Close()
		kv, err := config.ReadKeyValues(resp.Body)
		if err != nil {
			return paths, crc, err
		}
		return WriteSharedKeyValues(h, ct, kv)
	default:
		// local file, read and write to new locations
		keyfile := config.KeyFile(config.ResolveHome(source))
		_, err = keyfile.ReadCRC(host.Localhost)
		if err != nil {
			return
		}
		kv, err := keyfile.Read(host.Localhost)
		if err != nil {
			return paths, crc, err
		}

		return WriteSharedKeyValues(h, ct, kv)
	}
}

// WriteSharedKeyValues writes key values kv to the host h and component
// type ct shared directory in a file `CRC.aes`. Host can be ALL and ct
// can be nil, in which case they are treated as wildcards.
func WriteSharedKeyValues(h *Host, ct *Component, kv *config.KeyValues) (paths []string, crc uint32, err error) {
	if crc, err = kv.Checksum(); err != nil {
		return
	}

	// at this point we have an AESValue struct and a CRC to use as
	// the filename base. create 'keyfiles' directory as required
	for h := range h.OrList() {
		for ct := range ct.OrList(UsesKeyFiles()...) {
			var p string
			if p, err = writeSharedKey(h, ct, kv); err != nil {
				return
			}
			paths = append(paths, p)
		}
	}
	return
}

// writeSharedKey writes key values kv to the shared keyfile directory
// for component ct on host h using the CRC32 checksum of the values as
// the base name. Both host h and component ct must be specific.
func writeSharedKey(h *Host, ct *Component, kv *config.KeyValues) (p string, err error) {
	if ct == nil || h == nil || h == ALL || kv == nil {
		return "", ErrInvalidArgs
	}

	crc, err := kv.ChecksumString()
	if err != nil {
		return
	}

	// save given keyfile
	p = ct.Shared(h, "keyfiles", crc+".aes")
	if _, err = h.Stat(p); err == nil {
		fmt.Printf("keyfile %s.aes already exists in %s shared directory on %s\n", crc, ct, h)
		return
	}
	if err = h.MkdirAll(path.Dir(p), 0775); err != nil {
		zlog.Error().Err(err).Msgf("host %s, component %s", h, ct)
		return
	}
	w, err := h.Create(p, 0600)
	if err != nil {
		zlog.Error().Err(err).Msgf("host %s, component %s", h, ct)
		return
	}
	defer w.Close()

	if err = kv.Write(w); err != nil {
		zlog.Error().Err(err).Msgf("host %s, component %s", h, ct)
	}
	zlog.Debug().Msgf("keyfile saved to %s on %s", p, h)
	return
}
