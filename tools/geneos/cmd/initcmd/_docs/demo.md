The `geneos init demo` command is used to initialise a Geneos Demo environment.

Without any arguments the command initialises a directory called `geneos` under your user's home directory (unless the user's home directory ends in `geneos` in which case it uses that directly), downloads the latest release archives and creates a Gateway instance using the name `Demo Gateway` (with embedded space) as required for Demo licensing, as Netprobe and a Webserver.

If given the `--minimal`/`-M` flag then the minimal Netprobe component is deployed, which can save 300MB+ of download when being run, for example, to build a docker container.

If the release archive files required have already been downloaded then use the `--archive`/`-A` flag to indicate their location. For each component type this directory is checked for the latest release.

To fetch the releases from the ITRS download server, authentication will be required. Use the `--username`/`-u` flag to specify the user account and you will be prompted for a password.

The initial configuration file for the Gateway is built from the default templates, which are installed at the same time, and located in `gateway/templates` but this can be overridden with the `-s` option. For the Gateway component you can add include files using `-i PRIORITY:PATH` flag. This can be repeated multiple times.

Other flags inherited from the `geneos init` command can be used to control the installation.

If you are running on a Linux system that is capable of supporting the X Window System (which is anything with a desktop display, including WSL2) and you have an environment variable `DISPLAY` set, then the command will also install an Active Console for Linux. This is not started by default. This does not currently work from a docker container.

## Example Usage

You can set-up a Demo environment like this:

```bash
geneos init demo -u email@example.com
```

or, to script this, do:

```bash
export ITRS_DOWNLOAD_USERNAME=email@example.com
export ITRS_DOWNLOAD_PASSWORD=mysecret
geneos init demo
```

Here you should replace the email address with the one you registered on the ITRS Resources website and the command will prompt you for your password if not supplied. The ITRS_DOWNLOAD_PASSWORD environment variable can also use "expandable format" which is the output of the `geneos aes password` command.
