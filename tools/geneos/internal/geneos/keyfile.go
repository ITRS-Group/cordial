/*
Copyright © 2023 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package geneos

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/host"
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

// ReadKeyValues returns a memguard enclave in kv containing the key
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
	case strings.HasPrefix(string(source), "~/"):
		// relative to home
		home, _ := config.UserHomeDir()
		source = strings.Replace(source, "~", home, 1)
		fallthrough
	default:
		// local file, read and write to new locations
		keyfile := config.KeyFile(source)
		_, _, err = keyfile.ReadOrCreate(host.Localhost, false)
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
	case strings.HasPrefix(string(source), "~/"):
		// relative to home
		home, _ := config.UserHomeDir()
		source = strings.Replace(source, "~", home, 1)
		fallthrough
	default:
		// local file, read and write to new locations
		keyfile := config.KeyFile(source)
		_, _, err = keyfile.ReadOrCreate(host.Localhost, false)
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
	for _, h := range h.OrList(AllHosts()...) {
		for _, ct := range ct.OrList(UsesKeyFiles()...) {
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
		log.Error().Err(err).Msgf("host %s, component %s", h, ct)
		return
	}
	w, err := h.Create(p, 0600)
	if err != nil {
		log.Error().Err(err).Msgf("host %s, component %s", h, ct)
		return
	}
	defer w.Close()

	if err = kv.Write(w); err != nil {
		log.Error().Err(err).Msgf("host %s, component %s", h, ct)
	}
	log.Debug().Msgf("keyfile saved to %s on %s", p, h)
	return
}

// KeyFileNormalise returns the input in for format "DIR/HEX.aes" where
// HEX is an 8 hexadecimal digit string in uppercase and DIR is any
// leading path before the file name. If the input is neither an 8 digit
// hex string (in upper or lower case) with or without the extension
// ".aes" (in upper or lower case) then the input is returned unchanged.
func KeyFileNormalise(in string) (out string) {
	out = in

	dir, file := path.Split(in)
	file = strings.ToUpper(file)
	ext := path.Ext(file) // ext is now in UPPER case

	log.Debug().Msgf("dir=%s file=%s ext=%s", dir, file, ext)

	if ext != "" && ext != ".AES" {
		return
	}
	file = strings.TrimSuffix(file, ext)

	hex, err := strconv.ParseUint(file, 16, 32)
	if err != nil {
		log.Debug().Err(err).Msg("")
		return
	}

	if fmt.Sprintf("%08X", hex) != file {
		log.Debug().Msgf("hex and file not the same: %X != %s", hex, file)
		return
	}

	if dir == "" {
		log.Debug().Msgf("returning: %s", file+".aes")
		return file + ".aes"
	}

	log.Debug().Msgf("returning: %s/%s", dir, file+".aes")
	return path.Join(dir, file+".aes")
}
