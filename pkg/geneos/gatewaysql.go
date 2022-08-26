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

// Gateway-SQL

type GatewaySQLPlugin struct {
	Setup  *SingleLineString `xml:"Gateway-sql>setupSql>sql"`
	Tables *GatewaySQLTables `xml:"Gateway-sql>tables"`
	Views  []GWSQLView       `xml:"Gateway-sql>views>view"`
}

type GatewaySQLTables struct {
	Tables []interface{}
}

type GatewaySQLTableDataview struct {
	XMLName xml.Name          `xml:"dataview"`
	Name    *SingleLineString `xml:"tableName"`
	XPath   string            `xml:"xpath"`
	Columns *[]GWSQLColumn    `xml:"columns>column,omitempty"`
}

type GatewaySQLTableHeadline struct {
	XMLName xml.Name          `xml:"headlines"`
	Name    *SingleLineString `xml:"tableName"`
	XPath   string            `xml:"xpath"`
}

type GatewaySQLTableXPath struct {
	XMLName xml.Name          `xml:"xpath"`
	Name    *SingleLineString `xml:"tableName"`
	XPaths  []string          `xml:"xpaths>xpath"`
	Columns []GWSQLColumn     `xml:"columns>column"`
}

type GWSQLColumn struct {
	Name  *SingleLineString `xml:"name"`
	XPath string            `xml:"xpath,omitempty"`
	Type  string            `xml:"type,omitempty"`
}

type GWSQLView struct {
	XMLName  xml.Name          `xml:"view"`
	ViewName *SingleLineString `xml:"name"`
	SQL      *SingleLineString `xml:"sql"`
}
