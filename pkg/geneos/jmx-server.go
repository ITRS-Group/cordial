/*
Copyright © 2022 ITRS Group

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
