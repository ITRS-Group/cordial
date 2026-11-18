/*
Copyright © 2026 ITRS Group

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

package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/itrs-group/cordial/pkg/certs"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
)

// tlsState is the TLS state change made to an existing instance.
type tlsState int

const (
	tlsUnchanged tlsState = iota
	tlsEnabled
	tlsDisabled
)

// tlsParameters are the instance parameters that make an instance TLS
// capable, and the references that go with them. The files that they
// refer to are never removed.
func tlsParameters(cf *config.Config) []string {
	return []string{
		cf.Join(instance.TLSBASE, instance.CERTIFICATE),
		cf.Join(instance.TLSBASE, instance.PRIVATEKEY),
		cf.Join(instance.TLSBASE, instance.CABUNDLE),
		instance.CERTIFICATE,
		instance.PRIVATEKEY,
		instance.CERTCHAIN,
		instance.USECHAIN,
	}
}

// applyTLSState compares the TLS state of the existing instance i with
// the effective `insecure` option of the profile and changes the
// instance to match. The instance configuration is updated but not
// written, and the caller must rebuild the instance so that the
// component specific settings, such as the port conventions, follow.
//
//   - secure and not TLS capable: reuse the certificate and key kept
//     from an earlier secure state, if still valid, otherwise create the
//     instance certificate as `geneos add` does. If there is no signing
//     certificate available this is an error, and the instance is not
//     left insecure silently.
//
//   - insecure and TLS parameters configured: remove them from the
//     instance configuration only. The certificate and key files are
//     kept for a later switch back to secure.
//
//   - otherwise nothing changes. This includes a profile with no saved
//     `insecure` option, such as one saved by earlier versions.
//
// A certificate bundle given in the profile is imported by the caller
// before this and so takes precedence over the option, as it does when
// an instance is created.
func applyTLSState(i geneos.Instance, pf *config.Config, common Common) (state tlsState, err error) {
	key := pf.Join("config", "options", "insecure")
	if !pf.IsSet(key) {
		return tlsUnchanged, nil
	}
	secure := !config.Get[bool](pf, key) || common.CertBundle != ""
	cf := i.Config()

	if secure {
		if instance.IsTLSCapable(i) {
			return tlsUnchanged, nil
		}
		if !restoreCertificate(i) {
			resp := instance.NewCertificate(i)
			if resp.Err != nil {
				return tlsUnchanged, fmt.Errorf("profile requires TLS for %s but a certificate cannot be created, use `geneos tls init` or the profile option --insecure: %w", i, resp.Err)
			}
		}
		return tlsEnabled, nil
	}

	configured := false
	for _, k := range tlsParameters(cf) {
		if cf.IsSet(k) {
			configured = true
			config.Delete(cf, k)
		}
	}
	if !configured {
		return tlsUnchanged, nil
	}

	if i.Type().IsA("webserver") {
		if err = disableWebserverKeystore(i); err != nil {
			return tlsUnchanged, err
		}
	}
	return tlsDisabled, nil
}

// restoreCertificate sets the instance parameters to the certificate
// and key files in the standard locations, which were left there when
// TLS was disabled, if they are present, valid and match each other.
func restoreCertificate(i geneos.Instance) bool {
	h := i.Host()
	certFile := instance.ComponentFilepath(i, certs.PEMExtension)
	keyFile := instance.ComponentFilepath(i, certs.KEYExtension)

	chain, err := certs.ReadCertificates(h, certFile)
	if err != nil || len(chain) == 0 {
		return false
	}
	key, err := certs.ReadPrivateKey(h, keyFile)
	if err != nil || key == nil {
		return false
	}
	leaf := chain[0]
	if !certs.IsValidLeafCert(leaf) || certs.IndexPrivateKey([]certs.PrivateKey{key}, leaf) != 0 {
		return false
	}

	cf := i.Config()
	config.Delete(cf, instance.CERTIFICATE)
	config.Delete(cf, instance.PRIVATEKEY)
	config.Set(cf, cf.Join(instance.TLSBASE, instance.CERTIFICATE), certFile, config.Replace("home"))
	config.Set(cf, cf.Join(instance.TLSBASE, instance.PRIVATEKEY), keyFile, config.Replace("home"))
	config.Set(cf, cf.Join(instance.TLSBASE, instance.CABUNDLE), geneos.PathToCABundlePEM(h))
	return true
}

// disableWebserverKeystore sets aside the keystore of a Web Dashboard
// server, as the server is started with HTTPS whenever its keystore
// exists. The keystore is only built from the instance certificate and
// key, and is created again when TLS is enabled, so it is renamed with a
// `.disabled` suffix and not removed. A keystore that is also the
// truststore is left alone.
//
// The keystores of an `sso-agent` are not touched, as it always has one
// for its signing key and its TLS setting is in its own
// configuration file.
func disableWebserverKeystore(i geneos.Instance) error {
	h := i.Host()
	spPath := instance.HomeRel(i, "config/security.properties")
	sp, err := instance.ReadKVConfig(h, spPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("reading %s: %w", spPath, err)
	}
	if sp["keyStore"] == "" {
		return nil
	}
	keystore := instance.HomeRel(i, sp["keyStore"])
	if sp["trustStore"] != "" && instance.HomeRel(i, sp["trustStore"]) == keystore {
		return nil
	}
	if _, err = h.Stat(keystore); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	i.Log().Debug("setting aside keystore for insecure instance", slog.String("keystore", keystore))
	if err = h.Rename(keystore, keystore+".disabled"); err != nil {
		return fmt.Errorf("setting aside keystore %s: %w", keystore, err)
	}
	return nil
}
