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

// SQL Toolkit

type SQLToolkitPlugin struct {
	XMLName    xml.Name     `xml:"sql-toolkit" json:"-" yaml:"-"`
	Queries    []Query      `xml:"queries>query"`
	Connection DBConnection `xml:"connection"`
}

func (p *SQLToolkitPlugin) String() string {
	return p.XMLName.Local
}

type Query struct {
	Name *SingleLineStringVar `xml:"name"`
	SQL  *SingleLineStringVar `xml:"sql"`
}

type DBConnection struct {
	MySQL                     *MySQL               `xml:"database>mysql,omitempty"`
	SQLServer                 *SQLServer           `xml:"database>sqlServer,omitempty"`
	Sybase                    *Sybase              `xml:"database>sybase,omitempty"`
	Username                  *SingleLineStringVar `xml:"var-userName"`
	Password                  *SingleLineStringVar `xml:"password"`
	CloseConnectionAfterQuery *Value               `xml:"closeConnectionAfterQuery,omitempty"`
}

type MySQL struct {
	ServerName *SingleLineStringVar `xml:"var-serverName"`
	DBName     *SingleLineStringVar `xml:"var-databaseName"`
	Port       *SingleLineStringVar `xml:"var-port"`
}

type SQLServer struct {
	ServerName *SingleLineStringVar `xml:"var-serverName"`
	DBName     *SingleLineStringVar `xml:"var-databaseName"`
	Port       *SingleLineStringVar `xml:"var-port"`
}

type Sybase struct {
	InstanceName *SingleLineStringVar `xml:"var-instanceName"`
	DBName       *SingleLineStringVar `xml:"var-databaseName"`
}
