# Using `geneos` to Manage Geneos

This guide gives examples of common commands you will likely use day-to-day to manage your Geneos installation.

You should already have installed the `geneos` program in a location that makes it simple to run from the command line. If, however, you still need to do this, please see the [README](README.md) or [INSTALL](INSTALL.md) guides. You will also find details of how to [Adopt An Existing Installation](README.md#adopting-an-existing-installation) in the README guide, if you have a Geneos installation that uses `gatewayctl` and related shell script commands.

<!-- TOC -->

* [Core Commands](#core-commands)
    * [geneos list](#geneos-list)
    * [geneos status](#geneos-status)
    * [geneos start, geneos stop and geneos restart](#geneos-start-geneos-stop-and-geneos-restart)
* [Managing Software Releases](#managing-software-releases)
    * [geneos package list](#geneos-package-list)
    * [geneos package install](#geneos-package-install)
    * [geneos package update](#geneos-package-update)
    * [geneos package uninstall](#geneos-package-uninstall)
* [Managing Instances](#managing-instances)
    * [geneos add](#geneos-add)
    * [geneos deploy](#geneos-deploy)
    * [geneos set and geneos unset](#geneos-set-and-geneos-unset)
    * [geneos rebuild](#geneos-rebuild)
* [Secure Connections](#secure-connections)
    * [geneos tls init](#geneos-tls-init)
    * [geneos tls new and geneos tls renew](#geneos-tls-new-and-geneos-tls-renew)
    * [geneos tls list](#geneos-tls-list)
    * [geneos tls sync](#geneos-tls-sync)
* [Diagnostics](#diagnostics)
    * [geneos logs](#geneos-logs)
    * [geneos show](#geneos-show)
    * [geneos command](#geneos-command)
* [Remote Hosts](#remote-hosts)
    * [geneos host list](#geneos-host-list)
    * [geneos host add](#geneos-host-add)
* [AES256 Encrypted Secrets and Credential Storage](#aes256-encrypted-secrets-and-credential-storage)
    * [geneos aes new](#geneos-aes-new)
    * [geneos aes encode and geneos aes decode](#geneos-aes-encode-and-geneos-aes-decode)
        * [App Keys](#app-keys)
    * [geneos aes password](#geneos-aes-password)
    * [geneos login](#geneos-login)
* [Miscellaneous](#miscellaneous)
    * [geneos import](#geneos-import)
    * [geneos protect](#geneos-protect)
    * [geneos disable and geneos enable](#geneos-disable-and-geneos-enable)
    * [geneos migrate and geneos revert](#geneos-migrate-and-geneos-revert)
    * [geneos clean](#geneos-clean)
    * [geneos delete](#geneos-delete)
    * [geneos copy and geneos move](#geneos-copy-and-geneos-move)

<!-- /TOC -->

## Core Commands

Let's start with some core commands, but first let's take a short look at the typical command line.

> 💡Most commands will accept a similar set of optional arguments. The normal format is:
>
>  ```bash
>  geneos COMMAND [flags] [TYPE] [NAME...]
>  ```
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

> ⚠ It's important to realise that if you don't specify a `TYPE` or instance `NAMEs` then commands will operate on all matching instances. This is especially important with these three commands and you can, unintentionally, affect instances that you didn't intend to change.

While there are a variety of options to these commands, all visible with the `--help`/`-h` flag, it is worth mentioning these:

* The `--log` / `-l` flag to `geneos start` and `geneos restart` will start watching the log files of all the instances that have been started and will continue to do so until you interrupt it using CTRL-C (which will not affect the running instances).

* The `--all` / `-a` flag to `geneos restart` tells the command to start all matching instances, even if they were already stopped. This is useful when you have a set of instances, e.g. all Netprobes, that you need to start but also stop any running instances first.

* The `--force` / `-F` flag tells the command to override protection labels - see below.

* The `--extras`/`-x` and `--env`/`-e` to both `geneos start` and `geneos restart` allow you to temporarily alter the starting environment of the instance, for example if the process is not behaving as expected. See the help for the commands for more information.

## Managing Software Releases

The `geneos` program includes commands to help manage the installed releases for each component type. These commands are all found in the `geneos package` sub-system.

As you have seen, each instance is of a specific component type. Each of these component types is associated with a software release available on the [ITRS Downloads](https://resources.itrsgroup.com/downloads) site. To allow you to manage which version is used for which instance we use the concept of a symbolic "base version". In most cases you will see this listed as the default, `active_prod`.

> ⚠️Note that installed releases are **only** related to their component types and not individual instances. You manage packages _per component_ and with the symbolic _base version_. Each instance then uses a base version, and most commonly all share the same one, per component. All of the commands below only work with components and base versions, not instances.

### `geneos package list`

> Alias `geneos package ls`

You can see what releases are installed using the `geneos package list` command. You can also limit this to a specific TYPE, which can be useful if you have many installed versions.

```bash
$ geneos package ls gateway
Component  Host       Version         Links        LastModified          Path
gateway    localhost  6.5.0 (latest)  active_prod  2023-09-05T10:50:10Z  /opt/geneos/packages/gateway/6.5.0
gateway    localhost  6.4.0                        2023-09-01T08:08:15Z  /opt/geneos/packages/gateway/6.4.0
gateway    localhost  5.14.3          current      2023-09-05T11:02:15Z  /opt/geneos/packages/gateway/5.14.3
```

> 💡 You can also output this information in CSV or JSON formats, for further processing. Use the `--help` or `-h` flag to any command to see the options available.

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

With this command you can download and install Geneos releases from the command line, without a web browser, and then copy files between systems as long as you can reach the download site from the server you are working on. You may need to configure details for your corporate web proxy - you can read more about this in the `geneos` documentation by following this link or running [`geneos help package install`](https://github.com/ITRS-Group/cordial/blob/main/tools/geneos/docs/geneos_package_install.md).

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
geneos package update gateway -V 6.6.0 -b dev --force
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

>💡Like for many other commands, if an instance is labelled protected then no action will be taken without the `--force`/`-F` flag.

## Managing Instances

Each instance you create has to have a unique name that is also not the same as any of the special reserved words, such as the component types and their aliases as well as the special "all" and "any". You can have two instances with the same name as long as they are of different component types, such as a Gateway and a Netprobe.

### `geneos add`

The `geneos add` command lets you add a new instance. You must supply at least the component type and a name for the new instance, like this:

```bash
$ geneos add netprobe myProbe
certificate created for netprobe "myProbe" (expires 2024-11-24 00:00:00 +0000 UTC)
netprobe "myProbe" added, port 7114
```

There are a large number of options to the `geneos add` command and you should take time to review them using the `--help`/`-h` flag. Some of the more commonly used ones are:

* `--start`/`-S` can be used to start the instance immediately after creation

* `--log`/`-l` will start the instance after creation and also follow the resulting log file(s) until you interrupt with CTRL-C (which will not stop the instance, just the log output)

* `--import [DEST=]PATH|URL` lets you add file(s) to the working directory of the new instance. This can be used, for example, to import a license file when creating a licence daemon instance, like this:

    ```bash
    $ geneos add licd perm --import geneos.lic=mylicence.txt
    ```

    `DEST` can be either a file name or a directory, if it ends with a `/`. The source for the import can be a local file (where a `~/` prefix means from your home directory) or a URL for a remote file.

    You can repeat this flag multiple times to import multiple files.

* `--env NAME=VALUE`/`-e NAME=VALUE` can be used to set environment variables for the start-up of the instance, e.g.:

    ```bash
    $ geneos add netprobe myJMXprobe --env JAVA_HOME=/usr/local/java11/jre
    ```

    You can see which environment variables are set for the start-up of an instance with the `geneos show` and `geneos command` commands, below.

* `--port N`/`-p N` can be used to override the automatically allocated listening port for the new instance.

* A number of options allow you to influence the creation of default configuration files, which is particularly important for Gateways, Self-Announcing and Floating Netprobes. Please see the component documentation (⚠️not yet complete!) from [`geneos help gateway`🔗](/tools/geneos/docs/geneos_gateway.md), [`geneos help san`🔗](/tools/geneos/docs/geneos_san.md) and [`geneos help floating`🔗](/tools/geneos/docs/geneos_floating.md) for more detailed information.

### `geneos deploy`

The `geneos deploy` command combines `geneos add` and `geneos package install` to allow you to implement a working Geneos component in a single command. This is especially useful for scripting and automation. Similar options to both the `geneos add` and `geneos package install` commands let you control how the new instance behaves.

In effect you use a similar command line as for `geneos add` but also add options for where to obtain the release software in case it is not already installed.

For example, to deploy a Self-Announcing Netprobe, using the default SAN template configuration, you can use something like this:

```bash
$ geneos deploy san LDN_APP1A --gateway gatewayhost1 --gateway gatewayhost2 --type "Infrastructure Defaults" --type "Linux Defaults" --attribute ENVIRONMENT=Production --attribute COMPONENT=Appl -l -u email@example.com
```

### `geneos set` and `geneos unset`

The instances you create may, over time, need settings changed. Those settings that are not directly managed by commands (e.g. the TLS and AES sub-systems) can be updated with these commands. You can change simple parameters, which are KEY=VALUE pairs and also more complex parameters like lists of environment variables and so on.

`geneos set` allows commands like this:

```bash
geneos set gateway LDN1_PROD param1=value
```

To remove a parameter, as opposed to updating it, you should use the `geneos unset` command with the `--key`/`-k` option, like this:

```bash
geneos unset gateway LDN1_PRD -k param1
```

The `-k` is necessary for un-setting a parameter as otherwise the program would not be able to distinguish between an instance name (e.g. `LDN1_PRD`) and the parameter `param1`.

Any instances that have their configurations updated by either command and have their rebuild configuration value set to `auto` will also rebuild configuration files. See below.

### `geneos rebuild`

Some components use configuration files; the Gateway, Self-Announcing and Floating Netprobes. When you create a new instance of these types they also have default configuration files created for them. These files are built from templates and have the details filled in depending on the instance configuration, for example the name and port numbers.

If you change any instance settings then you may need to rebuild the configuration files to reflect these changes.

Use `geneos rebuild` after setting any of the 

## Secure Connections

Instances can use TLS to encrypt traffic between each other and also for external access.The `geneos tls` sub-system lets you work with local certificates, create a certificate authority and instance certificates (and keys) and renew them as and when required.

Imported certificate support is currently limited, and the commands in this guide are intended for self-contained set of certificates and keys.

### `geneos tls init`

Unless you used the `--makecerts`/`-C` option for the initialisation of TLS support when you first deployed your Geneos environment using one of the `geneos init` commands, then you will have to use `geneos tls init` to do so. Just run the command like this:

```bash
$ geneos tls init
```

💡Once your TLS sub-system is initialised all subsequently created **new** instances will have their own certificate and private key, but no existing instances will be updated. To create certificates and keys for existing instances use the `geneos tls new` after initialisation.

When run, the command creates a root certificate and key as well as a signing certificate and key, both pairs of files in your `${HOME}/.config/geneos` directory. The root certificate is only used to sign itself and the 2nd level signing certificate. It is this signing, or intermediate, certificate that is used to sign instance certificates. The command also creates a global chain file which contains both certificates, for verification of instance certificates by components.

You can run `geneos tls init` without affecting an existing TLS environment as it will not overwrite the root and signing certificates unless called with the `--force`/`-F` flag.

### `geneos tls new` and `geneos tls renew`

If you have instances without certificates and you have initialised the TLS sub-system, as above, then you can use `geneos tls new` to create new certificates and keys and also update the starting parameters for instances. Running `geneos tls new` on instances that already have certificates and private keys doesn't change them, so it is generally safe to simply run:

```bash
geneos tls new
```

You can, like with other commands, restrict the actions to specific component types and instances names (even though this is unlikely to be necessary), e.g.:

```bash
geneos tls new netprobe
geneos tls new 'PROD*'
```

To replace existing certificates, either because they have expired or you have new signing certificates for your installation, use the `geneos tls renew` command. This will replace existing certificates (but reuse any existing private keys) so you may want to limit this by specifying component types and names, like this:

```bash
geneos tls renew gateway LDN_1
```

### `geneos tls list`

> Also aliased as `geneos tls ls`

You can list the certificates any their details using the `geneos tls list` command:

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
| SubjAltNames | A list of Subject Alternative Names in the certificate. This is in the format of a comma separated list enclosed in `[ ... ]` |
| IPs | A list of IP addresses in the certificate, usually empty |
| Fingerprint | The certificate finger print. This is used to verify the identity of a client if they present a certificate, as mentioned in the [documentation](https://docs.itrsgroup.com/docs/geneos/6.6.0/security/access-controls/geneos_authentication_tr/index.html#authentication--users--user--sslidentities--id--fingerprint) |



### `geneos tls sync`

When using [remote hosts](#remote-hosts) you can use `geneos tls sync` to copy verification chain file to other servers. The root and signing certificate are contained in the chain file but not the private keys, so you can still only create or renew certificates from the local server.

## Diagnostics

### `geneos logs`

Every Geneos component creates log files. You can view, search or track these logs using the `geneos logs` command. You can view logs for multiple instances and across multiple hosts, in any combination.

Like other commands, you can run `geneos logs` with no arguments and you will see the logs for all instances. By default you will be shown the last 10 lines per instance log file, with a heading telling you where the log lines are from. You can see less or more by using the `--lines N` or `-n N` flag, where N is the number of lines.

If you use the `--follow` or `-f` flag then the command will follow all the matching instances and output new lines until you interrupt the command using CTRL-C. This flag is just like the UNIX / Linux `tail` utility.

It's also possible to see the whole log file with the `--cat` or `-c` flag, to "grep" (search for matching lines) with the `--match STRING` or `-g STRING` flag, or the opposite and ignore lines with `--ignore STRING` or `-v STRING` flag.

### `geneos show`

The `geneos show` command will display various aspects of instance configurations. Without other options it will output the configuration (used by `geneos` to manage the instance, not the component specific configuration) in a JSON format. This configuration included the values of all parameters as well as metadata about the instance itself, e.g.:

```bash
$ geneos show gateway "Demo Gateway"
[
    {
        "instance": {
            "name": "Demo Gateway",
            "host": "localhost",
            "type": "gateway",
            "disabled": false,
            "protected": false
        },
        "configuration": {
            "autostart": "true",
            "binary": "gateway2.linux_64",
            "certchain": "/opt/geneos/gateway/gateways/Demo Gateway/chain.pem",
            "certificate": "/opt/geneos/gateway/gateways/Demo Gateway/gateway.pem",
            "config": {
                "rebuild": "initial",
                "template": "gateway.setup.xml.gotmpl"
            },
            "gatewayname": "Demo Gateway",
            "home": "/opt/geneos/gateway/gateways/Demo Gateway",
            "install": "/opt/geneos/packages/gateway",
            "keyfile": "gateway.aes",
            "libpaths": "/opt/geneos/packages/gateway/active_prod/lib64:/usr/lib64",
            "logfile": "gateway.log",
            "name": "Demo Gateway",
            "options": "-demo",
            "port": 7039,
            "privatekey": "/opt/geneos/gateway/gateways/Demo Gateway/gateway.key",
            "program": "/opt/geneos/packages/gateway/active_prod/gateway2.linux_64",
            "setup": "/opt/geneos/gateway/gateways/Demo Gateway/gateway.setup.xml",
            "usekeyfile": "false",
            "version": "active_prod"
        }
    }
]
```

The parameters in the `configuration` section can be set or changed using `geneos set`.

The underlying configuration file may actually contain variable values in some parameters and you can see these using the `--raw`/`-r` option, as the default output expands these variables. e.g. for the above the raw output may look like this:

```json
            "libpaths": "${config:install}/${config:version}/lib64:/usr/lib64",
```

💡 Note that the use of `config:` prefix in the variables instead of `configuration:` to match the displayed output is a side effect of how the `geneos show` command synthesises the `instance` and `configuration` elements, and the real ("raw") values are whatever is output by `--raw`.

The `geneos show` command can also be used to display the component configuration file, if there is one, with the `--setup`/`-s` option as well as the Gateway specific merged configuration using the `--merge`/`-m` option and more. See the [full documentation](/tools/geneos/docs/geneos_show.md) for the command for more details.

### `geneos command`

The command `geneos command` outputs details of the command line used to start the instance, along with the working (starting) directory and environment variables in that starting environment, e.g.:

```bash
$ geneos command gateway "Demo Gateway"
=== gateway "Demo Gateway" ===
command line:
        /opt/geneos/packages/gateway/active_prod/gateway2.linux_64 -gateway-name Demo Gateway -resources-dir /opt/geneos/packages/gateway/active_prod/resources -log /opt/geneos/gateway/gateways/Demo Gateway/gateway.log -setup /opt/geneos/gateway/gateways/Demo Gateway/gateway.setup.xml -ssl-certificate /opt/geneos/gateway/gateways/Demo Gateway/gateway.pem -ssl-certificate-key /opt/geneos/gateway/gateways/Demo Gateway/gateway.key -ssl-certificate-chain /opt/geneos/gateway/gateways/Demo Gateway/chain.pem -licd-secure -demo

working directory:
        /opt/geneos/gateway/gateways/Demo Gateway

environment:
        LD_LIBRARY_PATH=/opt/geneos/packages/gateway/active_prod/lib64:/usr/lib64
```

### `geneos home`

The `geneos home` command prints the home directory of the matching instance. This is useful as a shell command, like this:

```bash
cd $(geneos home gateway LDN_PROD1)
```

If there is no matching instance or too many then the output will be a parent of the nearest matching item, e.g. `geneos home gateway` will output the directory that all the Gateway sub-directories and other configurations are kept in, and `geneos home` or a no-match will output the directory of the overall Geneos installation.

## Remote Hosts

You can manage Geneos instances across multiple Linux servers transparently using SSH. In many production environments this feature will not be allowed by your local security policies, as most Geneos installations are managed using service accounts and direct access to service accounts is, typically, blocked. If this is not the case for you then these features will make managing even a moderately sized Geneos estate much simpler.

The `geneos host` feature works by using SSH to connect remote servers, using public/private keys for password-less access, typically through a local SSH Agent to safely hold your credentials, or through locally encrypted username/password credentials. You should use `ssh-agent` where possible, and this should be implemented on your local system / desktop for the appropriate security. While beyond the scope of this guide, it is worth noting that all modern Windows desktops support the OpenSSH Agent as a service, but it is disabled by default.

You will have seen in earlier examples the `Host` column in the outputs of `geneos list` and `geneos status`. This is the `geneos` label for each host and not the server hostname - but they could be the same. `localhost` and `all` as reserved names. When referring to a specific instance you can use the format `NAME@HOST`, where `NAME` is the instance name and `HOST` is the host label. If you do not specify one or the other then this is treated as a wildcard and means either all instance on `HOST` or `NAME` on all hosts, respectively. In addition to this name format you can also limit commands to specific hosts using the `--host HOST` or `-H HOST` option, as some commands may not accept instance names, such as the `geneos package` sub-system.

### `geneos host list`

Use this command to show a list of existing remote hosts. The command will only list remote hosts and will not show details of the `localhost`.

```bash
$ ./geneos host list
Name    Username  Hostname  Flags  Port  Directory
ubuntu  -         ubuntu    -      22    /home/user/geneos
```

### `geneos host add`

To add a new remote server use the `geneos host add` command. A host must have a local name and information about the remote server connection. You must supply either a `NAME` or an `SSHURL` or both. If you give just the `NAME` then the `SSHURL` will use that as the host name and use defaults for the other values, or if you only supply the `SSHURL`, then the local name will be set to the host name of the remote server, like this:

```bash
geneos host add server1
geneos host add ssh://server2.example.com
geneos host add server3 ssh://user@server3.example.com/opt/itrs
geneos host add server4 -p
```

All of these examples will add a remote host, but with different options.

1. `geneos host add server1`

This command adds a remote host called `server1` using the same host name and uses the default SSH port 22 and the user name of the one running the command. Authentication will assume and SSH Agent as no password is given. The remote Geneos installation will be located in the user's home directory using the same rules as for a local installation, i.e. if the user name is `geneos` then directly in the home directory, otherwise in a sub-directory called `geneos`.

> 💡More strictly, the directory should match the command name, so if you have renamed `geneos` to something else then that user name will be checked or a sub-directory with that name will be used.

2. `geneos host add ssh://server2.example.com`

This command adds a remote host called `server2.example.com`, as no shorter name is given, again with defaults and using SSH Agent for authentication and install directory.

3. `geneos host add server3 ssh://user@server3.example.com/opt/itrs`

This command adds a remote host called `server3` on a remote server with the full hostname of `server3.example.com` in a directory `/opt/itrs`. The port number and authentication are as for the examples above.

4. `geneos host add server4 -p`

This command is similar to 1. above but will prompt for a password which is stored in the hosts configuration file encrypted using AES256 with your default user key file.

## AES256 Encrypted Secrets and Credential Storage

Geneos Gateways support customer AES256 key files for the transparent encryption and decryption of passwords in configuration files. These key file can also be used with Toolkit samplers to support Secure Environment Variables. All instances created by `geneos` also get a new key file unless you have specified an existing one in advance. 
  
### `geneos aes new`

### `geneos aes encode` and `geneos aes decode`

#### App Keys

### `geneos aes password`

### `geneos login`

You can store encrypted credentials using `geneos login` for a number of uses:

1. Downloading Geneos releases
2. Use by other tools, including libemail.so and dv2email for both Gateway and SMTP Relay authentication
3. Gateway REST Command API authentication
    * e.g. the `geneos snapshot` command

The credentials are encrypted using an AES256 key-file, which defaults to your user key-file; See the section above for more details. The credentials file itself is in JSON format and without the key-file to encrypt the included credentials it is not at risk of revealing the plain text versions of them.

At the time of writing the only types of credential supported is username / password combinations. Each credential is associated with a domain, which helps identify which credential to use for a given request.

## Miscellaneous

### `geneos import`

The `geneos import` command let's you copy files into instance directories from a variety of sources. As for many other commands this also applies to multiple instances. Using this command instead of de facto Linux commands like `cp` or `mv` ensures that the files are placed in the correct location and also have appropriate permissions for access by your user account and hence any instances you start.

### `geneos protect`

To help avoid commands affecting more of your instances than you intended, you can label any of them as _protected_ with the `geneos protect` command. Any instance that is protected will not be affected by commands with side-effects, such as the ones above. Instead, you have to run commands using the `--force` or `-F` flag.

Protecting an instance also prevents accidental deletion and other impacting changes.

You can see which instances are currently protected using the `geeneos list` command; the `Flags` column will show a `P` for each protected instance.

> ⚠ There is no `geneos unprotect` command and this is intentional. Instead there is an `--unprotect` or `-U` to this command to reverse it's affects.

### `geneos disable` and `geneos enable`

Another way to control how instances behave is through the `geneos disable` and `geneos enable` commands. The principal difference between `geneos protect` and `geneos disable` is that if you disable an instance then you cannot override this state per command with `--force`, unlike a protected instance.

Disabling an instance is useful when you want to perform maintenance or you want to create a backup copy of an instance and disable it to ensure it is not started by accident.

Disabled instances show in the `geneos list` output with a `D` flag.

### `geneos migrate` and `geneos revert`

### `geneos clean`

All components generate log, temporary and other files. You can use the `geneos clean` command to remove the files each component creates but is no longer likely to be useful. The set of files removed is different for each component type and can also be changed for your specific requirements. It's also possible to perform a more complete clean-up, called a purge, when an instance is not running.

### `geneos delete`

A command that you will not use very often is the `geneos delete` command.

### `geneos copy` and `geneos move`

> Also aliased as `geneos cp` and `geneos mv`, respectively.

Instance can be duplicated using the `geneos copy` command or renamed or moved between servers using `geneos move`. While these commands are useful on your local server, they become more so between servers. See [Remote Hosts](#remote-hosts) for more examples.
