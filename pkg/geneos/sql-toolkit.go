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

import (
	"encoding/xml"
	"fmt"
)

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
	Oracle                    *Oracle              `xml:"database>oracle,omitempty"`
	Username                  *SingleLineStringVar `xml:"var-userName"`
	Password                  *SingleLineStringVar `xml:"password"`
	CloseConnectionAfterQuery *Value               `xml:"closeConnectionAfterQuery,omitempty"`
}

func (d *DBConnection) String() string {
	if d.MySQL != nil {
		port := d.MySQL.Port.String()
		if port == "" {
			port = "3306"
		}
		return fmt.Sprintf("mysql://%s:%s/%s", d.MySQL.ServerName, port, d.MySQL.DBName)
	}

	if d.SQLServer != nil {
		port := d.SQLServer.Port.String()
		if port == "" {
			port = "1433"
		}
		return fmt.Sprintf("sqlserver://%s:%s/%s", d.SQLServer.ServerName, port, d.SQLServer.DBName)
	}

	if d.Sybase != nil {
		return fmt.Sprintf("sybase:%s/%s", d.Sybase.InterfaceEntry, d.Sybase.DBName)
	}

	if d.Oracle != nil {
		if d.Oracle.DBName != nil {
			return fmt.Sprintf("oracle:%s", d.Oracle.DBName)
		}
		return fmt.Sprintf("oracle:%s", d.Oracle.DBNameAlt)
	}

	return "unsupported"
}

type MySQL struct {
	ServerName *SingleLineStringVar `xml:"var-serverName,omitempty"`
	DBName     *SingleLineStringVar `xml:"var-databaseName,omitempty"`
	Port       *SingleLineStringVar `xml:"var-port,omitempty"`
}

type SQLServer struct {
	ServerName *SingleLineStringVar `xml:"var-serverName,omitempty"`
	DBName     *SingleLineStringVar `xml:"var-databaseName,omitempty"`
	Port       *SingleLineStringVar `xml:"var-port,omitempty"`
}

type Oracle struct {
	DBName          *SingleLineStringVar `xml:"var-databaseName,omitempty"`
	DBNameAlt       *SingleLineStringVar `xml:"databaseName,omitempty"`
	ApplicationName *SingleLineStringVar `xml:"var-applicationName,omitempty"`
}

type Sybase struct {
	InterfaceEntry  *SingleLineStringVar `xml:"var-interfaceEntry,omitempty"`
	DBName          *SingleLineStringVar `xml:"var-databaseName,omitempty"`
	ApplicationName *SingleLineStringVar `xml:"var-applicationName,omitempty"`
}
