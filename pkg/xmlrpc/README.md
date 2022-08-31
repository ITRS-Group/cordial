# xmlrpc

The xmlrpc package is the low level implementation of an ITRS Geneox XML-RPC client
API to communicate with a Netprobe.

The package implements structs that keep the minimum state data. No state is
kept regarding the data being sent to the Netprobe.

The files in this package have the following functions:

* xmlrpc.go
    This file provides the NewClient() method and some common logging setup

    NewClient() is the initial entry point to create an xmlrpc.Client and it
    returns a Sampler struct. While there are lowe evels calls to check for
    the existance of specific Managed Entities and Samplers and also to check
    for Gateway connectivity, these are not normally exposed and used by common
    plugin code.

    The principal consumer of this method is the ConnectSampler method in the
    plugins package.

* client.go
    The client.go file defines the Client struct which forms the common basic for
    other structs in this package. The principal method in this file is, oddly,
    NewSampler() which given a variable of type Client will return a more useful
    Sampler object.

* sampler.go
    Sampler level struct and methods. The principal method is NewDataview() to
    create a new dataview on the given sampler. The NewDataview() method checks for
    and, if found, removes any dataview with the same name. The CreateDataview()
    method just does it directly, wuthout checking.

* dataview.go
    Almost all data updates are done in this file. Creating and removing headlines,
    rows and creating columns are all done through here. You cannot remove columns
    using the API and so there is no mapping to an exported method to try the same.

    UpdateTable and AddRow/RemoveRow are the most likely to be used methods.

    Other methods provide a way to query the size of the dataview dimensions and names
    of rows, columns and headlines as per the published Geneos API.

* streams.so
    The basic stream methods in a golang form.

* rawapi.go
    All the methods in this file are unexported but map to the Netprobe API and
    API-Streams interface functions by name and including the list of arguments
    (some functions use separate args for entity and sampler, most a merged
    single argument). The only method not implemented is a deprecated function to
    create entities on ancient Gateway 1 systems.

    The API-Streams signOn and signOff functions have been renamed to not clash with
    the pricipal functions of the same name but with different argument paths. This
    shouldn't matter unless you intend to use them yourself.

* internals.go
    Like the name suggests, internal methods to provide common code the the rawapi.go
    methods, including post() and a custom marshal() mthod.

    One change from the underlying API is that error messages are merged back into
    to golang type err as an int + string.
