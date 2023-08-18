# Change Log

## Version v1.8.1

> **Released 2023-08-xx**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.8.1 Fixes

* [`tools/geneos`](tools/geneos/README.md)
  - `unset` should not present a warning if special parameters are passed but not actions, e.g. removing a non-existing environment variable

---

## Version v1.8.0

> **Released 2023-08-16**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.8.0 Highlights

This version changes the way [`geneos tls import`](tools/geneos/docs/geneos_tls_import.md) works to add support for the import of external "real" certificates into your Geneos environment. You can now supply a PEM formatted certificate and key with an options verification chain and add these to existing instances.

Key files are now enabled for use by default for all new Gateway instances. Key files have been automatically created for some time now, but not automatically enabled in the starting environment of the Gateway.

## v1.8.0 Changes

* [`tools/geneos`](tools/geneos/README.md)
  - Enable the use of external key-files for all *new* Gateways running on version GA5.14 and above. Existing Gateways will not be affected as the default is `usekeyfile=false`. If you do not want to use an external key-file set `usekeyfile=false` before starting for the first time. If a Gateway has been started with or without a keyfile and created a cache directory then you must follow the instructions in the documentation, <https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm#How_to_change_the_key_file_of_your_Gateway>, otherwise your Gateway will not start-up.

  - New options to the `start`, `restart` and `command` sub-commands allow you to add one-off command line arguments and environment variables to an instance. This is useful, for example, to pass a `-skip-cache` argument to a Gateway.

  - Extensive rework to the internal handling of loops-over-instances to pass-back an `instance.Response` struct and handle output at the caller. This is preparation for work on non-CLI interfaces (think: REST API and web). This may break some output formatting, please report via github issues.

  - `tls import` has changed to support the import of instance certificate, signing certs and chains in a more organised way. It is unlikely anyone was using the previous incarnation which was highly limited but just in case, this is a **breaking change** to the syntax and functionality of `tls import`.

* [`pkg/geneos/api`](pkg/geneos/api/README.md)
  - A new API for inbound data to Geneos. This package is not yet ready for real-world use.

## v1.8.0 Fixes

* [`pkg/config`](pkg/config/README.md)
  - [#176](https://github.com/ITRS-Group/cordial/issues/176) fix support for Windows paths in `${enc:...}` expansion formats

* [`tools/geneos`](tools/geneos/README.md)
  - A fix for a long time bug in an internal routine that checked reserved names. This was found during the refactoring of code above. Oddly this doesn't appear to have been noticed, not sure why.

  - Fix closing of open file descriptors when starting a local instance. This needed cmd.Extrafiles slice having empty nils added through the the largest FD.

  - Fix merging of aliases during instance config load.

---

## Version v1.7.2

> **Released 2023-07-28**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.7.2 Changes

* `pkg/gwhub`, `pkg/icp`, `pkg/streams` and `pkg/geneos` have been updated to match real APIs and to add access to REST API streams

## v1.7.2 Fixes

* [#172](https://github.com/ITRS-Group/cordial/issues/172) - viper doesn't do the right thing with overridden values in maps containing defaults. This would affect GetStringMap*() callers, and we also now have our own UnmarshalKey() function

* `pkg/config` and `tools/geneos`: Fix handling of command line plaintext passwords (those not prompted for). When passed a pointer to a method you have to set the destination of the pointer, not the ephemeral pointer itself

* `tools/geneos` would not correctly initialise web server directories after changes to import earlier in v1.7. This is now fixed along with the removal of a confusing treatment of "~/" in an import path not meaning the user's home directory

---

## Version v1.7.1

> **Released 2023-07-25**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.7.1 Changes

* `pkg/gwhub` & `pkg/icp` - Updates for ongoing project

## v1.7.1 Fixes

* [#167](https://github.com/ITRS-Group/cordial/issues/167) - Only load template files with a `.gotmpl` extension.

* [#169](https://github.com/ITRS-Group/cordial/issues/169) - If the file being imported is the *same* as the destination, skip the copy.

* `tools/geneos` - Fix installation of packages from local sources with or without component on command line

---

## Version v1.7.0

> **Released 2023-07-11**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.7.0 Changes

* `tools/geneos` - Optimisation and parallel execution. All operations on Geneos instances are now run in parallel which has resulted in significant improvements in responsiveness and has reduced delays waiting for things to happen on larger installations and remote hosts. While extensive testing has taken place to ensure that the underlying functionality is not affected, there may still be issues in some cases - please report them as soon as you can!

* Add support for TLS key type selection, defaulting to ECDH (see `geneos help tls init`)

* Split `help` and `-h` options - `help` now gives the long description and usage while `--help`/`-h` only gives short description plus usage

* `pkg/geneos` updates to XML parsing structures, fix regex handling

* `pkg/gwhub` updates for better API support (work in progress)

* `pkg/config` updates, with some API changes, to better support `tools/geneos` configuration handling and other refactoring and update ExpandString option NoDecode()

* Use `upx` for compression of binaries during releases build - saves about 2/3rd space

* Make consistent the handling of TLS certs and keys internally

* `geneos ps` will show the actual version of each instance running, in case the base symlink has been updated and the process not restarted

* Quite a bit of redecorating inside `tools/geneos` internal packages to make things clearer (refactoring, merge and split of functions etc.)

* `tools/geneos` - Initial support for "remote only" working; i.e. if GENEOS_HOME is not set but there are remotes then try to "do stuff". This will break if you perform a local operation such as `add` as the root then is the current directory. Further work required, but getting Windows support working again is on the way.

* `tools/geneos` - Add a basic `--install` option to `package update` to allow checking of package that match the ones being updated and download them if found.

## v1.7.0 Fixes

* [#156](https://github.com/ITRS-Group/cordial/issues/156) - fix progressbar newline issue

* [#155](https://github.com/ITRS-Group/cordial/issues/155) - refactor instance home directory handling (mostly internal)

* [#153](https://github.com/ITRS-Group/cordial/issues/153) - fix local install of only components available

* `tools/geneos` - fix order of columns in plain `geneos ls`

* [#38](https://github.com/ITRS-Group/cordial/issues/38) - fix update stop/start as well as a number of related issues in `package install` and the handling of `fa2` packages

* [#152](https://github.com/ITRS-Group/cordial/issues/152) - call Rebuild() on *every* instance config save - then instance.setup.xml will stay in sync with config

* [#150](https://github.com/ITRS-Group/cordial/issues/150) - document `deploy` behaviour when versions clash

## 1.7.0 Known Issues

* [#165](https://github.com/ITRS-Group/cordial/issues/165) - restarting while updating SANs is not working

---

## Version v1.6.6

> **Released 2023-06-28**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.6.6 Fixes

* [#145](https://github.com/ITRS-Group/cordial/issues/145) - Wrap `geneos tls list` JSON output in an array correctly

* `tools/geneos` - Package version handling was broken in some cases, especially for components with 'parent' types

## v1.6.6 Changes

* `tools/geneos` - Add a progress bar for downloads when running interactively. Make checking if running interactively consistent

* `tools/geneos` - Add `package install -D` to download packages without unpacking them

* `tools/geneos` - Refactor internal variable names in subsystem packages to shrink very long names

* `pkg/geneos` - Various updates to plugins and other structures to support an ongoing project

---

## Version v1.6.5

> **Released 2023-06-23**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.6.5 Fixes

* [#146](https://github.com/ITRS-Group/cordial/issues/146) - Entering empty passwords could cause pointer panics

* [#148](https://github.com/ITRS-Group/cordial/issues/148) - Fallback to environment vars when user.Current() fails because user is not in local passwd file with static binary.

---

## Version v1.6.4

> **Released 2023-06-22**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

Note: v1.6.3 was removed, and v.1.6.4 releases to address some last minute issues.

## v1.6.4 Changes

* `tools/geneos` - Add an initial `hidden` flag to hosts, so they don't participate in wildcarded operations.

* `tools/geneos` - Add new `--validate` option to `geneos show` to run a validation and output results as JSON. 

* `pkg/geneos` - Updates for further parsing of config files, fix the treatment of `geneos.Value` type.

* `pkg/config` - Add a `config.UseEnvs()` option to `New()` and `Load()` to trigger viper `AutomaticEnv()` and use prefixes.

* Convert all packages and programs to use `*config.Plaintext` and not `config.Plaintext`.

## v1.6.4 Fixes

* Fix remote host optional encoded password handling

* [#142](https://github.com/ITRS-Group/cordial/issues/142) - Fix expansion of non-encoded config strings in `show` and other places

* [#140](https://github.com/ITRS-Group/cordial/issues/140) - Fix generation of 'secure' args for command start-up

* [#138](https://github.com/ITRS-Group/cordial/issues/138) - Fix autostart behaviour for `geneos restart`

* [#139](https://github.com/ITRS-Group/cordial/issues/139) - Show running AC2 instances

* [#134](https://github.com/ITRS-Group/cordial/issues/134) - Update some `geneos` commands that need either flags or args set to just output usage otherwise.

* [#133](https://github.com/ITRS-Group/cordial/issues/133) - Check restart logic and fix for when instance is already stopped. Also update the Stop() function and it's usage in other callers.

---

## Version v1.6.2

> **Released 2023-06-14**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.6.2 Fixes

* `tools/geneos` Fix late found bug with `deploy` and home directories

---

## Version v1.6.1

> **Released 2023-06-13**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.6.1 Changes

* `pkg/geneos` Changes

  Some API changes to support work on a project for reading Gateway configuration files. Existing structures used to write config files cannot co-exist and have been renamed with an "Out" suffix down to the SamplerOut level. While the old names should have been retained and the new API requirements used new names, it was decided that this is the more common use case in the future.

* `tools/geneos` Command updates

  The `show` command can now output an instance's own configuration file (for types of Netprobe and Gateway) and also try to produce a merged Gateway file using a modified command line with the Gateway `-dump-xml` command line option.

  A new instance flag `autostart` has been added, set to `true` for all types except `ac2` which defaults to `false`. Documentation updtes to follow.

  The `init demo` command now detects if the user has a `DISPLAY` environment variable set and if so also installs an `ac2` instance.

  The `command` command can now output the details in JSON format. This format is not quite compatible with the `pkg/process` Run and Batch functions, but the aim is to eventually merge the formats so that they can also share the implementation later.

## v1.6.1 Fixes

* Minor ongoing changes

  While adding new features there is ongoing review work and refactoring of code.

---

## Version v1.6.0

> **Released 2023-06-07**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.6.0 Changes

* [#116](https://github.com/ITRS-Group/cordial/issues/116)

  Added a new [`geneos deploy`](tools/geneos/docs/geneos_deploy.md)
  command that combines `geneos init` and `geneos add` but takes more
  care over existing installations and/or creating new directories.

  As part of this work all the `geneos init` command will prompt the
  user for a directory if none is given on the command line. If the
  command is run from a non-interactive parent (e.g. a pipe from the
  shell) then the prompt is skipped and the default directory is used.
  
  The `geneos deploy` command uses the same initialisation rules but
  reduces the number of options. The intended audience is around
  automation where the deployment scripts may not have the knowledge or
  logic to check for existing installations.

* [#114](https://github.com/ITRS-Group/cordial/issues/114)

  For `geneos` instances that have both the default `libpaths` and an
  environment variable `LD_LIBRARY_PATH` configured these are now
  concatenated with `libpaths` always first.

* [#117](https://github.com/ITRS-Group/cordial/issues/117)

  Based on user feedback all the Netprobe types have been merged under
  the `netprobe/` directory in their respective plural names, e.g.
  `netprobe/sans`. Existing installations should continue to work
  unchanged but you can use the `geneos migrate` command to
  automatically merge the instance directories under `netprobe/`
  including the update of configuration files.

* [#97](https://github.com/ITRS-Group/cordial/issues/97)

  The Linux Active Console is now treated like any other component and
  instance. At the moment, if you issue a `geneos start` command then
  all instances including Active Console(s) will be run. In a future
  release we may add an `autostart` like flag that can prevent this and
  require a manual start using the full `geneos start ac2 abcde` syntax.

* [`pkg/process`](pkg/process/) - New features

  New functions have been added to support the running of single
  processes and batches based on a Program struct. This is for running
  tasks loaded from a config file (typically YAML) for an ongoing
  project. The reason for not using existing external packages was the
  integration with other `cordial` tooling. This functionality is
  currently maturing and is very sparsely documented and subject to
  major changes.

* [`pkg/icp`](pkg/icp) and [`pkg/gwhub`](pkg/gwhub) - New APIs

  These two packages are the start of Go APIs for ITRS Capacity Planner
  and Gateway Hub, respectively. These should not yet be used and have
  been included to track progress over the next few releases.

## v1.6.0 Fixes

* [#126](https://github.com/ITRS-Group/cordial/issues/126)

  In the [config package](pkg/config/) the Load() function would fail if
  used with a file format set bu other defaults and run in the same
  directory as the binary it ran in because viper would also try to load
  the bare-named program binary as a config file of the type given. The
  package now does it's own file name construction to avoid this.

  As a consequence of the work done around this fix to make the usage of
  options to Load() and Save() clearer some have changed names. Existing
  code that wants to use v1.6.0 will experience minor API breakage. The
  fixes are simple refactors, so no backward compatibility has been
  retained.

* [#124](https://github.com/ITRS-Group/cordial/issues/124)

  The work done for
  [#117](https://github.com/ITRS-Group/cordial/issues/117) above meant
  that all templates are now located under `netprobe/templates` and both
  `san` and `floating` templates had the same name. The default
  templates now have updated root names, e.g. `san.setup.xml.gotmpl`,
  but existing configuration may need updating if the existing templates
  clash.

  To help users control which configuration files are created from
  templates, and which to use for instance start-up, a new instance
  parameter `setup` has been introduced for this. The defaults are
  `netprobe.setup.xml` and `gateway.setup.xml` for the two affected
  component types which means no change for existing users.

  So even though the new SAN template is `san.setup.xml.gotmpl`, for
  example, running `geneos rebuild san` will still result in a
  `netprobe.setup.xml` in the instance directory.

* [#120](https://github.com/ITRS-Group/cordial/issues/120)

  While this report was a misunderstanding of the way to use Daemon()
  the comments have been updated to give better direction on how to use
  the pidfile io.Writer parameters.

---

## Version v1.5.2

> **Released 2023/05/31**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.5.2 Changes

  * [#102](https://github.com/ITRS-Group/cordial/issues/102) - `process`
    package additions

  * [#109](https://github.com/ITRS-Group/cordial/issues/109) - New `tls
    create` command

    Move the functions of `--name` etc from `tls new` to `tls create` to
    remove dependency on the Geneos home directory.

  * [#106](https://github.com/ITRS-Group/cordial/issues/106) - Change
    directory for TLS root and signing certs and keys

  * [#97](https://github.com/ITRS-Group/cordial/issues/97) - Start of
    Linux AC2 support. Not yet fully functional.

  * [#98](https://github.com/ITRS-Group/cordial/issues/98) - Work done
    and then superceeded by
    [#109](https://github.com/ITRS-Group/cordial/issues/109) above.


## v1.5.2 Fixes

  * [#111](https://github.com/ITRS-Group/cordial/issues/111) - Fix
    gateway instance template ports

  * [#103](https://github.com/ITRS-Group/cordial/issues/103) - Fix
    default web server log file name

## Version v1.5.1

> **Releases 2023/05/25**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

## v1.5.1 Changes

* [`tools/geneos`](tools/geneos/README.md):

  * [#85](https://github.com/ITRS-Group/cordial/issues/85) - TLS Issues

    Added verification for certificates against existing root and
    signing certificates.

    Added an option to recreate root and signing certs from `tls init`.

  * [#86](https://github.com/ITRS-Group/cordial/issues/86) - New log
    option for stderr logs blocked main logs

    Reworked the way that logs are followed to allow both normal and
    stderr logs to be followed. This fixes `start -l`, `restart -l` and
    by implication `init -l` options. Added a `--nostandard`/`-N` option
    to `logs` to allow normal log to be surpressed when you only want
    the error logs.

  * [#88](https://github.com/ITRS-Group/cordial/issues/88) - Floating
    probe configuration template output malformed

    The defaults for new floating probes used `sanname` instead of
    `floatingname` as the parameter for the template rendering.

  * [#90](https://github.com/ITRS-Group/cordial/issues/90) - Variables
    in SANs

    The san template was not corrected updated for the new variable
    structure.

  * [#43](https://github.com/ITRS-Group/cordial/issues/43) - Minor error
    in `disable`
   
    In `disable` check if stopped and print error if not `--force`

---

## Version v1.5.0

> **Released 2023/05/22**
>
> Please report issues via [github](https://github.com/ITRS-Group/cordial/issues) or the [ITRS Community Forum](https://community.itrsgroup.com/).

### v1.5.0 - Highlights

This release brings numerous changes to the `cordial` tools, especially
the `geneos` utility. We have tried to improve the reliability and
usability of the utility through updated and improved documentation and
subsequent fixes and changes that arose from writing and checking that
documentation.

### v1.5.0 - **BREAKING CHANGES**

  * `pkg/logger`:

    * **REMOVED**. This simplistic internal logging package has been
      deprecated in favour of more mature solutions, such as
      [zerolog](https://pkg.go.dev/github.com/rs/zerolog).

  * `tools/geneos`:

    * The way SAN instances handle gateway connections has been fixed to
      allow gateway represented as FDQNs or IP addresses. The old way
      resulted in a mess - viper defaults to dots ('.') as configuration
      item hierarchy delimiters and this caused issues. Most users with
      SANs should not notice any change, but if you see problems please
      check the san XML file and correct the `gateways` section as
      necessary. One way is to remove and re-set them using:
      
      > `geneos set san mySan -g gateway.example.com ...`

      Running `set` will rewrite the configuration in the new format bu
      there is a chance that the previous configuration will continue to
      occupy settings. You may need to manually edit the instance
      configuration file `san.json` anyway.

    * Like the above any variables defined for either SAN or Gateway
      instances used to generate XML from templates will have not worked
      in a case sensitive manner to mirror how Geneos treats variable
      names. To fix this the internal format of the `variables` section
      has been updated to move the variable name from the configuration
      key to a separate structure as it's own value. Code has been added
      to automatically convert from the old format to the new when the
      configuration file is updated however there is no fix for the
      correction of variable case name being incorrect from previous
      configurations. Please review and adjust as necessary.

    * Support for running instances as other user accounts or under
      `sudo` has been deprecated. Security is _hard_, and the support
      for these was poorly implemented. A better way should be coming in
      a future release.

      This may mean that where users has configured netprobes to run as
      different users and have previously run `sudo geneos start` to let
      the program do the right thing will run into issues. Please be
      careful if any of your instances run as other users and do not run
      the `geneos` program with `sudo`. There is no additional
      checking/rejection of running under `sudo` or any other privilege
      escalation system so this is important!

### v1.5.0 - Other Changes

  * There has been a significant amount of refactoring and moving around
    of the code-base. Most of this should not be user visible, but some
    public APIs have changed. As with all major changes there may be
    problems that have not been caught in testing. Please report
    anything you see as either a [github
    issue](https://github.com/ITRS-Group/cordial/issues) or via the
    [ITRS Community Forum](https://community.itrsgroup.com/).

    There are too many changed to list them all in detail but specific
    ones worth mentioning include:

    * [`memguard`](https://pkg.go.dev/github.com/awnumar/memguard)
      support for protected memory. Credentials (passwords, TLS keys and
      so on) should now be handled as Enclaves (for plaintext or private
      keys) or as LockedBuffers (for ciphertexts of sensitive data).

      The [`config`](pkg/config/README.md) package includes new methods
      for handling configuration file data as Enclaves and LockedBuffers
      to try to reduce the amount of confidential data visible in the
      process.

      The changes are ongoing and, in addition to adding a layer of data
      security to `cordial`, an added benefit is the interception of
      memory use errors etc. If you see errors, panic etc. please report
      them as a [github
      issue](https://github.com/ITRS-Group/cordial/issues)

    * A number of the previous package APIs have undergone review and
      changed as needed. In particular the
      [`config`](pkg/config/README.md) API has been through the wringer
      and if you have any code that relies on it from v1.4 or earlier
      then it will require changes. There are new functions, which is to
      be expected, but also some existing ones have been renamed or had
      their argument signatures changed. Please review the documentation
      to see what the methods and functions have become.

    * Credentials support. There is both general purpose and
      [`geneos`](tools/geneos/README.md) specific support for the local
      storage of credentials. Passwords and other secrets "at rest" are
      stored in Geneos AES256 format using a key file that is initial
      auto-generated. To decode these passwords you must have both the
      key file (which is by default only user readable) and the
      credentials file. There should be support for other credentials
      types, such as OAuth style client secrets and tokens, in future
      releases. The _username_ and the _domain_ that the credentials
      apply to are not encrypted, by design. This is however subject to
      change in a future release.

      Credentials currently works with a free-text domain that matches a
      destination using a "longest match wins" search, e.g. for a URL
      this may be a full or partial domain name, and for Geneos
      component authentication, e.g. the REST command API, the domain is
      in the form `gateway:NAME`. Others will be added later, probably
      including TLS certificates and keys as well as SSH password and
      private keys.

    * Releases now include selected binaries with a semantic version
      suffix. The programs in `cordial` use the base name of the binary
      as a key to select which configuration files to load, so that
      renaming the binary will result in a different set of
      configuration file being used, automatically.

      To make life simpler, any version suffix is automatically stripped
      if, and only if, it matches the one used to build the binary. This
      means you can now download `geneos-v.1.5.0` and use it without
      having to rename it (useful for initial testing of new releases).

  * [`tools/geneos`](tools/geneos/README.md):

    * Extensive documentation restructuring and rewriting. This is still
      work in progress but largely complet. Built-in help text (shown
      with the `help` command or the `--help`/`-h` option) should now
      align much more closely with real functionality and the online
      documentation is now almost completely built from the same source.

    * Addition of _subsystems_ to group commands.

    * Move `aes` and `tls` command sources to their subsystems.

    * Add `host` and `package` subsystems and create aliases for the
      original commands, e.g.
      * `add host` becomes `host add`
      * `install` becomes `package install`
      * etc.

    * The `set user`, `show user` etc. commands are now under single
      `config` subystem, e.g. `geneos config set mykey=value`

    * The `set global` and related commands have been deprecated.

    * The new `package` subsystem command pulls all Geneos release
      management into one place

    * New `login` and `logout` commands to manage credentials.

    * New `ca3` and `floating` components for Collection Agent 3 and Floating
      Netprobes

  * [`tools/dv2email`](tools/dv2email/README.md):

    * This new utility can be run as a Geneos Action or Effect to
      capture one or more Dataviews and send as an email. The
      configuration is extensive and the layout and contents are
      completely configurable through the use of Go templates.

### v1.5.0 - Bug Fixes

  * [`tools/geneos`](tools/geneos/README.md):

    * Version checking of local release archives was broken because of
      overloading of a common function. This is now split and checking
      should work once again.

    * Most reported issues on github have been fixed.

### v1.5.0 - To Do

  * Documentation needs more work and refinement. The built-in help for
    almost all commands is now up-to-date but the `init` and `tls`
    subsystems need to be reviewed further and completed. This should be
    in a patch release soon.

  * [`tools/geneos`](tools/geneos/README.md):

    * Local storage of encrypted passwords for remote SSH access needs
      documenting

---

## Version v1.4.4 - 2023/04/12

* Fixes

  * New `Default` expand option should NOT itself default to `nil`

---

## Version v1.4.3 - 2023/04/12

* Fixes

  * tools/geneos: fix `ps` not showing open ports on systems with IPv6 enabled
  * tools/geneos: make `ls` and `ps` command flags more consistent
  * tools/geneos: add an -el8 runtime to docker images when built
  * tools/geneos: fix RHEL8/Centos8 download support for localhost using new SetStringMapString() method
  * pkg/config: add SetStringMapString() methods to support settings maps (which viper doesn't support until you write a file out and read it back)
  * tools/geneos: adjust the way we choose package version, convert "-el8" to "+el8" to satisfy semver ordering
  * tools/geneos: package version number are now prefixes only

* Changes

  * tools/geneos: add `-f` flag to `ps` command to show open files. formatting subject to change for now.
  * tools/geneos: add a `update ls` command to show available package versions
  * pkg/config: added more ExpandOptions and support more Get* functions such as GetInt
  * pkg/geneos: added more Geneos XML config support, specifically Sampler Schemas and Standardised Formatting
  * libraries/libemail: added initial msTeams notification function

---

## Version v1.4.2 - 2022/12/21

* Fixes

  * tools/geneos: fix `update` to only optional restart (`-R`) the component type given
  * tools/geneos: check RHEL8 download in a case independent way - fixes remotes
  * tools/geneos: create user config directory for remote hosts in case of old location for main config
  * tools/geneos: `install` should error out is passed `@host` instead of `-H host`
  * tools/geneos: ssh known hosts handling improved (for mixed IP / hostnames)
  * tools/geneos: remote hosts with IP names are now renamed `A-B-C-D` to avoid issues with viper names

---

## Version v1.4.1 - 2022/12/19

* Fixes

  * tools/geneos: check return from user.Current() as it can fail (but shouldn't)
  * tools/geneos: numerous fixes for logic around handling of remote hosts
  * tools/geneos: fix remote host naming to be work with capitalisations
  * tools/geneos: actually load SSH private key files, if available
  * tools/geneos: re-order SSH HostKeyAlgorithms so that, bizarrely, IP based remotes work
  * tools/geneos: better handling of instance config aliases when writing config files
  * tools/geneos: fixes to unset to ignore values that may be passed in with keys to unset
  * tools/geneos: refactor CopyInstance() to preserve ports, other details
  * build: create static executables, using alpine, and a centos 7 compatible libemail.so
  * tools/geneos: add the beginnings of support for YAML instance config files. not enabled yet.
  * tools/geneos: fix crash when importing to common directories of components without the component name
  * tools/geneos: fix fileagent support by adding implicit imports with side-effects for all component packages
  * tools/geneos: skip failed permissions on /proc/*/fd - let 'ps' work for restricted processes
  * tools/geneos: fix update-during-install support, add --force flag for this too
  * tools/geneos: fix logic to match latest packages when major number changes

* Changes

  * tools/geneos: clean-up various comments, refactor methods, add license/copyright notices to many files
  * pkg/config: Add an options `expr` prefix to expansion items which supports <https://pkg.go.dev/github.com/maja42/goval> syntax
  * pkg/config: API change: Add options to the config expansion functions rather than just lookup maps
  * tools/geneos: add SSH password support for remote hosts
  * tools/geneos: support embedded SSH passwords in hosts config, using new 'set host' sub-command
  * tools/geneos: support additional SSH private key files per host via 'set host sshkeys=X,Y' sub-command
  * tools/geneos: begin implementation of support for YAML config files via 'configtype' user setting
  * pkg/geneos: add EnvironmentRef and fix periodStartTime attribute

* Other

  * tools/geneos: ongoing documentation and command help usage updates
  * tools/geneos: update README.md with more information about instance configuration files and their values (@gvastel)

---

## Version v1.3.2 - 2022/11/02

* Fixes

  * tools/geneos: fix running as root (or via sudo) and creation of config directories and file ownerships
  * tools/geneos: fix creation of full user config directories when running 'set user'

---

## Version v1.3.1 - 2022/11/01

* Fixes

  * tools/geneos: chown files and directories creates when run as root
  * tools/geneos: ensure plain 'init' creates all components dirs

---

## Version v1.3.0 - 2022/10/25

* Changes

  * PagerDuty integration
  * Merged ServiceNow integration, single binary build
  * tools/geneos: add instance protection against stop (and related) or delete commands
  * tools/geneos: support legacy command through emulating `*ctl` named commands
  * tools/geneos: allow remote operations without local directories

* Fixes

  * tools/geneos: fix logic around creating user default AES keyfiles and directory permissions
  * tools/geneos: round certificate expiry to midnight
  * tools/geneos: round tls remaining column to seconds correctly
  * tools/geneos: fix webserver command build typo. now webserver starts correctly
  * libemail: fix default _SUBJECT handling
  * tools/geneos: split over complex 'init' command into sub-commands
  * updated command usage information and reordered various internal function calls
  * tools/geneos: add password verify to aes encode and a --once flag to override
  * tools/geneos: add local JoinSlash and Dir to use Linux paths on Windows builds
  * tools/geneos: fix ssh-agent support on windows
  * tools/geneos: build on windows
  * integrations: Add PagerDuty integration
  * Integrations: Merge ServiceNow binaries into one
  * tools/geneos: change internal remote Stat() API

---

## Version v1.2.1 - 2022/10/11

Final release after numerous small fixes.

---

## Version v1.2.1-rc3 - 2022/10/07

* Fixes

  * `geneos` command fixes:
    * Fixed `init` download credential handling
    * Fixes JSON output format from `ls` commands
    * Local-only installs now work again (including default "latest" support)

  * Security
    * Updated Labstack Echo to 4.9.0 to address security advisory
      [CVE-2022-40083](https://nvd.nist.gov/vuln/detail/CVE-2022-40083).
      To best of our knowledge this particular set of features was never
      used in this package.

* Additional features and improvements

  * `geneos` command improvements:
    * Added `--raw` to `show` to not output expanded configuration values
    * Many improvements and changes to the new `aes` sub-commands. Please see [documentation](tools/geneos/README.md) for details
    * Removed built-in opaquing of credentials in output in favour of new `${enc:...}` support

  * `libemail.so` gets direct passwords back, with ExpandString support. See [documentation](libraries/libemail/README.md) for details

  * General package improvements
    * Enhanced `OpenLocalFileOrURL` to support `~/` paths
    * Enhanced `ExpandString` to support direct file paths and updates package docs further

---

## Version v1.2.1-rc1 - 2022/09/28

* Fixes

  * `geneos` instance configuration files now have expansion applied to
    string-like values. This means, for example, that changing the
    `version` of an instance from `active_prod` will correctly be
    reflected in the executable path and library paths. Previously these
    needed to be manually changed. Please note that existing instance
    configuration files will NOT be updated and will require editing.
    You can go from:

        "program": ".../packages/gateway/active_prod/gateway2.linux_64",

    to

        "program": "${config:install}/${config:version}/${config:binary}",

  For a complete list of supported expansions see `ExpandString()` in the [`config`](pkg/config/README.md) package.

* Additional features and improvements

  * `ExpandString()` was enhanced to add a `config:` prefix so that
    configurations with a flat structure, i.e. no "." in names, could be
    referenced.
  * To support the changes above in instance configurations a new method
    was added - `ExpandAllSettings()` - and the `geneos show` command
    enhanced to display both expanded and raw configurations via the new
    `--raw` flag.
  * Additional configuration item support in the
    [`geneos`](pkg/geneos/README.md) package

---

## Version v1.2.0-rc2 - 2022/09/26

* Fixes found during testing

  * Removed support for `$var` format expansion, now it's `${var}` only.
    This prevents configuration issues when, for example, plain text
    passwords contain dollar signs. The documented workaround if you
    need to include literal `${` in a configuration value still applies.

* Additional features and improvements

  * Added command `geneos aes update`. This may still be renamed before final release to `geneos aes import` depending on feedback.
  * Improvements to `geneos aes new`
  * Improvements, clarification to package and function documentation
  * Code clean-up and refactor to make some internals more understandable and to remove code duplication

---

## Version v1.2.0-rc1 - 2022/09/21

* Breaking Changes

  There are quite a lot of changes to the various components and
  packages since the original v1.0.0. Given that almost no-one outside
  the components contained in the repo itself is using the public
  package APIs I have broken the rules around semantic versioning and
  changed parts of the API.

* Highlights

  * Package changes
    * **Breaking changes**: Geneos `api` and `api-streams` XML-RPC supporting packages have had a big clean-up to make them easier to use
    * New `config` package to overlay `viper` with support for value expansion and crypto convenience functions
    * New `geneos` package to aid in the construction of XML configurations for Gateway and Netprobe. This is work in progress.
    * New `commands` package to provide the start of support for REST API Commands to the Gateway. This is work in progress.
    * New `xpath` package to work with the above and also the base for the `snapshot` command below. This is also work in progress.
    * New `cordial` package that initially carries a version constant.
    * New `process` package, providing a way to Daemon()ise callers on both Linux and Windows.
    * **Deprecation Notice**: The `logger` package will probably be removed as it was a stop-gap and is slowly being replaced with `zerolog`
  * Addition of the following commands to `tools/geneos`:
    * `aes` - Manage Geneos key-files and encoding/decoding of values
    * `snapshot` - Take dataview snapshots directly from the command line (requires GA5.14+)
  * ServiceNow integration updates
    * Configuration support is now direct with `config` above, allowing full value expansions support, including encoded credentials.
  * Logging changes
    * The logging in `tools/geneos` has been migrated to `zerolog` from the internal `logger` for a more flexible package. This will be further rolled-out to other parts of the repo in time.

---

## Version v1.0.0 - 2022/06/14

* First Release
