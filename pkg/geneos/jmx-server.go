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

package geneos

import "encoding/xml"

// JMX-Server

type JMXServerPlugin struct {
	XMLName    xml.Name      `xml:"jmx-server" json:"-" yaml:"-"`
	Connection JMXConnection `xml:"connection"`
}

func (p *JMXServerPlugin) String() string {
	return p.XMLName.Local
}

type JMXConnection struct {
	Generic   *JMXGenericConnection
	WebLogic  *JMXWebLogicConnection
	WebSphere *JMXWebSphereConnection
}

type JMXGenericConnection struct {
	ServiceURL        *SingleLineStringVar
	ConnectionDetails *JMXGenericConnectionDetails
	JMXId             *SingleLineStringVar
	Username          *SingleLineStringVar
	Password          *Value
	Timeout           *Value
}

type JMXGenericConnectionDetails struct {
	Host    *SingleLineStringVar
	Port    *Value
	URLPath *SingleLineStringVar
}

type JMXWebLogicConnection struct {
	URL         *SingleLineStringVar
	Principal   *SingleLineStringVar
	Credentials *Value
	Certificate *SingleLineStringVar
	Key         *SingleLineStringVar
	Password    *Value
}

type JMXWebSphereConnection struct {
	Host                        *SingleLineStringVar
	Port                        *Value
	Username                    *SingleLineStringVar
	Password                    *Value
	Type                        *SingleLineStringVar
	SSLClientKeyStore           *SingleLineStringVar
	SSLClientKeyStorePassword   *Value
	SSLClientTrustStore         *SingleLineStringVar
	SSLClientTrustStorePassword *Value
}
