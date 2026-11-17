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

package imscmd

import (
	"errors"
	"fmt"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/pkg/ims"
	"github.com/itrs-group/cordial/pkg/rest"
)

// tryGateways calls fn with a client for each configured IMS gateway
// URL in turn, as described for [ims.Try], returning an error that
// names the configuration key if there are no gateways configured.
func tryGateways(cf *config.Config, imsType string, fn func(rc *rest.Client) error) error {
	err := ims.Try(cf.Sub("ims-gateway"), imsType, fn)
	if errors.Is(err, ims.ErrNoURL) {
		return fmt.Errorf("no IMS gateway URL configured (%s)", cf.Join("ims-gateway", "url"))
	}
	return err
}
