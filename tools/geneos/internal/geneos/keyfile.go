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
)

// UseKeyFile sets the keyfile for component ct from either the given
// (local) keyfile or the CRC for an existing file.
func UseKeyFile(h *Host, ct *Component, keyfile config.KeyFile, keycrc string) (crc string, err error) {
	var path string

	if keycrc == "" {
		var crc32 uint32
		crc32, err = ImportSharedKey(h, ct, string(keyfile))
		crc = fmt.Sprintf("%08X", crc32)
		return
	}

	// if no CRC given then use the keyfile or the user's default one
	// search for existing CRC in all shared dirs

	// the crc may have come from a different host, check and remember.
	// look local first/only?
	crcfile := KeyFileNormalise(keycrc)

	onHost := h
	for _, h := range h.OrList(ALL) {
		for _, ct := range ct.OrList(UsesKeyFiles()...) {
			path = ct.Shared(h, "keyfiles", crcfile)
			log.Debug().Msgf("looking for keyfile %s on %s", path, h)
			if _, err := h.Stat(path); err == nil {
				onHost = h
				break
			}
			path = ""
		}
	}
	if path == "" {
		err = fmt.Errorf("keyfile (%q) with CRC %q not found", crcfile, keycrc)
		return
	}

	k, err := onHost.Open(path)
	if err != nil {
		return
	}
	defer k.Close()
	kv := config.Read(k)

	crc32, err := ImportSharedKeyValues(h, ct, kv)
	crc = fmt.Sprintf("%08X", crc32)
	return
}

// ImportSharedKey writes the contents of source to a shared keyfile on
// host h, component type ct. Host can be `ALL` and ct can be nil, in
// which case they are treated as wildcards. source can be a local file
// ("~/" relative to user home), a remote URL or "-" for STDIN.
func ImportSharedKey(h *Host, ct *Component, source string) (crc uint32, err error) {
	switch {
	case source == "":
		err = ErrInvalidArgs
		return
	case source == "-":
		// STDIN
		return ImportSharedKeyValues(h, ct, config.Read(os.Stdin))

	case strings.HasPrefix(source, "https://"), strings.HasPrefix(source, "http://"):
		// remote
		resp, err := http.Get(source)
		if err != nil {
			return crc, err
		}
		defer resp.Body.Close()
		return ImportSharedKeyValues(h, ct, config.Read(resp.Body))
	case strings.HasPrefix(source, "~/"):
		// relative to home
		home, _ := config.UserHomeDir()
		source = strings.Replace(source, "~", home, 1)
		fallthrough
	default:
		// local file
		keyfile := config.KeyFile(source)
		crc, _, err = keyfile.Check(false)
		if err != nil {
			return
		}
		kv, err := keyfile.Read()
		if err != nil {
			return crc, err
		}

		return ImportSharedKeyValues(h, ct, kv)
	}
}

// ImportSharedKeyValues writes key values kv to the host h and
// component type ct shared directory. Host can be ALL and ct can be
// nil, in which case they are treated as wildcards.
func ImportSharedKeyValues(h *Host, ct *Component, kv *config.KeyValues) (crc uint32, err error) {
	if kv == nil {
		err = ErrInvalidArgs
		return
	}
	crc, err = kv.Checksum()
	if err != nil {
		return
	}

	// at this point we have an AESValue struct and a CRC to use as
	// the filename base. create 'keyfiles' directory as required
	for _, h := range h.OrList(AllHosts()...) {
		for _, ct := range ct.OrList(UsesKeyFiles()...) {
			if err = WriteSharedKey(h, ct, kv); err != nil {
				return
			} else if err == nil {
				log.Debug().Msgf("not importing existing %q CRC named keyfile on %s", crc, h)
			}
		}
	}
	return
}

// WriteSharedKey writes key values kv to the shared keyfile directory
// for component ct on host h using the CRC32 checksum of the values as
// the base name. Both host h and component ct must be specific.
func WriteSharedKey(h *Host, ct *Component, kv *config.KeyValues) (err error) {
	if ct == nil || h == nil || h == ALL || kv == nil {
		return ErrInvalidArgs
	}

	crc, err := kv.Checksum()
	if err != nil {
		return err
	}
	crcstr := fmt.Sprintf("%08X", crc)

	// save given keyfile
	file := ct.Shared(h, "keyfiles", crcstr+".aes")
	if _, err := h.Stat(file); err == nil {
		fmt.Printf("keyfile %s.aes already exists in %s shared directory on %s\n", crcstr, ct, h)
		return nil
	}
	if err := h.MkdirAll(path.Dir(file), 0775); err != nil {
		log.Error().Err(err).Msgf("host %s, component %s", h, ct)
		return err
	}
	w, err := h.Create(file, 0600)
	if err != nil {
		log.Error().Err(err).Msgf("host %s, component %s", h, ct)
		return
	}
	defer w.Close()

	if err = kv.Write(w); err != nil {
		log.Error().Err(err).Msgf("host %s, component %s", h, ct)
	}
	fmt.Printf("keyfile %s.aes saved to %s shared directory on %s\n", crcstr, ct, h)
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
