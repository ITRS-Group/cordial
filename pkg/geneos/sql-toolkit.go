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

import (
	"encoding/xml"
	"fmt"
	"io"
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
		return fmt.Sprintf("oracle:%s", d.Oracle.DBName)
	}

	return "unsupported"
}

type MySQL struct {
	ServerName *SingleLineStringVar `xml:"var-serverName,omitempty"`
	DBName     *SingleLineStringVar `xml:"var-databaseName,omitempty"`
	Port       *SingleLineStringVar `xml:"var-port,omitempty"`
}

var _ xml.Unmarshaler = (*MySQL)(nil)

// UnmarshalXML deals with the case where merged XML configs have the
// "var-" prefix of the tags removed
func (v *MySQL) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	if v == nil {
		v = &MySQL{}
	}

	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		element, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		switch element.Name.Local {
		case "var-serverName", "serverName":
			s := &SingleLineStringVar{}
			err = d.DecodeElement(&s, &element)
			if err != nil {
				return err
			}
			v.ServerName = s
		case "var-databaseName", "databaseName":
			s := &SingleLineStringVar{}
			err = d.DecodeElement(&s, &element)
			if err != nil {
				return err
			}
			v.DBName = s
		case "var-port", "port":
			s := &SingleLineStringVar{}
			err = d.DecodeElement(&s, &element)
			if err != nil {
				return err
			}
			v.Port = s
		}

	}
}

type SQLServer struct {
	ServerName *SingleLineStringVar `xml:"var-serverName,omitempty"`
	DBName     *SingleLineStringVar `xml:"var-databaseName,omitempty"`
	Port       *SingleLineStringVar `xml:"var-port,omitempty"`
}

var _ xml.Unmarshaler = (*SQLServer)(nil)

// UnmarshalXML deals with the case where merged XML configs have the
// "var-" prefix of the tags removed
func (v *SQLServer) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	if v == nil {
		v = &SQLServer{}
	}

	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		element, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		switch element.Name.Local {
		case "var-serverName", "serverName":
			s := &SingleLineStringVar{}
			err = d.DecodeElement(&s, &element)
			if err != nil {
				return err
			}
			v.ServerName = s
		case "var-databaseName", "databaseName":
			s := &SingleLineStringVar{}
			err = d.DecodeElement(&s, &element)
			if err != nil {
				return err
			}
			v.DBName = s
		case "var-port", "port":
			s := &SingleLineStringVar{}
			err = d.DecodeElement(&s, &element)
			if err != nil {
				return err
			}
			v.Port = s
		}
	}
}

type Oracle struct {
	DBName          *SingleLineStringVar `xml:"var-databaseName,omitempty"`
	ApplicationName *SingleLineStringVar `xml:"var-applicationName,omitempty"`
}

var _ xml.Unmarshaler = (*Oracle)(nil)

// UnmarshalXML deals with the case where merged XML configs have the
// "var-" prefix of the tags removed
func (v *Oracle) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	if v == nil {
		v = &Oracle{}
	}

	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		element, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		switch element.Name.Local {
		case "var-applicationName", "applicationName":
			s := &SingleLineStringVar{}
			err = d.DecodeElement(&s, &element)
			if err != nil {
				return err
			}
			v.ApplicationName = s
		case "var-databaseName", "databaseName":
			s := &SingleLineStringVar{}
			err = d.DecodeElement(&s, &element)
			if err != nil {
				return err
			}
			v.DBName = s
		}
	}
}

type Sybase struct {
	InterfaceEntry  *SingleLineStringVar `xml:"var-interfaceEntry,omitempty"`
	DBName          *SingleLineStringVar `xml:"var-databaseName,omitempty"`
	ApplicationName *SingleLineStringVar `xml:"var-applicationName,omitempty"`
}

var _ xml.Unmarshaler = (*Sybase)(nil)

// UnmarshalXML deals with the case where merged XML configs have the
// "var-" prefix of the tags removed
func (v *Sybase) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	if v == nil {
		v = &Sybase{}
	}

	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		element, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		switch element.Name.Local {
		case "var-applicationName", "applicationName":
			s := &SingleLineStringVar{}
			err = d.DecodeElement(&s, &element)
			if err != nil {
				return err
			}
			v.ApplicationName = s
		case "var-databaseName", "databaseName":
			s := &SingleLineStringVar{}
			err = d.DecodeElement(&s, &element)
			if err != nil {
				return err
			}
			v.DBName = s
		}
	}
}
