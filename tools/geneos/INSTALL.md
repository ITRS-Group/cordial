# Installing the `geneos` Command

## Install

### Download a pre-built binary

The latest pre-built binary is for Linux x86-64. On common Linux distributions such as Ubuntu, Debian, and RHEL, use a user-owned directory such as `~/.local/bin` or `~/bin` and add it to your `PATH`:

```bash
mkdir -p "$HOME/.local/bin"
curl --fail --location --output "$HOME/.local/bin/geneos" https://github.com/itrs-group/cordial/releases/latest/download/geneos
chmod 0755 "$HOME/.local/bin/geneos"
export PATH="$HOME/.local/bin:$PATH"
```

### Build from source

The current source requires [Go](https://go.dev) 1.27.3 or later.

#### Install with Go

```bash
go install github.com/itrs-group/cordial/tools/geneos@latest
```

Ensure the `geneos` executable is on your `PATH`. If you used `go install`, add `$HOME/go/bin` to your `PATH`.

#### Clone and build manually

Clone the repository, build the command from its module directory, and install the binary:

```bash
git clone https://github.com/itrs-group/cordial.git
cd cordial/tools/geneos
go build -o geneos .
mkdir -p "$HOME/.local/bin"
install -m 0755 geneos "$HOME/.local/bin/geneos"
export PATH="$HOME/.local/bin:$PATH"
```
