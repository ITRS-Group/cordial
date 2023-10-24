The `geneos` program will help you manage your Geneos environment.

The program will help you initialise a new installation, migrate an old `geneos-utils` one, install and update software releases, add and remove instances, control processes and build template based configuration files for SANs and more.

The program works best on Linux but is built for both Windows and MacOS. In the latter two cases it is primarily for the remote management (see below) of Linux instances. There is no support at this time for managing local Windows or MacOS instances of Geneos components.

Most commands work on "instances" of "components". As the names suggest, a "component" is a type of Geneos component such as a Gateway or a Netprobe. An "instance" is a configured instance of a component, a specific Gateway and so on. For a list of components see the "Registered Component Types" below. Each component has it's own help which you can read using `geneos gateway help` etc.

Instances can be created locally or on remote hosts over SSH connections without the need to install `geneos` on the remote Linux server.

Instance names are in the form `[TYPE]:NAME[@HOST]`, where the `[...]` mean that part is optional. The `TYPE` is only used to select the underlying type of Netprobe, e.g. Fix Analyser or plain, for Self-Announcing and Floating Netprobe components during deployment. The `HOST` part is the name of a configured remote host (which may not be the hostname); see the `host` sub-system help with `geneos host help` for more information.

For many commands you can also use wildcards for the `NAME` part. These wildcards are not complex regular expressions but instead follow more common file system patterns. (Note that the exact patterns support are the same as for the Go [`path.Match`](https://pkg.go.dev/path#Match) function.). These wildcards only work on `NAME` and not the `HOST` part and then only for those commands where they make sense, such as `geneos ls`, `geneos start` and so on.

The subsystems below group related functions together and have their own sub-commands, such as `geneos aes password` and `geneos init demo`. Use `geneos SUBSYSTEM help` to see more or, if you are reading this online you should be able to click through for further information.
