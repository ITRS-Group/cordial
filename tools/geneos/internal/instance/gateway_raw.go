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

package instance

import (
	"fmt"
	"strings"

	"github.com/itrs-group/cordial/pkg/config"
	geneosxml "github.com/itrs-group/cordial/pkg/geneos"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
)

// ReadGatewaySetupBytes returns the contents of the gateway setup XML
// file of the instance exactly as it is on the instance host, for
// callers that edit it as text.
func ReadGatewaySetupBytes(i geneos.Instance) ([]byte, error) {
	if setupFile := config.Get[string](i.Config(), "setup"); strings.Contains(setupFile, "://") {
		return nil, fmt.Errorf("cannot edit gateway setup URL %q", setupFile)
	}
	return i.Host().ReadFile(PathTo(i, "setup"))
}

// WriteGatewaySetupBytes saves the setup XML data, as it is, as the
// setup file of the instance on the instance host with a history backup.
// It does the same as WriteGatewaySetup, including keeping the mode of
// an existing file, but without serializing a parsed setup.
func WriteGatewaySetupBytes(i geneos.Instance, data []byte) (string, error) {
	if setupFile := config.Get[string](i.Config(), "setup"); strings.Contains(setupFile, "://") {
		return "", fmt.Errorf("cannot write gateway setup URL %q", setupFile)
	}
	return geneosxml.WriteGatewayData(PathTo(i, "setup"), data, geneosxml.WithGatewayWriteHost(i.Host()), geneosxml.WithGatewayBackup())
}
