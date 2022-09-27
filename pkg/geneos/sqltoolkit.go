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

// SQL Toolkit

type SQLToolkitPlugin struct {
	Queries    []Query      `xml:"sql-toolkit>queries>query"`
	Connection DBConnection `xml:"sql-toolkit>connection"`
}

type Query struct {
	Name *SingleLineString `xml:"name"`
	SQL  *SingleLineString `xml:"sql"`
}

type DBConnection struct {
	MySQL                     *MySQL            `xml:"database>mysql,omitempty"`
	SQLServer                 *SQLServer        `xml:"database>sqlServer,omitempty"`
	Sybase                    *Sybase           `xml:"database>sybase,omitempty"`
	Username                  *SingleLineString `xml:"var-userName"`
	Password                  *SingleLineString `xml:"password"`
	CloseConnectionAfterQuery *Value            `xml:"closeConnectionAfterQuery,omitempty"`
}

type MySQL struct {
	ServerName *SingleLineString `xml:"var-serverName"`
	DBName     *SingleLineString `xml:"var-databaseName"`
	Port       *SingleLineString `xml:"var-port"`
}

type SQLServer struct {
	ServerName *SingleLineString `xml:"var-serverName"`
	DBName     *SingleLineString `xml:"var-databaseName"`
	Port       *SingleLineString `xml:"var-port"`
}

type Sybase struct {
	InstanceName *SingleLineString `xml:"var-instanceName"`
	DBName       *SingleLineString `xml:"var-databaseName"`
}
