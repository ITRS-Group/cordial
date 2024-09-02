# `geneos package update`

The `package update` command sets the base link for the given component `TYPE`, or all types if not given, to the latest version found in the same package directory.

Use `package list` to see which versions are installed.

💡Note: The `--install`/`-I` option has been deprecated in favour of using the `geneos package install` command directly. To install general releases use the `package install` command or, for more limited uses, you can also use the `--install`/`-I` option to download and install packages that match the options given to `update`. To make things simpler, the `--install`/`-I` option only works with predefined credentials (via `geneos login`) and the official download site, but it will use the version given with the `--version`/`-V` option below.

Alternative versions can be selected via the `--version`/`-V` option or by the first argument after options and component. The base link that is updated defaults to `active_prod` but can be set with `--base`/`-b`.

The `package update` command will create new base links given with the `--base`/`-b` option, so if you maintain multiple base links then check the spelling carefully.

Base links that are in use by protected instance are not updated without the `--force`/`-F` option. Because multiple instances of a component often share the same base link, if any instance is protected then no update is done without `--force`/`-F`.

Otherwise, by default any running instances that use the base link that is being upgraded will be restarted around the update. While not recommended you can prevent this by passing a false value to the `--restart`/`-R` option (`--restart=false`). 

```text
geneos package update [flags] [TYPE] [VERSION]
```

### Options

```text
  -V, --version string   Update to this version, defaults to latest
  -b, --base string      Base name for the symlink, defaults to active_prod (default "active_prod")
  -R, --restart          Restart all instances that may have an update applied (default true)
  -F, --force            Will also update and restart protected instances
```

## Examples

```bash
geneos package update gateway -b active_prod
geneos package update gateway -b active_dev -V 5.11
geneos package update
geneos package update netprobe --version 5.13.2

```

## SEE ALSO

* [geneos package](geneos_package.md)	 - Package Operations
