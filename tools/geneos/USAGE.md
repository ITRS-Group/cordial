# Using `geneos` to Manage Geneos

This guide gives examples of common commands you will likely use day-to-day to manage your Geneos installation.

You should already have installed the `geneos` program in a location that makes it simple to run from the command line. If, however, you still need to do this, please see the [README](README.md) or [INSTALL](INSTALL.md) guides. You will also find details of how to [Adopt An Existing Installation](README.md#adopting-an-existing-installation) in the README guide, if you have a Geneos installation that uses `gatewayctl` and related shell script commands.

## Core Commands

Let's start with some core commands, but first let's take a short look at the typical command line.

> 💡Most commands will accept a similar set of optional arguments. The normal format is:
>
>  `geneos COMMAND [flags] [TYPE] [NAME...]`
>
> The square brackets denote optional arguments and the ellipsis means that the option `NAME` argument can be repeated.
>
> The options `flags` argument(s) will vary from command to command, and you should use the `--help` (or `-h`) flag to see help for a specific command, e.g. `geneos ls --help`
>
> The optional `TYPE` argument can restrict a command to a specific component type, e.g. `gateway` or `netprobe`. For a full list of currently supported component types use `geneos help`.
>
> The optional list of `NAMES` for instance can also restrict commands to only apply to a selected set of instance names. These names can also use wildcards, e.g. `name*` but be careful to quote these as your command line shell may instead apply them first to files in the current directory.
>
> Without `TYPE` or `NAMES` the command will apply to all instances.

### `geneos list`

> Also available as the alias `geneos ls`

To see a list of Geneos instances, including information about the component type and the version of release installed, use the `list` command (or, for UNIX/Linux users,  the shorter alias `ls`):

```bash
$ geneos ls
Type      Name                 Host       Flags Port  Version                     Home
gateway   test1                localhost  A     7038  active_prod:6.5.0           /opt/geneos/gateway/gateways/test1
licd      perm                 localhost  PA    7041  active_prod:6.5.0           /opt/geneos/licd/licds/perm
netprobe  localhost            localhost  PA    7036  active_prod:6.5.0           /opt/geneos/netprobe/netprobes/localhost
san       hdci-ecr1adh01a      localhost  A     7103  netprobe/active_prod:6.5.0  /opt/geneos/netprobe/sans/hdci-ecr1adh01a
```

> 💡 You can also change this output to CSV or JSON formats, for further processing. For all commands you can use the `--help` or `-h` flag to any command to see the options available.

Above you can see these columns:

| Column | Descriptions |
|--------|--------------|
| `Type` | The component type |
| `Name` | The instance name |
| `Host` | The host the instance is configured on |
| `Flags` | Flags that show if the instance is `P`rotected, `A`uto-start, `D`isabled, `T`LS Configured ("Secure Communications") |
| `Port` | The TCP port the instance is configured to listen on |
| `Version` | The component package type, base name and underlying version. For the `san` type the `netprobe/` prefix tells you that the underlying release is a normal Netprobe |
| `Home` | The working (run time) directory |



### `geneos status`

> Also available as the alias `geneos ps`

To see which instances are running use the `status` command (or, again, the shorter alias `ps`):

```bash
$ geneos ps
Type      Name                 Host       PID      Ports        User   Group  Starttime             Version                     Home
gateway   test1                localhost  1017     [7038]       peter  peter  2023-11-15T11:31:06Z  active_prod:6.5.0           /opt/geneos/gateway/gateways/test1
licd      perm                 localhost  1014     [7041 7853]  peter  peter  2023-11-15T11:31:06Z  active_prod:6.5.0           /opt/geneos/licd/licds/perm
netprobe  localhost            localhost  1016     [7036]       peter  peter  2023-11-15T11:31:06Z  active_prod:6.5.0           /opt/geneos/netprobe/netprobes/localhost
san       hdci-ecr1adh01a      localhost  1028     []           peter  peter  2023-11-15T11:31:06Z  netprobe/active_prod:6.5.0  /opt/geneos/netprobe/sans/hdci-ecr1adh01a
```

> 💡 As for the `list` command, you can also change this output to CSV or JSON formats, for further processing.

The output looks similar to the `list` / `ls` command but with some important differences. Notably the `Ports` column contains the actual TCP ports the running process is listening on and these might not be the same as those that may be configured - knowing this can be important in some situations.

The first three - `Type`, `Name` and `Host` - and last two columns - `Version` and `Home` have the same meaning, but then there are these additional columns:

| Column | Descriptions |
|--------|--------------|
| `PID` | The PID (Process ID) of the running process |
| `Ports` | The actual TCP ports the process is listening on |
| `User` and `Group` | The User and Group of the user running the process |
| `Starttime` | The process start time |

> 💡You can also limit the results by giving additional arguments on the command line, for example:

```bash
geneos ps gateway
geneos ps "LDN*"
```

### `geneos start`, `geneos stop` and `geneos restart`

You can control instances using the three commands `geneos start`, `geneos stop` and `geneos restart`. Each command does what the name suggests.

> ⚠ It is important to recall that if you do not specify a `TYPE` or instance `NAMEs` then commands will operate on all matching instances. This is especially important with these three commands and you can, unintentionally, affect instances that you didn't intend to change.

While there are a variety of options to these commands, all visible with the `--help`/`-h` flag, it is worth mentioning these:

* The `--log` / `-l` flag to `geneos start` and `geneos restart` will start watching the log files of all the instances that have been started and will continue to do so until you interrupt it using CTRL-C (which will not affect the running instances).

* The `--all` / `-a` flag to `geneos restart` tells the command to start all matching instances, even if they were already stopped. This is useful when you have a set of instances, e.g. all netprobes, that you need to start but also stop any running instances first.

* The `--force` / `-F` flag tells the command to override protection labels - see below.

### `geneos protect`

To help avoid commands affecting more of your instances than you intended, you can label any of them as _protected_ with the `geneos protect` command. Any instance that is protected will not be affected by commands with side-effects, such as the ones above. Instead, you have to run commands using the `--force` or `-F` flag.

Protecting an instance also prevents accidental deletion and other impacting changes.

You can see which instances are currently protected using the `geeneos list` command; the `Flags` column will show a `P` for each protected instance.

> ⚠ There is no `geneos unprotect` command and this is intentional. Instead there is an `--unprotect` or `-U` to this command to reverse it's affects.

### `geneos disable` and `geneos enable`

Another way to control how instances behave is through the `geneos disable` and `geneos enable` commands. The principal difference between `geneos protect` and `geneos disable` is that if you disable an instance then you cannot override this state per command with `--force`, unlike a protected instance.

Disabling an instance is useful when you want to perform maintenance or you want to create a backup copy of an instance and disable it to ensure it is not started by accident.

Disabled instances show in the `geneos list` output with a `D` flag.

## Managing Installed Software

The `geneos` program includes commands to help manage the installed Geneos releases for each component type. These are all found in the `geneos package` sub-system.

As you have seen, each instance is of a specific component type. Each of these components is associated with a software release available on the [ITRS Downloads](https://resources.itrsgroup.com/downloads) sub-site. To allow you to manage which version is used for which instance we use the concept of a symbolic "base version". In most cases you will see this listed as `active_prod`.

> ⚠ Note that installed packages are only related to their component types and not specific instances. You manage packages _per component_ and additionally with the symbolic _base version_. Each instance then uses a base version, and most commonly all share the same one, per component. All of the commands below only work with components and base versions, not instances.

### `geneos package list`

> Alias `geneos package ls`

You can see what packages are installed using the `geneos package list` command. You can also limit this to a specific TYPE, which can be useful if you have many installed versions.

```bash
$ geneos package ls gateway
Component  Host       Version         Links        LastModified          Path
gateway    localhost  6.5.0 (latest)  active_prod  2023-09-05T10:50:10Z  /opt/geneos/packages/gateway/6.5.0
gateway    localhost  6.4.0                        2023-09-01T08:08:15Z  /opt/geneos/packages/gateway/6.4.0
gateway    localhost  5.14.3          current      2023-09-05T11:02:15Z  /opt/geneos/packages/gateway/5.14.3
```

> 💡 You can also choose to have this information in CSV or JSON formats, for further processing. Use the `--help` or `-h` flag to any command to see the options available.

In the output above you will see the following columns:

| Column | Descriptions |
|--------|--------------|
| `Component` | The component type |
| `Host` | The host for this installed release |
| `Version` | The underlying version, based on the directory name |
| `Links` | The symbolic base version name(s) linked to this release. Note the default `active_prod` but also the `current` base names |
| `LastModified` | The modification time of the top-level directory for the release. This is a fair indication of when it was installed |
| `Path` | The installation path |

### `geneos package install`

You can download and install Geneos releases directly from the command line without a web browser, and then copy files between systems, as long as you can reach the download site from the server you are working on. You may need to configure details for your corporate web proxy - you can read more about this in the `geneos` documentation by following this link or running [`geneos help package install`](https://github.com/ITRS-Group/cordial/blob/main/tools/geneos/docs/geneos_package_install.md).

💡The `geneos package install` command will not, by default, change any running instance - it will only download and install releases. To also update which release version an instance uses you must also either use additional flags, such as `--update` or use the `geneos package update` command.

To download ITRS software you must have a registered account and use these to access the releases. `geneos package install` can accept your user name with the `--username EMAIL` / `-u EMAIL` flag and will then prompt you for your password, or you can use the `geneos login` command to securely save your credentials in an encrypted file.

With no other arguments the command will download and install the latest available version of all supported components. This is normally too many, so you should specify the component type on the command line:

```bash
$ geneos package install netprobe -u email@example.com
Password:
geneos-netprobe-6.6.0-linux-x64.tar.gz 100% |███████████████████████████████████████████████████| (394/394 MB, 27 MB/s)         
installed "geneos-netprobe-6.6.0-linux-x64.tar.gz" to "/opt/geneos/packages/netprobe/6.6.0"
```

The `--update` / `-U` flag also lets you trigger an update of the symbolic base version, which will stop and restart instances as needed, unless they are protected. See the `geneos package update` command below for more. If any protected instances match the base version you want to update then it will skip the update and output a warning telling you this.

If you cannot access the ITRS Download site directly from your server you can instead download and copy them to your server from another location. You can then install from local copies:

```bash
$ geneos package install -L /tmp/downloads/geneos-gateway-5.14.3-linux-x64.tar.gz 
installed "geneos-gateway-5.14.3-linux-x64.tar.gz" to "/opt/geneos/packages/gateway/5.14.3"
```

Conversely, the `--download` / `-D` flag tells the command to only download the release but not to install. You will then be able to find the downloaded release in the current directory.

The `geneos package install` command has a number of other useful options, too many to mention in this guide. Please use the `geneos help COMMAND` or `--help`/`-h` flags to find out more.

### `geneos package update`

The `geneos package update` command lets you control when instances are updated, which installed release versions to use and more.

A basic update can be run at the same time as installation using the `--update` flags to the `geneos package install` command, but the `geneos package update` command let's you exert a finer level of control over specific base versions and releases.

Once you know the version and symbolic base name for the component you want, then:

```bash
$ geneos package update gateway -V 6.6.0 -b dev
```

> ⚠ Remember that all of the `geneos package` commands work on component types and base versions and not on individual instances.

Any Gateway instance that uses the symbolic base `dev` will be stopped, the link updated and the same instances started. If any of the instances are protected then the command will stop and not update the link or stop any of the other matching instances. This is one of the uses for the protected label. If you are sure you want to update, then also supply the `--force`/`-F` flag:

```bash
$ geneos package update gateway -V 6.6.0 -b dev --force
```

> ⚠️ The update will still stop even if the protected instances are not running. The protected label is not just to prevent the instance being stopped but also changed.

### `geneos package uninstall`

To remove old releases and to clean-up the downloaded release files you can use `geneos package uninstall`.

Without any options the command removes older, unused releases and all downloaded archive files. The `geneos package uninstall` command will **not** remove the latest release for any component type or any release that is in use by an instance through a symbolic base version. So that it can remove older releases, the command will also keep any unused symbolic base versions by updating them to the latest available release.

```bash
$ geneos package uninstall
removed "/opt/geneos/packages/downloads/geneos-gateway-5.14.3-linux-x64.tar.gz"
removed "/opt/geneos/packages/downloads/geneos-gateway-6.3.1-linux-x64.tar.gz"
netprobe "localhost" is marked protected and uses version 6.5.0, skipping
licd "perm" is marked protected and uses version 6.5.0, skipping
gateway "Demo Gateway" is marked protected and uses version 6.5.0, skipping
removed gateway release 5.14.3 from localhost:/opt/geneos/packages/gateway
removed gateway release 6.3.1 from localhost:/opt/geneos/packages/gateway
gateway "Demo Gateway@ubuntu" is marked protected and uses version 6.5.0, skipping
removed gateway release 5.14.3 from ubuntu:/opt/geneos/packages/gateway
removed gateway release 6.3.1 from ubuntu:/opt/geneos/packages/gateway
```

> 💡As for other commands, if an instance is labelled protected then no action will be taken without the `--force`/`-F` flag.

## Managing Instances

### `geneos add`

The `geneos add` command lets you add a new Geneos instance. You must supply at least the component type and a name for the new instance, like this:

```bash
$ geneos add netprobe myProbe
certificate created for netprobe "myProbe" (expires 2024-11-24 00:00:00 +0000 UTC)
netprobe "myProbe" added, port 7114
```

### `geneos deploy`

The `geneos deploy` command combines `geneos add` and `geneos package install` to allow you to implement a working Geneos component in a single command. This is especially useful for scripting and automation.

### `geneos delete`

A command that you will not use very often is the `geneos delete` command. 

### `geneos copy` and `geneos move`

> Also aliased as `geneos cp` and `geneos mv`, respectively.



### `geneos set` and `geneos unset`

The instances you create and manage may, over time, need settings changed. Those settings that are not automatically managed by commands can be updated with these commands. You can change simple parameters, which are KEY=VALUE pairs and also more complex parameters like lists of environment variables and more. 


### `geneos rebuild`

Certain Geneos components use configuration files; the Gateway and Self-Announcing / Floating Netprobes. When you create a new instance of these types they also have default configuration files created for them. These files are built from templates and have the details filled in depending on the instance configuration, for example the name and port numbers.

If you change any instance settings then you may need to rebuild the configuration files to reflect these changes.

### `geneos clean`

## Secure Connections

Geneos components can use TLS to encrypt traffic between each other and also for external access. The creation and maintenance of public certificates has improved with LetsEncrypt but as the vast majority of Geneos implementations will be on private networks this is not a real option. Instead the `geneos tls` sub-system lets you work with local certificates, create a certificate authority and instance certificates and keys and renew them when required.

Supporting internal corporate certificates is currently limited, and the commands in this guide are intended for self-contained set of certificates and keys.

### `geneos tls list`

> Also aliased as `geneos tls ls`

If your Geneos installation already has TLS configured then you can list the certificates any their details using the `geneos tls list` command:

```bash
$ geneos tls list
Type      Name                 Host       Remaining  Expires                          CommonName                        Valid
gateway   Demo Gateway         localhost  28378669   "2024-10-18 00:00:00 +0000 UTC"  "geneos gateway Demo Gateway"     true 
licd      perm                 localhost  28378669   "2024-10-18 00:00:00 +0000 UTC"  "geneos licd perm"                true  
netprobe  localhost            localhost  28378669   "2024-10-18 00:00:00 +0000 UTC"  "geneos netprobe localhost"       true  
```

> 💡 You can also choose to have this information in CSV or JSON formats, for further processing. Use the `--help` or `-h` flag to any command to see the options available.

Just like the `geneos list` command for instances themselves, you can see columns for the component type, the instance name and the host. The other columns are:

| Column | Descriptions |
|--------|--------------|
| Remaining | The number of seconds that the certificate remains valid |
| Expires | A human readable expiry time - future releases will normalise this to ISO format |
| CommonName | The Common Name ("CN") of the certificate |
| Valid | A basic validity check that confirms the certificate is inside it's validity period, has the correct type and that it validates against the chain file configured for that instance |

If you use the `--all` / `-a` flag you will also be shown the details of the root and signing certificates. These are not normally needed for reviewing instance settings.

```bash
$ geneos tls list -a
Type      Name                 Host       Remaining  Expires                          CommonName                         Valid
global    rootCA               localhost  310988097  "2033-10-02 00:00:00 +0000 UTC"  "geneos root certificate"          true
global    geneos               localhost  310988097  "2033-10-02 00:00:00 +0000 UTC"  "geneos intermediate certificate"  true
gateway   Demo Gateway         localhost  28373696   "2024-10-18 00:00:00 +0000 UTC"  "geneos gateway Demo Gateway"      true
...
```

You can see many more details by using the long listing format by adding the `--long` or `-l` flag. Normally you would only use the `--long` flag in conjunction with CSV or JSON output as the width of the output becomes difficult to manage for normal human-readable format. One column of information that can be useful is the last one, the `Fingerprint` column, which can be used directly in the Geneos Gateway configuration for some validation/authentication fields.

```bash
$ geneos tls list -al gateway
Type     Name          Host       Remaining  Expires                          CommonName                         Valid  ChainFile                                                     Issuer                             SubjAltNames  IPs  Fingerprint
global   rootCA        localhost  310988058  "2033-10-02 00:00:00 +0000 UTC"  "geneos root certificate"          true   "/opt/geneos/.config/geneos/rootCA.pem"                       "geneos root certificate"                             F19C65E68BD5C0C0C69540D2C4C6EBB6536B4652
global   geneos        localhost  310988058  "2033-10-02 00:00:00 +0000 UTC"  "geneos intermediate certificate"  true   "/opt/geneos/.config/geneos/rootCA.pem"                       "geneos root certificate"                             3F63CAD7BE464123884838A20C86397AD5C0A7EA
gateway  Demo Gateway  localhost  28373658   "2024-10-18 00:00:00 +0000 UTC"  "geneos gateway Demo Gateway"      true   "/opt/geneos/gateway/gateways/Demo Gateway/chain.pem"  "geneos intermediate certificate"  [thinkpad]         82255E4BA89406F29D3753BBA9C205BE536931FE
```

The extra columns shown are:

| Column | Descriptions |
|--------|--------------|
| ChainFile | The certificate chain file used for the instance. The chain file normally contains copies of the root and signing certificates and can be used to verify that a certificate was issued by a valid authority |
| Issuer | The Common Name of the issuer of the certificate |
| SubjAltNames | A list of Subject Alternative Names in the certificate. This is in the format of a command separated list encloded in `[ ... ]` |
| IPs | A list of IP addresses in the certificate, usually empty |
| Fingerprint | The certificate finger print. This is used to verify the identity of a client if they present a certificate, as mentioned in the [documentation](https://docs.itrsgroup.com/docs/geneos/6.6.0/Gateway_Reference_Guide/geneos_authentication_tr.html#authentication__users__user__sslIdentities__id__fingerprint) |

### `geneos tls new` and `geneos tls renew`



### `geneos tls init`

### `geneos tls sync`


## Diagnostics

### `geneos logs`

Every Geneos component creates log files. You can view, search or track these logs using the `geneos logs` command. You can view logs for multiple instances and across multiple hosts, in any combination.

Like other commands, you can run `geneos logs` with no arguments and you will see the logs for all instances. By default you will be shown the last 10 lines per instance log file, with a heading telling you where the log lines are from. You can see less or more by using the `--lines N` or `-n N` flag, where N is the number of lines.

If you use the `--follow` or `-f` flag then the command will follow all the matching instances and output new lines until you interrupt the command using CTRL-C. This flag is just like the UNIX / Linux `tail` utility.

It's also possible to see the whole log file with the `--cat` or `-c` flag, to "grep" (search for matching lines) with the `--match STRING` or `-g STRING` flag, or the opposite and ignore lines with `--ignore STRING` or `-v STRING` flag.

### `geneos show` and `geneos command`




