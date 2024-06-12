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

package netprobe

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos"
)

// These types represent those found in a netprobe setup file (SAN or
// floating) and not a probe config in the gateway

type Netprobe struct {
	XMLName          xml.Name          `xml:"netprobe"`
	Compatibility    int               `xml:"compatibility,attr"`                 // 1
	XMLNs            string            `xml:"xmlns:xsi,attr"`                     // http://www.w3.org/2001/XMLSchema-instance
	XSI              string            `xml:"xsi:noNamespaceSchemaLocation,attr"` // http://schema.itrsgroup.com/GA5.12.0-220125/netprobe.xsd
	FloatingNetprobe *FloatingNetprobe `xml:"floatingProbe,omitempty"`
	PluginWhiteList  []string          `xml:"pluginWhiteList,omitempty"`
	CommandWhiteList []string          `xml:"commandWhiteList,omitempty"`
	SelfAnnounce     *SelfAnnounce     `xml:"selfAnnounce,omitempty"`
}

type FloatingNetprobe struct {
	Enabled                  bool      `xml:"enabled"`
	RetryInterval            int       `xml:"retryInterval,omitempty"`
	RequireReverseConnection bool      `xml:"requireReverseConnection,omitempty"`
	ProbeName                string    `xml:"probeName"`
	Gateways                 []Gateway `xml:"gateways>gateway"`
}

type SelfAnnounce struct {
	Enabled                  bool             `xml:"enabled"`
	RetryInterval            int              `xml:"retryInterval,omitempty"`
	RequireReverseConnection bool             `xml:"requireReverseConnection,omitempty"`
	ProbeName                string           `xml:"probeName"`
	EncodedPassword          string           `xml:"encodedPassword,omitempty"`
	RESTAPIHTTPPort          int              `xml:"restApiHttpPort,omitempty"`
	RESTAPIHTTPSPort         int              `xml:"restApiHttpsPort,omitempty"`
	CyberArkApplicationID    string           `xml:"cyberArkApplicationID,omitempty"`
	CyberArkSDKPath          string           `xml:"cyberArkSdkPath,omitempty"`
	ManagedEntity            *ManagedEntity   `xml:"managedEntity,omitempty"`
	ManagedEntities          []ManagedEntity  `xml:"managedEntities>managedEntity,omitempty"`
	CollectionAgent          *CollectionAgent `xml:"collectionAgent,omitempty"`
	DynamicEntities          *DynamicEntities `xml:"dynamicEntities,omitempty"`
	Gateways                 []Gateway        `xml:"gateways>gateway"`
}

type Gateway struct {
	XMLName  xml.Name `xml:"gateway"`
	Hostname string   `xml:"hostname"`
	Port     int      `xml:"port,omitempty"`
	Secure   bool     `xml:"secure,omitempty"`
}

type ManagedEntity struct {
	XMLName    xml.Name    `xml:"managedEntity"`
	Name       string      `xml:"name"`
	Attributes *Attributes `xml:"attributes,omitempty"`
	Vars       *Vars       `xml:"variables,omitempty"`
	Types      *Types      `xml:"types,omitempty"`
}

type Attributes struct {
	XMLName    xml.Name `xml:"attributes"`
	Attributes []geneos.Attribute
}

type Vars struct {
	XMLName xml.Name `xml:"variables"`
	Vars    []geneos.Vars
}

type Types struct {
	XMLName xml.Name `xml:"types"`
	Types   []string `xml:"type"`
}

// type Type struct {
// 	Type string `xml:"type"`
// }

type CollectionAgent struct {
	Start        bool   `xml:"start,omitempty"`
	JVMArgs      string `xml:"jvmArgs,omitempty"`
	HealthPort   int    `xml:"healthPort,omitempty"`
	ReporterPort int    `xml:"reporterPort,omitempty"`
	Detached     bool   `xml:"detached"`
}

type DynamicEntities struct {
	MappingType []string `xml:"mappingType,omitempty"`
}
