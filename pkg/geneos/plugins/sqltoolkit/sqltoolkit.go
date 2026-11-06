package sqltoolkit

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/itrs-group/cordial/pkg/geneos/common"
)

type SingleLineStringVar = common.SingleLineStringVar
type Value = common.Value

type Plugin struct {
	XMLName    xml.Name     `xml:"sql-toolkit" json:"-" yaml:"-"`
	Queries    []Query      `xml:"queries>query"`
	Connection DBConnection `xml:"connection"`
}

func (p *Plugin) String() string { return p.XMLName.Local }

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

func (v *MySQL) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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
			value := &SingleLineStringVar{}
			if err := d.DecodeElement(value, &element); err != nil {
				return err
			}
			v.ServerName = value
		case "var-databaseName", "databaseName":
			value := &SingleLineStringVar{}
			if err := d.DecodeElement(value, &element); err != nil {
				return err
			}
			v.DBName = value
		case "var-port", "port":
			value := &SingleLineStringVar{}
			if err := d.DecodeElement(value, &element); err != nil {
				return err
			}
			v.Port = value
		}
	}
}

type SQLServer struct {
	ServerName *SingleLineStringVar `xml:"var-serverName,omitempty"`
	DBName     *SingleLineStringVar `xml:"var-databaseName,omitempty"`
	Port       *SingleLineStringVar `xml:"var-port,omitempty"`
}

var _ xml.Unmarshaler = (*SQLServer)(nil)

func (v *SQLServer) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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
			value := &SingleLineStringVar{}
			if err := d.DecodeElement(value, &element); err != nil {
				return err
			}
			v.ServerName = value
		case "var-databaseName", "databaseName":
			value := &SingleLineStringVar{}
			if err := d.DecodeElement(value, &element); err != nil {
				return err
			}
			v.DBName = value
		case "var-port", "port":
			value := &SingleLineStringVar{}
			if err := d.DecodeElement(value, &element); err != nil {
				return err
			}
			v.Port = value
		}
	}
}

type Oracle struct {
	DBName          *SingleLineStringVar `xml:"var-databaseName,omitempty"`
	ApplicationName *SingleLineStringVar `xml:"var-applicationName,omitempty"`
}

var _ xml.Unmarshaler = (*Oracle)(nil)

func (v *Oracle) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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
			value := &SingleLineStringVar{}
			if err := d.DecodeElement(value, &element); err != nil {
				return err
			}
			v.ApplicationName = value
		case "var-databaseName", "databaseName":
			value := &SingleLineStringVar{}
			if err := d.DecodeElement(value, &element); err != nil {
				return err
			}
			v.DBName = value
		}
	}
}

type Sybase struct {
	InterfaceEntry  *SingleLineStringVar `xml:"var-interfaceEntry,omitempty"`
	DBName          *SingleLineStringVar `xml:"var-databaseName,omitempty"`
	ApplicationName *SingleLineStringVar `xml:"var-applicationName,omitempty"`
}

var _ xml.Unmarshaler = (*Sybase)(nil)

func (v *Sybase) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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
			value := &SingleLineStringVar{}
			if err := d.DecodeElement(value, &element); err != nil {
				return err
			}
			v.ApplicationName = value
		case "var-databaseName", "databaseName":
			value := &SingleLineStringVar{}
			if err := d.DecodeElement(value, &element); err != nil {
				return err
			}
			v.DBName = value
		}
	}
}
