/*
Copyright © 2022 ITRS Group

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

package values

import (
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// interfaces for pflag Var interface

type SecureValues []*SecureValue

type SecureValue struct {
	Value      string
	Secret     config.Secret
	Ciphertext string
}

func (p *SecureValues) String() string {
	return ""
}

// Set a SecureValue. If there is a "=VALUE" part then this is saved in
// Secret, otherwise only the NAME is set. This allows later
// processing to either encode the Secret into Ciphertext or to
// prompt the user for a secret
func (p *SecureValues) Set(v string) error {
	if p == nil {
		return geneos.ErrInvalidArgs
	}
	value, secret, found := strings.Cut(v, "=")
	if !found {
		*p = append(*p, &SecureValue{
			Value: value,
		})
	} else {
		*p = append(*p, &SecureValue{
			Value:  value,
			Secret: config.Secret(secret),
		})
	}
	return nil
}

func (p *SecureValues) Type() string {
	return "NAME[=VALUE]"
}
