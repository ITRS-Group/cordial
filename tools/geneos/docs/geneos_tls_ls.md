# `geneos tls ls`

List certificates

```text
geneos tls ls [flags] [TYPE] [NAME...]
```

List certificates and their details. The root and signing certs are only
shown if the `--all`/`-a` flag is given. A list with more details can be
seen with the `--long`/`-l` flag, otherwise options are the same as for
the main ls command.
### Options

```text
  -a, --all      Show all certs, including global and signing certs
  -l, --long     Long output
  -j, --json     Output JSON
  -i, --pretty   Output indented JSON
  -c, --csv      Output CSV
```

## SEE ALSO

* [geneos tls](geneos_tls.md)	 - TLS Certificate Operations
