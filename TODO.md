# To Do list

## `geneos` tool

(unordered)

* Windows build, in stages:
  * Initially for remote management (ssh etc)
  * Add netprobe support, services?
* Split 'init' common and sub-sub-commands, as it's too complicated
* Add a 'selfupdate' like, but simpler than, rclone
* When 'moving' a gateway, update licd connection details
  * Also, keep port(s) unchanged where possible
* TLS sync should copy root CA
* Positive confirmations of all commands unless quiet mode - PARTIAL
  * Should be an 'action taken' return from commands for output
  * create a seperate "verbose" logger and work through output to choose
  * or more if verbose ... logic
* Warnings when a name cannot be processed (but continue)
  * Help highlight typos rather than skip them
* Command line verbosity control - PARTIAL
* TLS support
  * output chain.pem file / or to stdout for sharing
* Docker Compose file build from selection of components
* check capabilities and not just setuid/root user
* Run REST commands against gateways
  * initially just a framework that picks up port number etc.
  * specific command output parsing
* command should show user information
* standalone collection agent
* centralised config
* web dashboard - mostly done, better port numbers and tls to do
* Support gateway2.gci format files
* Add a 'clone' command (rename without delete) - for backup gateways etc.
  * reset configs / clean etc.
* Redo template support, primarily for SANs but also gateways
  * document changes
* Update docs to include configuration file rebuilds, gateway includes etc.
* Look at 'sudo' support for remotes
* web interface
  * first pass review configs
  * second to edit
  * use a REST interface
* move/copy - need to update ports when moving to another remote or copying to same remote
* explore gRPC and other options over ssh for remotes (required daemon mode)
* add socket and open file details to ls (ala lsof) - perhaps a "details" command or an option to "show" ?
  * /proc/N/fd/* links

## libemail.so

* fix image attachments (0-many, cid links by name)
* allow empty text template
* attach dataview snapshot as html table

## Other

### XML-RPC API

* Reconnection support
* Look at contexts
* Heartbeat support by default
* Add higher level methods to update small sets of data, e.g. rows
