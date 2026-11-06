package jmxserver

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type Plugin struct {
	XMLName      xml.Name                     `xml:"jmx-server" json:"-" yaml:"-"`
	Connection   Connection                   `xml:"connection"`
	Aliases      []Alias                      `xml:"aliases>alias,omitempty"`
	Summary      *Summary                     `xml:"summary,omitempty"`
	ShowCommands *ShowCommands                `xml:"showCommands,omitempty"`
	Headlines    []Headline                   `xml:"headlines>headline,omitempty"`
	JProperties  []common.SingleLineStringVar `xml:"jproperties>jproperty,omitempty"`
	IDAttributes []common.SingleLineStringVar `xml:"idAttributes>idAttribute,omitempty"`
	Exclusions   []common.SingleLineStringVar `xml:"exclusionlist>exclusion,omitempty"`
	RowLimit     *common.Value                `xml:"rowLimit,omitempty"`
	Private      *Private                     `xml:"private,omitempty"`
}

func (p *Plugin) String() string { return p.XMLName.Local }

type Alias struct {
	Name          string                      `xml:"name"`
	Value         *common.SingleLineStringVar `xml:"value"`
	AttributePath *common.SingleLineStringVar `xml:"attributePath,omitempty"`
}

type Summary struct {
	Enable   *common.Value               `xml:"enable"`
	Grouping *common.SingleLineStringVar `xml:"grouping,omitempty"`
}

type ShowCommands struct {
	Headlines    *bool `xml:"headlines,omitempty"`
	TableColumns *bool `xml:"tableColumns,omitempty"`
	Dataviews    *bool `xml:"dataviews,omitempty"`
	Samplers     *bool `xml:"samplers,omitempty"`
}

type Headline struct {
	Name  string                      `xml:"name"`
	Value *common.SingleLineStringVar `xml:"value"`
	Mask  *common.SingleLineStringVar `xml:"mask,omitempty"`
}

type Private struct {
	LogJavaMemoryInfo *bool `xml:"logJavaMemoryInfo,omitempty"`
}

type Connection struct {
	Generic   *GenericConnection   `xml:"generic,omitempty"`
	WebLogic  *WebLogicConnection  `xml:"weblogic,omitempty"`
	WebSphere *WebSphereConnection `xml:"websphere,omitempty"`
}

type GenericConnection struct {
	ServiceURL        *common.SingleLineStringVar `xml:"serviceURL,omitempty"`
	ConnectionDetails *GenericConnectionDetails   `xml:"connectionDetails,omitempty"`
	JMXId             *common.SingleLineStringVar `xml:"jmxId,omitempty"`
	Username          *common.SingleLineStringVar `xml:"username,omitempty"`
	Password          *common.Value               `xml:"password,omitempty"`
	Timeout           *common.Value               `xml:"timeout,omitempty"`
}

type GenericConnectionDetails struct {
	Host    *common.SingleLineStringVar `xml:"host,omitempty"`
	Port    *common.Value               `xml:"port,omitempty"`
	URLPath *common.SingleLineStringVar `xml:"urlPath,omitempty"`
}

type WebLogicConnection struct {
	URL         *common.SingleLineStringVar `xml:"java.naming.provider.url,omitempty"`
	Principal   *common.SingleLineStringVar `xml:"java.naming.security.principal,omitempty"`
	Credentials *common.Value               `xml:"java.naming.security.credentials,omitempty"`
	Certificate *common.SingleLineStringVar `xml:"SSLClientCertificateFile,omitempty"`
	Key         *common.SingleLineStringVar `xml:"SSLClientCertificateKeyFile,omitempty"`
	Password    *common.Value               `xml:"password,omitempty"`
}

type WebSphereConnection struct {
	Host                        *common.SingleLineStringVar `xml:"host,omitempty"`
	Port                        *common.Value               `xml:"port,omitempty"`
	Username                    *common.SingleLineStringVar `xml:"username,omitempty"`
	Password                    *common.Value               `xml:"password,omitempty"`
	Type                        *common.SingleLineStringVar `xml:"type,omitempty"`
	SSLClientKeyStore           *common.SingleLineStringVar `xml:"sslKeyStoreDetails>SSLClientKeyStore,omitempty"`
	SSLClientKeyStorePassword   *common.Value               `xml:"sslKeyStoreDetails>password,omitempty"`
	SSLClientTrustStore         *common.SingleLineStringVar `xml:"sslTrustStoreDetails>SSLClientTrustStore,omitempty"`
	SSLClientTrustStorePassword *common.Value               `xml:"sslTrustStoreDetails>password,omitempty"`
}
