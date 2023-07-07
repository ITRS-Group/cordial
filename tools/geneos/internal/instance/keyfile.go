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

package instance

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// UseKeyFile sets the keyfile for component ct from either the given
// (local) keyfile or the CRC for an existing file.
func UseKeyFile(h *geneos.Host, ct *geneos.Component, keyfile config.KeyFile, keycrc string) (crc string, err error) {
	var path string

	if keycrc == "" {
		var crc32 uint32
		crc32, err = ImportKeyFile(h, ct, keyfile)
		crc = fmt.Sprintf("%08X", crc32)
		return
	}

	// if no CRC given then use the keyfile or the user's default one
	// search for existing CRC in all shared dirs

	// the crc may have come from a different host, check and remember.
	// look local first/only?
	crcfile := KeyFileNormalise(keycrc)

	onHost := h
	for _, ct := range ct.OrList(geneos.UsesKeyFiles()...) {
		for _, h := range h.OrList(geneos.ALL) {
			path = ct.SharedPath(h, "keyfiles", crcfile)
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

	crc32, err := ImportKeyValues(h, ct, kv)
	crc = fmt.Sprintf("%08X", crc32)
	return
}

// ImportKeyFile copies the keyfile to the host h and component type ct
// shared directory. Host can be geneos.ALL and ct can be nil, in which
// case they are treated as wildcards.
func ImportKeyFile(h *geneos.Host, ct *geneos.Component, keyfile config.KeyFile) (crc uint32, err error) {
	crc, _, err = keyfile.Check(false)
	if err != nil {
		return
	}
	kv, err := keyfile.Read()
	if err != nil {
		return
	}

	return ImportKeyValues(h, ct, kv)
}

// ImportKeyValues saves a keyfile with values kv to the host h and
// component type ct shared directory. Host can be geneos.ALL and ct can
// be nil, in which case they are treated as wildcards.
func ImportKeyValues(h *geneos.Host, ct *geneos.Component, kv *config.KeyValues) (crc uint32, err error) {
	crc, err = kv.Checksum()
	if err != nil {
		return
	}

	// at this point we have an AESValue struct and a CRC to use as
	// the filename base. create 'keyfiles' directory as required
	for _, ct := range ct.OrList(geneos.UsesKeyFiles()...) {
		for _, h := range h.OrList(geneos.AllHosts()...) {
			if err = SaveKeyFileShared(h, ct, kv); err != nil {
				return
			} else if err == nil {
				log.Debug().Msgf("not importing existing %q CRC named keyfile on %s", crc, h)
			}
		}
	}
	return
}

// SaveKeyFileShared saves a key file with values a to the shared
// keyfile directory for component ct on host h
func SaveKeyFileShared(h *geneos.Host, ct *geneos.Component, a *config.KeyValues) (err error) {
	if ct == nil || h == nil || a == nil {
		return geneos.ErrInvalidArgs
	}

	crc, err := a.Checksum()
	if err != nil {
		return err
	}
	crcstr := fmt.Sprintf("%08X", crc)

	// save given keyfile
	file := ct.SharedPath(h, "keyfiles", crcstr+".aes")
	if _, err := h.Stat(file); err == nil {
		fmt.Printf("keyfile %s.aes already exists in shared directory for %s on %s\n", crcstr, ct, h)
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

	if err = a.Write(w); err != nil {
		log.Error().Err(err).Msgf("host %s, component %s", h, ct)
	}
	fmt.Printf("keyfile %s.aes saved to shared directory for %s on %s\n", crcstr, ct, h)
	return
}

// KeyFileNormalise returns the input in for format "DIR/HEX.aes" where
// HEX is an 8 hexadecimal digit string in uppercase and DIR is any
// leading path before the file name. If the input is not either an 8
// digit hex string (in any case) with or without the extension ".aes"
// (in any case) then the input is returned unchanged.
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
