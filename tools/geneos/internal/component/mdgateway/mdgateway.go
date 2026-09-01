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

package mdgateway

import "github.com/itrs-group/cordial/tools/geneos/internal/component/appgw"

// MDGateway is the registered md-gateway component type.
var MDGateway = appgw.Register(appgw.Spec{
	Name:      "md-gateway",
	Aliases:   []string{"mdgateway", "mdgw"},
	Jar:       "lib/md-gateway.jar",
	MainClass: "com.itrsgroup.mdgateway.Main",
	SetupFile: "md-gateway.yaml",
	LogFile:   "md-gateway.log",
	Ports:     "18000-",
})
