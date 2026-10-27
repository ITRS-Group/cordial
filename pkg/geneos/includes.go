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

package geneos

import "encoding/xml"

// Includes is the top-level includes element
type Includes struct {
	XMLName       xml.Name       `xml:"includes"`
	Priority      int            `xml:"priority"`
	Reloading     *Reloading     `xml:"reloading,omitempty"`
	Includes      []Include      `xml:"include,omitempty"`
	IncludeGroups []IncludeGroup `xml:"includeGroup,omitempty"`
}

// Reloading defines reload settings for includes
type Reloading struct {
	XMLName         xml.Name `xml:"reloading"`
	ReloadInterval  *int     `xml:"reloadInterval,omitempty"`
	ActiveTime      string   `xml:"activeTime,omitempty"`
	ApplyToIncludes *bool    `xml:"applyToIncludes,omitempty"`
}

// IncludesGroup is the reusable group of include and includeGroup elements
type IncludesGroup struct {
	Includes      []Include      `xml:"include,omitempty"`
	IncludeGroups []IncludeGroup `xml:"includeGroup,omitempty"`
}

// Include is a single include of another setup file
type Include struct {
	XMLName        xml.Name               `xml:"include"`
	Disabled       bool                   `xml:"disabled,attr,omitempty"`
	Priority       int                    `xml:"priority"`
	Required       bool                   `xml:"required"`
	Location       string                 `xml:"location"`
	ReloadInterval *int                   `xml:"reloadInterval,omitempty"`
	ActiveTime     string                 `xml:"activeTime,omitempty"`
	Remote         *RemoteIncludeSettings `xml:"remote,omitempty"`
}

// IncludeGroup is a named group of includes
type IncludeGroup struct {
	XMLName       xml.Name               `xml:"includeGroup"`
	Name          string                 `xml:"name,attr"`
	Disabled      bool                   `xml:"disabled,attr,omitempty"`
	Includes      []Include              `xml:"include,omitempty"`
	IncludeGroups []IncludeGroup         `xml:"includeGroup,omitempty"`
	Remote        *RemoteIncludeSettings `xml:"remote,omitempty"`
}

// RemoteIncludeSettings is remoteIncludeSettingsType
type RemoteIncludeSettings struct {
	CertificateVerification *CertificateVerification `xml:"certificateVerification,omitempty"`
	Authentication          *IncludeAuthentication   `xml:"authentication,omitempty"`
	Proxy                   *IncludeProxy            `xml:"proxy,omitempty"`
	Timeout                 *int                     `xml:"timeout,omitempty"`
}

// CertificateVerification is a choice of one of its members
type CertificateVerification struct {
	None                        *EmptyType `xml:"none,omitempty"`
	AgainstAuthority            *CABundle  `xml:"againstAuthority,omitempty"`
	AgainstAuthorityAndHostname *CABundle  `xml:"againstAuthorityAndHostname,omitempty"`
}

// EmptyType is an empty element
type EmptyType struct{}

// CABundle holds a CA bundle
type CABundle struct {
	CABundle string `xml:"caBundle,omitempty"`
}

// IncludeAuthentication is a choice of none or password
type IncludeAuthentication struct {
	None     *EmptyType       `xml:"none,omitempty"`
	Password *IncludePassword `xml:"password,omitempty"`
}

// IncludePassword holds password authentication settings
type IncludePassword struct {
	Username string         `xml:"username,omitempty"`
	Password string         `xml:"password,omitempty"`
	Method   *IncludeMethod `xml:"method,omitempty"`
}

// IncludeMethod holds authentication methods
type IncludeMethod struct {
	Basic        *bool `xml:"basic,omitempty"`
	GSSNegotiate *bool `xml:"gssnegotiate,omitempty"`
}

// IncludeProxy is a choice of none or proxyServer
type IncludeProxy struct {
	None        *EmptyType `xml:"none,omitempty"`
	ProxyServer *struct {
		Hostname string `xml:"hostname,omitempty"`
		Port     *int   `xml:"port,omitempty"`
		Username string `xml:"username,omitempty"`
		Password string `xml:"password,omitempty"`
	} `xml:"proxyServer,omitempty"`
}
