package geneos

import "encoding/xml"

type MQChannelPlugin struct {
	XMLName      xml.Name      `xml:"mq-channel" json:"-" yaml:"-"`
	Connection   *MQConnection `xml:"connection,omitempty"`
	QueueManager *Value        `xml:"queueManager"`
	Channels     []MQChannel   `xml:"channels>channel,omitempty"`
	Columns      []string      `xml:"columns>column,omitempty"`
}

func (p *MQChannelPlugin) String() string {
	return p.XMLName.Local
}

type MQQInfoPlugin struct {
	XMLName         xml.Name             `xml:"mq-qinfo" json:"-" yaml:"-"`
	Queuemanager    *Value               `xml:"queueManager"`
	MQServer        *SingleLineStringVar `xml:"mqServer,omitempty"`
	QueueName       *SingleLineStringVar `xml:"queueName"`
	HideUnavailable *SingleLineStringVar `xml:"hideUnavailable,omitempty"`
}

func (p *MQQInfoPlugin) String() string {
	return p.XMLName.Local
}

type MQQueuePlugin struct {
	XMLName      xml.Name      `xml:"mq-queue" json:"-" yaml:"-"`
	Connection   *MQConnection `xml:"connection,omitempty"`
	QueueManager *Value        `xml:"queueManager"`
	Queues       []MQQueue     `xml:"queues>queue,omitempty"`
	Columns      []string      `xml:"columns>column,omitempty"`
}

func (p *MQQueuePlugin) String() string {
	return p.XMLName.Local
}

type MQConnection struct {
	MQServer       *SingleLineStringVar `xml:"mqServer,omitempty"`
	MQChannelTable *SingleLineStringVar `xml:"mqChannelTable,omitempty"`
}

type MQChannel struct {
	Matches    *SingleLineStringVar `xml:"matches,omitempty"`
	StartsWith *SingleLineStringVar `xml:"startsWith,omitempty"`
}

type MQQueue struct {
	Matches    *SingleLineStringVar `xml:"matches,omitempty"`
	StartsWith *SingleLineStringVar `xml:"startsWith,omitempty"`
}
