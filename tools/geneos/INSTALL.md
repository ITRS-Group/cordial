# `geneos`

## Installation

### Download the binary

You can download a pre-built binary version (for Linux on amd64 only) from [this link](https://github.com/itrs-group/cordial/releases/latest/download/geneos) or like this:

```bash
curl -OL https://github.com/itrs-group/cordial/releases/latest/download/geneos
chmod 555 geneos
sudo mv geneos /usr/local/bin/
```

### Build from source

To build from source you must have [Go](https://go.dev) 1.21.5 or later installed.

#### One line installation

```bash
go install github.com/itrs-group/cordial/tools/geneos@latest
```

Make sure that the `geneos` program is in your normal `PATH` - or that $HOME/go/bin is if you used the method above - to make things simpler.

#### Download from github and build manually

Make sure you do not have an existing file or directory called `cordial` and then:

```bash
github clone https://github.com/itrs-group/cordial.git
cd cordial/cmd/geneos
go build
sudo mv geneos /usr/local/bin
```
