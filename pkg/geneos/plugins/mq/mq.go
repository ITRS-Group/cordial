package mq

import (
	"encoding/xml"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type MQChannelPlugin struct {
	XMLName       xml.Name        `xml:"mq-channel" json:"-" yaml:"-"`
	Connection    *MQConnection   `xml:"connection,omitempty"`
	QueueManager  *common.Value   `xml:"queueManager"`
	Channels      []MQChannel     `xml:"channels>channel,omitempty"`
	Columns       []string        `xml:"columns>column,omitempty"`
	DisplayFormat *string         `xml:"displayFormat,omitempty"`
	Debug         *MQChannelDebug `xml:"debug,omitempty"`
}

func (p *MQChannelPlugin) String() string { return p.XMLName.Local }

type MQQInfoPlugin struct {
	XMLName                                xml.Name                    `xml:"mq-qinfo" json:"-" yaml:"-"`
	Queuemanager                           *common.Value               `xml:"queueManagerName"`
	MQServer                               *common.SingleLineStringVar `xml:"mqServer,omitempty"`
	QueueName                              *common.SingleLineStringVar `xml:"queueName,omitempty"`
	HideUnavailable                        *common.Value               `xml:"hideUnavailable,omitempty"`
	StagnantMessageAge                     *common.Value               `xml:"stagnantMessageAge,omitempty"`
	StatsCalcTimeOut                       *int                        `xml:"statsCalcTimeOut,omitempty"`
	SecurityExitConfig                     *MQSecurityExitConfig       `xml:"securityExitConfig,omitempty"`
	SecurityConnectionAuthenticationConfig *MQSecurityConnAuth         `xml:"securityConnectionAuthenticationConfig,omitempty"`
	Private                                *MQPrivate                  `xml:"private,omitempty"`
}

func (p *MQQInfoPlugin) String() string { return p.XMLName.Local }

type MQQueuePlugin struct {
	XMLName                      xml.Name      `xml:"mq-queue" json:"-" yaml:"-"`
	Connection                   *MQConnection `xml:"connection,omitempty"`
	QueueManager                 *common.Value `xml:"queueManager"`
	Queues                       []MQQueue     `xml:"queues>queue,omitempty"`
	Columns                      []string      `xml:"columns>column,omitempty"`
	DisableQueueResetAttributes  *common.Value `xml:"disableQueueResetAttributes,omitempty"`
	DisableQueueBrowseAttributes *common.Value `xml:"disableQueueBrowseAttributes,omitempty"`
	StagnantMessageAge           *common.Value `xml:"stagnantMessageAge,omitempty"`
	Debug                        *MQQueueDebug `xml:"debug,omitempty"`
}

func (p *MQQueuePlugin) String() string { return p.XMLName.Local }

type MQConnection struct {
	MQServer                               *common.SingleLineStringVar `xml:"mqServer,omitempty"`
	MQChannelTable                         *common.SingleLineStringVar `xml:"mqChannelTable,omitempty"`
	SSL                                    *MQSSL                      `xml:"ssl,omitempty"`
	SecurityExitConfig                     *MQSecurityExitConfig       `xml:"securityExitConfig,omitempty"`
	SecurityConnectionAuthenticationConfig *MQSecurityConnAuth         `xml:"securityConnectionAuthenticationConfig,omitempty"`
}

type MQSSL struct {
	KeyRepository *common.SingleLineStringVar `xml:"keyRepository,omitempty"`
	CipherSpec    *common.SingleLineStringVar `xml:"cipherSpec,omitempty"`
	MQSender      *common.Value               `xml:"mqSender,omitempty"`
	MQReceiver    *common.Value               `xml:"mqReceiver,omitempty"`
}

type MQSecurityExitConfig struct {
	SecurityExit         *common.SingleLineStringVar `xml:"securityExit"`
	RemoteUserIdentifier *common.SingleLineStringVar `xml:"remoteUserIdentifier,omitempty"`
	RemotePassword       *common.Value               `xml:"remotePassword,omitempty"`
}

type MQSecurityConnAuth struct {
	Username *common.SingleLineStringVar `xml:"username,omitempty"`
	Password *common.Value               `xml:"password,omitempty"`
}

type MQChannelDebug struct {
	SampleTimeout *int                  `xml:"sampleTimeout,omitempty"`
	Output        *MQChannelDebugOutput `xml:"output,omitempty"`
}

type MQChannelDebugOutput struct {
	Setup      *bool `xml:"setup,omitempty"`
	MQAPI      *bool `xml:"mqApi,omitempty"`
	Connection *bool `xml:"connection,omitempty"`
	Commands   *bool `xml:"commands,omitempty"`
	Results    *bool `xml:"results,omitempty"`
	Summary    *bool `xml:"summary,omitempty"`
}

type MQQueueDebug struct {
	SampleTimeout           *int                `xml:"sampleTimeout,omitempty"`
	DisableBrowseTimeAdjust *bool               `xml:"disableBrowseTimeAdjust,omitempty"`
	BrowseWarnThreshold     *int                `xml:"browseWarnThreshold,omitempty"`
	Output                  *MQQueueDebugOutput `xml:"output,omitempty"`
}

type MQQueueDebugOutput struct {
	Setup      *bool `xml:"setup,omitempty"`
	MQAPI      *bool `xml:"mqApi,omitempty"`
	Connection *bool `xml:"connection,omitempty"`
	Commands   *bool `xml:"commands,omitempty"`
	Results    *bool `xml:"results,omitempty"`
	Browsing   *bool `xml:"browsing,omitempty"`
}

type MQPrivate struct {
	Debug *common.Value `xml:"debug,omitempty"`
}

type MQChannel struct {
	Matches    *common.SingleLineStringVar `xml:"matches,omitempty"`
	StartsWith *common.SingleLineStringVar `xml:"startsWith,omitempty"`
}

type MQQueue struct {
	Matches    *common.SingleLineStringVar `xml:"matches,omitempty"`
	StartsWith *common.SingleLineStringVar `xml:"startsWith,omitempty"`
}
