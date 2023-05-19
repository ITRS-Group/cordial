# To Do list

These should be moved into github issues

## `geneos` tool

(unordered)

* Windows build, in stages:
  * Initially for remote management (ssh etc)
  * Add netprobe support, services?
* Add a 'selfupdate' like, but simpler than, rclone
* When 'moving' a gateway, update licd connection details (if licd-host is undefined or localhost)
* Positive confirmations of all commands unless quiet mode - PARTIAL
  * Should be an 'action taken' return from commands for output
  * create a separate "verbose" logger and work through output to choose
  * or more if verbose ... logic
* Warnings when a name cannot be processed (but continue)
  * Help highlight typos rather than skip them
* Command line verbosity control - PARTIAL
* TLS support
  * output chain.pem file / or to stdout for sharing
  * TLS sync should copy root CA
* Docker Compose file build from selection of components
* Run REST commands against gateways
  * initially just a framework that picks up port number etc.
  * specific command output parsing
* centralised config
* web dashboard - mostly done, better port numbers and tls to do
* Support gateway2.gci format files
* web interface
  * first pass review configs
  * second to edit
  * use a REST interface
* explore gRPC and other options over ssh for remotes (required daemon mode)
* add socket and open file details to ps (ala lsof) - perhaps a "details" command or an option to "show" ?
  * /proc/N/fd/* links

## Other

### XML-RPC API

* Reconnection support
* Look at contexts
* Heartbeat support by default
* Add higher level methods to update small sets of data, e.g. rows
