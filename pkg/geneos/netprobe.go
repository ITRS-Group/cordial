package geneos

import "encoding/xml"

type Netprobe struct {
	XMLName          xml.Name       `xml:"netprobe"`
	Compatibility    int            `xml:"compatibility,attr"`                 // 1
	XMLNs            string         `xml:"xmlns:xsi,attr"`                     // http://www.w3.org/2001/XMLSchema-instance
	XSI              string         `xml:"xsi:noNamespaceSchemaLocation,attr"` // http://schema.itrsgroup.com/GA5.12.0-220125/netprobe.xsd
	FloatingProbe    *FloatingProbe `xml:"floatingProbe,omitempty"`
	PluginWhiteList  []string       `xml:"pluginWhiteList,omitempty"`
	CommandWhiteList []string       `xml:"commandWhiteList,omitempty"`
	SelfAnnounce     *SelfAnnounce  `xml:"selfAnnounce,omitempty"`
}

type FloatingProbe struct {
	Enabled                  bool       `xml:"enabled"`
	RetryInterval            int        `xml:"retryInterval,omitempty"`
	RequireReverseConnection bool       `xml:"requireReverseConnection,omitempty"`
	ProbeName                string     `xml:"probeName"`
	Gateways                 []Gateways `xml:"gateways"`
}

type Gateways struct {
	XMLName  xml.Name `xml:"gateway"`
	Hostname string   `xml:"hostname"`
	Port     int      `xml:"port,omitempty"`
	Secure   bool     `xml:"secure,omitempty"`
}

type SelfAnnounce struct {
	Enabled                  bool              `xml:"enabled"`
	RetryInterval            int               `xml:"retryInterval,omitempty"`
	RequireReverseConnection bool              `xml:"requireReverseConnection,omitempty"`
	ProbeName                string            `xml:"probeName"`
	EncodedPassword          string            `xml:"encodedPassword,omitempty"`
	RESTAPIHTTPPort          int               `xml:"restApiHttpPort,omitempty"`
	RESTAPIHTTPSPort         int               `xml:"restApiHttpsPort,omitempty"`
	CyberArkApplicationID    string            `xml:"cyberArkApplicationID,omitempty"`
	CyberArkSDKPath          string            `xml:"cyberArkSdkPath,omitempty"`
	ManagedEntity            *SAManagedEntity  `xml:"managedEntity,omitempty"`
	ManagedEntities          []SAManagedEntity `xml:"managedEntities,omitempty"`
	CollectionAgent          *CollectionAgent  `xml:"collectionAgent,omitempty"`
	DynamicEntities          *DynamicEntities  `xml:"dynamicEntities,omitempty"`
	Gateways                 []Gateways        `xml:"gateways"`
}

type SAManagedEntity struct {
	XMLName    xml.Name    `xml:"managedEntity"`
	Name       string      `xml:"name"`
	Attributes []Attribute `xml:"attributes,omitempty"`
	Vars       []Vars      `xml:"variables,omitempty"`
	Types      []PlainType `xml:"types,omitempty"`
}

type PlainType struct {
	Type string `xml:"type"`
}

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
