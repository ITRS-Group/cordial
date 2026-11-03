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

// MQChannelPlugin TODO: Describe its purpose and intended use.
type MQChannelPlugin struct {
	XMLName      xml.Name      `xml:"mq-channel" json:"-" yaml:"-"`
	Connection   *MQConnection `xml:"connection,omitempty"`
	QueueManager *Value        `xml:"queueManager"`
	Channels     []MQChannel   `xml:"channels>channel,omitempty"`
	Columns      []string      `xml:"columns>column,omitempty"`
}

// String TODO: Document its behavior, inputs, and effects.
func (p *MQChannelPlugin) String() string {
	return p.XMLName.Local
}

// MQQInfoPlugin TODO: Describe its purpose and intended use.
type MQQInfoPlugin struct {
	XMLName         xml.Name             `xml:"mq-qinfo" json:"-" yaml:"-"`
	Queuemanager    *Value               `xml:"queueManager"`
	MQServer        *SingleLineStringVar `xml:"mqServer,omitempty"`
	QueueName       *SingleLineStringVar `xml:"queueName"`
	HideUnavailable *SingleLineStringVar `xml:"hideUnavailable,omitempty"`
}

// String TODO: Document its behavior, inputs, and effects.
func (p *MQQInfoPlugin) String() string {
	return p.XMLName.Local
}

// MQQueuePlugin TODO: Describe its purpose and intended use.
type MQQueuePlugin struct {
	XMLName      xml.Name      `xml:"mq-queue" json:"-" yaml:"-"`
	Connection   *MQConnection `xml:"connection,omitempty"`
	QueueManager *Value        `xml:"queueManager"`
	Queues       []MQQueue     `xml:"queues>queue,omitempty"`
	Columns      []string      `xml:"columns>column,omitempty"`
}

// String TODO: Document its behavior, inputs, and effects.
func (p *MQQueuePlugin) String() string {
	return p.XMLName.Local
}

// MQConnection TODO: Describe its purpose and intended use.
type MQConnection struct {
	MQServer       *SingleLineStringVar `xml:"mqServer,omitempty"`
	MQChannelTable *SingleLineStringVar `xml:"mqChannelTable,omitempty"`
}

// MQChannel TODO: Describe its purpose and intended use.
type MQChannel struct {
	Matches    *SingleLineStringVar `xml:"matches,omitempty"`
	StartsWith *SingleLineStringVar `xml:"startsWith,omitempty"`
}

// MQQueue TODO: Describe its purpose and intended use.
type MQQueue struct {
	Matches    *SingleLineStringVar `xml:"matches,omitempty"`
	StartsWith *SingleLineStringVar `xml:"startsWith,omitempty"`
}
