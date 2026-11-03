# Getting Started with the `geneos` Command

Use this guide to install the `geneos` command, connect it to an existing installation, or deploy a demo environment.

## Install

### Download a pre-built binary

For Linux x86-64, download the latest binary to a user-owned directory such as `~/.local/bin` or `~/bin` and add it to your `PATH`:

```bash
mkdir -p "$HOME/.local/bin"
curl --fail --location --output "$HOME/.local/bin/geneos" https://github.com/ITRS-Group/cordial/releases/latest/download/geneos
chmod 0755 "$HOME/.local/bin/geneos"
export PATH="$HOME/.local/bin:$PATH"
geneos version
```

Add the `PATH` update to your shell's startup file to keep it in future sessions. Use `~/bin` instead if that is your preferred user binary directory.

### Install from source

Install the latest released command with Go:

```bash
go install github.com/itrs-group/cordial/tools/geneos@latest
```

Ensure `$HOME/go/bin` is on your `PATH`, then verify the installation:

```bash
geneos version
```

## First-time setup

### Use an existing installation

Point the command to the Geneos root directory, which contains `packages`, `gateway`, and related directories:

```bash
geneos config set geneos=/opt/itrs
geneos list
geneos ps
```

### Deploy a demo environment

The demo profile downloads Geneos releases, so you need an ITRS download account. Supply your account name with `-u`; the command prompts for your password:

```bash
geneos init demo -u user@example.com
```

Follow the prompts to select an installation directory. After deployment, check the configured and running instances:

```bash
geneos list
geneos ps
```

If the instances are not running, start the profile with `geneos start profile demo`.

## Common workflows

### Review the environment

```bash
geneos list
geneos ps
geneos logs --lines 20
```

### Start and stop instances

```bash
geneos start gateway MyGateway
geneos stop netprobe 'prod*'
geneos restart gateway MyGateway
```

### Inspect an instance

```bash
geneos show gateway MyGateway
geneos command gateway MyGateway
geneos home gateway MyGateway
```

### Add components

```bash
geneos add gateway MyGateway -p 7040
geneos start gateway MyGateway -l
geneos add netprobe MyProbe -g localhost:7039
geneos start netprobe MyProbe -l
```

### Manage software releases

```bash
geneos package list
geneos package install gateway
geneos package update
```

## Quick reference

| Task | Command |
| --- | --- |
| List configured instances | `geneos list` |
| Show running instances | `geneos ps` |
| Start or stop instances | `geneos start` / `geneos stop` |
| Follow logs | `geneos logs --follow` |
| Add a Gateway | `geneos add gateway NAME -p PORT` |
| Add a Netprobe | `geneos add netprobe NAME -g HOST:PORT` |
| Get command help | `geneos help COMMAND` |

## Troubleshooting

### Command not found

Check that the installation directory is on your `PATH`:

```bash
command -v geneos
```

### Geneos installation not found

Set the Geneos root to the directory containing `packages`, `gateway`, and related component directories:

```bash
geneos config set geneos=/opt/itrs
```

### Instance will not start

Check its status, logs, and configuration:

```bash
geneos ps TYPE NAME
geneos logs TYPE NAME
geneos show TYPE NAME
```

For further assistance, run `geneos help` or see the [User Guide](USER_GUIDE.md) and its [troubleshooting section](USER_GUIDE.md#troubleshooting).
