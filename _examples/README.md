# Examples

Example programs that use the `cordial` packages. This directory is a separate Go module, using the `cordial` module in the parent directory, so that the extra dependencies the examples need (such as the Docker client and OIDC libraries) are not dependencies of `cordial` itself. As the directory name starts with an underscore, the examples are also ignored by `go ./...` in the main module.

| Example | Description |
|---------|-------------|
| `api` | Geneos Netprobe samplers using the `plugins` and `streams` packages |
| `api2` | A sampler using the `api` package |
| `cliauth` | OIDC authentication from a command-line program, using a local web browser |
| `dockerlogs` | Feed Docker container logs to a Geneos API Streams plugin |

To check that the examples still build, use `make check-examples` in the repository root or, from this directory:

```bash
go vet ./...
```

To run an example, change to this directory first, for example `go run ./api -h`.
