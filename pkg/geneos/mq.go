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
