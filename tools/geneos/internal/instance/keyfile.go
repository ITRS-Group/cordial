package instance

import (
	"path/filepath"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// CreateAESKeyFile creates a new key file, for secure passwords as per
// https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm
func CreateAESKeyFile(i geneos.Instance) (err error) {
	kv := config.NewRandomKeyValues()

	_, _, err = WriteAESKeyFile(i, kv)
	return
}

// ReadAESKeyFile locates the path to the keyfile in the instance
// configuration, using the first setting if passed otherwise defaulting
// to `keyfile`. If found, return the key values in kv or an error,
func ReadAESKeyFile(i geneos.Instance, setting ...string) (keyfile config.KeyFile, kv *config.KeyValues, crc uint32, err error) {
	k := "keyfile"
	if len(setting) > 0 && setting[0] != "" {
		k = "keyfile"
	}
	kp := i.Config().GetString(k)
	if kp == "" {
		err = geneos.ErrNotExist
		return
	}
	keyfile = config.KeyFile(kp)
	kv, err = keyfile.Read(i.Host())
	if err != nil {
		crc, err = kv.Checksum()
		// fallthrough and return any err
	}
	return
}

// WriteAESKeyFile writes key values to a an instance key file, for
// secure passwords as per
// https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm
//
// If the instance config is updated then it is also saved.
func WriteAESKeyFile(i geneos.Instance, kv *config.KeyValues) (keyfile config.KeyFile, crc uint32, err error) {
	crc, err = kv.Checksum()
	if err != nil {
		return
	}
	keyfile = config.KeyFile(ComponentFilepath(i, "aes"))
	w, err := i.Host().Create(string(keyfile), 0600)
	if err != nil {
		return
	}
	defer w.Close()

	if err = kv.Write(w); err != nil {
		return
	}

	i.Config().Set("keyfile", keyfile)

	err = SaveConfig(i)
	return
}

// RollAESKeyFile moves any existing instance key file to a "previous"
// filename unless it is in a shared keyfiles directory, in which case
// it sets `prevkeyfile` to this path and saves the new key file values
// nkv to the instance directory.
//
// Otherwise, the current keyfile is renamed to a file where the backup
// string is appended to the base name before the extension and the key
// file values nkv are written to a new file and the instance settings
// updated.
func RollAESKeyFile(i geneos.Instance, nkv *config.KeyValues, backup string) (keyfile config.KeyFile, crc uint32, err error) {
	kp := i.Config().GetString("keyfile")

	// if existing keyfile is in a shared keyfile folder do not backup
	// and write new keyfile in instance folder
	if filepath.Dir(kp) == i.Type().Shared(i.Host(), "keyfiles") {
		i.Config().Set("prevkeyfile", kp)
		return WriteAESKeyFile(i, nkv)
	}

	// read any existing keyfile, backup if suffix given and not in
	// component shared directory with CRC prefix
	if backup != "" {
		ext := filepath.Ext(kp)
		basename := strings.TrimSuffix(filepath.Base(kp), ext)
		dir := filepath.Dir(kp)
		bkp := filepath.Join(dir, basename+backup+ext)
		if err = i.Host().Rename(kp, bkp); err != nil {
			return "", 0, err
		}
		i.Config().Set("prevkeyfile", bkp)
		// fallthrough
	}

	keyfile, crc, err = WriteAESKeyFile(i, nkv)
	return
}
