# XML-RPC API Go bindings

This set of packages provides a wrapper around low-level XML-RPC calls to send data to a Geneos Netprobe.

The code is still very much in the early stages of development and the API may change at any time. Feedback, via Issues, and Pull requests are welcome but without any guarentees if I'll have time to do them.

The package import path is rooted at vanity URL `wonderland.org/geneos` which currently resolves to https://github.com/pgalbavy/geneos 

The documentation for the underlying API is here:

https://docs.itrsgroup.com/docs/geneos/current/Netprobe/api/xml-rpc-api.html

While direct mappings from golang to the API are available in the [xmlrpc](geneos/pkg/xmlrpc) package most users will want to look at the higher-level [samplers](geneos/pkg/samplers) and [streams](geneos/pkg/streams) packages that try to implement an easier to use high-level interface.

## Examples of use

The [example](geneos/example) package directory contains a number of simple implmentations of common plugin types that show how to use the different types of data update methods.

The [example/generic](geneos/example/geneic) directory is described in further detail below. It uses this method to deliver updates:

```go
func (s Samplers) UpdateTableFromSlice(rowdata interface{}) error
```

The other two methods both take maps as follows:

```go
func (s *Samplers) UpdateTableFromMap(data interface{}) error
```
```go
func (s *Samplers) UpdateTableFromMapDelta(newdata, olddata interface{}, interval time.Duration) error
```

The `UpdateTableFromMapDelta()` also takes an `time.Duration` interval that allows scaling of the difference between the two datasets. 

## Create a basic plugin

First, import the necessary packages

```go
package generic

import (
	"wonderland.org/geneos/pkg/plugins"
	"wonderland.org/geneos/pkg/sampler"
)
```

Next, create two structs, one to hold the per-sample data and another to hold the Sampler amd any other local data that is needed for the lifetime of the sampler:

```go
type GenericData struct {
	RowName string
	Column1 string
	Column2 string
}

type GenericSampler struct {
	samplers.Samplers
	localdata string
}
```

Now create the required methods. There are three and they must meet this interface (from the samplers package):

```go
type SamplerInstance interface {
	New(plugins.Connection, string, string) *SamplerInstance
	InitSampler(*SamplerInstance) (err error)
	DoSample(*SamplerInstance) (err error)
}
```

First a `New()` method that your main package will call to create an instance of the plugin - aka. the sampler - and do some housekeeping:

```go
func New(s plugins.Connection, name string, group string) (*GenericSampler, error) {
	c := new(GenericSampler)
	c.Plugins = c
	return c, c.New(s, name, group)
}
```

As an aside, this New() method has to work this way because it's how I have found to make the underlying _type_ of Plugins take on that of the specific plugin package. Along with some internals this is how the plugin exposes it's own methods in the above interface correctly, without infinite recursion. 

The next method is `InitSampler()` which is called once upon start-up of the sampler instance. The first part of this example locates a parameter in the Geneos configurationa and assigns is to the local data struct.

```go
func (g *GenericSampler) InitSampler() error {
	example, err := g.Parameter("EXAMPLE")
	if err != nil {
		return nil
	}
	g.localdata = example

```

It is worth noting at this point that the `InitSampler()` being called only once means that if there is any change in the Geneos configuration there is no way for the running program to notice. The XML-RPC API is stateless (we'll ignore the heartbeat functions for now) and these plugins may not notice a Netprobe or related restart. So, the `Parameter()` call above is only an example and should probably be refreshed using a timer, but not every sample most likely.   

The second part is required to initialise the helper methods which we'll used see below:

```go

	columns, columnnames, sortcol, err := g.ColumnInfo(GenericData{})
	g.SetColumns(columns)
	g.SetColumnNames(columnnames)
	g.SetSortColumn(sortcol)
	return g.Headline("example", g.localdata)
}
```

The final mandatory method is `DoSample()` which is called to update the data:

```go
func (p *GenericSampler) DoSample() error {
	var rowdata = []GenericData{
		{"row4", "data1", "data2"},
		{"row2", "data1", "data2"},
		{"row3", "data1", "data2"},
		{"row1", "data1", "data2"},
	}
	return p.UpdateTableFromSlice(rowdata)
}
```

The call to `UpdateTableFromSlice()` uses the column data initialised earlier to ensure the dataview is rendered correctly.

## More features

You can use tags to control the rendering of the data, like this example of a [CPU plugin for Windows](geneos/example/cpu/cpu_windows.go):

```go
// +build windows
package cpu

import (
	"log"
	"time"

	"github.com/StackExchange/wmi"
	"wonderland.org/geneos/pkg/samplers"
)

// Win32_PerfRawData_PerfOS_Processor must be exported along with all it's
// fields so that methods in plugins package can output the results
type Win32_PerfRawData_PerfOS_Processor struct {
	Name                  string `column:"cpuName"`
	PercentUserTime       uint64 `column:"% User Time,format=%.2f %%"`
	PercentPrivilegedTime uint64 `column:"% Priv Time,format=%.2f %%"`
	PercentIdleTime       uint64 `column:"% Idle Time,format=%.2f %%"`
	PercentProcessorTime  uint64 `column:"% Proc Time,format=%.2f %%"`
	PercentInterruptTime  uint64 `column:"% Intr Time,format=%.2f %%"`
	PercentDPCTime        uint64 `column:"% DPC Time,format=%.2f %%"`
	Timestamp_PerfTime    uint64 `column:"OMIT"`
	Frequency_PerfTime    uint64 `column:"OMIT"`
}

// one entry for each CPU row in /proc/stats
type cpustat struct {
	cpus       map[string]Win32_PerfRawData_PerfOS_Processor
	lastsample float64
	frequency  float64
}

```

The tag is _column_ and the comma seperated tag values currently supported are:

* `name` - any value without an "=" is treated as a display name for the column created from this field. The special name "OMIT" means that the fields should not create a column, but the data will still be avilable for calculations etc. Any normal ASCII characters are permitted except a comma. No validation is done and the string is passed to the Netprobe as-is.
* `format=FORMAT` - FORMAT is a `Printf` style format string used to render the value of the cell in the most appropriate way for the data
* `sort=[+|-][num]` - the _sort_ tag defines which field - and only one field can be selected - should be used to sort the resulting rows published via the _Map_ rendering methods. The valid values are an option leading + or - representing ascending or descending order and the option suffix "num" to indicate a numeric sort. "sort=" means to sort ascending in lexographical order, which is the same as "sort=+"

The _sort_ tag only applies to those dataviews populated from maps like this call below:

```go
func (p *CPUSampler) DoSample() (err error) {
...
		err = p.UpdateTableFromMapDelta(stat.cpus, laststats.cpus, time.Duration(interval)*10*time.Millisecond)
```

The `UpdateTableFromSlice()` shown in the _generic_ example assumes that the slice has been passed in the order required. Maps on the other hand have no defined order and the package allows you to define the natural sort order. This can of course be overridden by the user of the Geneos Active Console.

## Initialise and start-up

To use your plugin in a program, use it like this:

```go
package main

import (
...
	"wonderland.org/geneos/pkg/plugins"
	"wonderland.org/geneos/pkg/streams"

	"example/generic"			// this will depend on how you name it
)
```

Do normal start-up configuration, process command line args etc. and then initialise the `Sampler` connection like this: 

```go
func main() {
...

	// connect to netprobe
	url := fmt.Sprintf("http://%s:%v/xmlrpc", hostname, port)
	s, err := plugins.Sampler(url, entityname, samplername)
	if err != nil {
		log.Fatal(err)
	}
```

Once you have your _sampler_ connection call the `New()` method with _dataview_ and _group_ names. The _group_ can be an empty string. Set the _interval_ as a Go `time.Duration` value. The default, if a zero is passed, is one second. One second is also the minimum interval.

Finally `Start()` the sampler by passing a `sync.WaitGroup` that you can later `Wait()` on so the program doesn't exit while the sampler runs.

```go
	g, err := generic.New(s, "example", "SYSTEM")
	defer g.Close()
	g.SetInterval(interval)
	g.Start(&wg)

	wg.Wait()
}

```

## Logging

There is a basic logging interface to allow for common logging formats for any plugins. To use import the [logger](geneos/pkg/logger) package and then make local copies of the Loggers, like this:

```go
import (
...
	"wonderland.org/geneos/pkg/logger"
)

func init() {
	logger.EnableDebugLog()
}

var (
	Logger      = logger.Logger
	DebugLogger = logger.DebugLogger
	ErrorLogger = logger.ErrorLogger
)
```

Then all of the normal _log_ package methods will work.

The `DebugLogger` is turned off by default and can be enabled using `logger.EnableDebugLog()` and then disabled again using `logger.DisableDebugLog()`. As Loggers are copies of the ones in the logger package the DebugLogger can be enabled or disabled per package. 

For this reason you may want to provide exported package methods to turn debug logging on and off from the calling program.

## Streams

Basic support for [streams](geneos/pkg/streams) are included. Streams must be predefined in the Geneos configuration and sending messages to a non-existent stream name results in an error.

```go
import (
	"wonderland.org/geneos/pkg/streams"
)
```

```go
func main() {
...
	streamsampler := "streams"
	sp, err := streams.Sampler(fmt.Sprintf("http://%s:%v/xmlrpc", hostname, port), entityname, streamsampler)
	if err != nil {
		log.Fatal(err)
	}

	err := sp.WriteMessage("teststream", time.Now().String()+" this is a test")
	if err != nil {
		log.Fatal(err)
		break
	}
```

For convenience the _streams_ package also acts as an _io.Writer_ and _io.StringWriter_ and so will respond to normal Go `Write()` and `WriteString()` calls. You must however call `SetStreamName()` before trying to write messages this way. There is no validation of data content or length.

So, instead of the above you can also do:

```go
	sp.SetStreamName("teststream")

	_, err := sp.WriteString(time.Now().String() + " this is a test")
	if err != nil {
		log.Fatal(err)
		break
	}
```

You can change the stream name as often as you want, but it will be easier to create multiple streams if you need to.

Note that the sampler name is always different to the normal dataview destination as the plugin on the Geneos side must be an _api-streams_ one. Also there is no `Close()` method. At the moment there is no direct support for heartbeats.


## Secure Connections

The packages support secure connections through the normal Golang http.Client but it is quite common for individual Netprobe instances to run with self-signed certificates so there is a method to allow unverified certificates and this must be called immediately after getting the new _sampler_ or _stream_ like this:

```go
	url := fmt.Sprintf("https://%s:%v/xmlrpc", hostname, port)
	p, err := plugins.Sampler(url, entityname, samplername)
	if err != nil {
		log.Fatal(err)
	}
	p.AllowUnverifiedCertificates()
```

Once allowed there is no way to turn this off until you create a new object. The setting is per _sampler_ or _stream_ so it can be on and off separately in the case where your program may send data to multiple Netprobes.
