# `gdna start`

Use `gdna start` to start a background process that acquires, process and reports data as well as being able to optionally send email reports on a schedule.


```text
gdna start
```

### Options

```text
  -D, --daemon              Daemonise the process
  -1, --once                Run once and exit
  -O, --on-start            Run immediately on start-up, then follow schedule
  -E, --on-start-email      Run immediately on start-up, send email report, then follow schedule
  -r, --reports string      Run only matching (file globbing style) reports
  -H, --hostname hostname   Connect to netprobe at hostname (default "localhost")
  -P, --port port           Connect to netprobe on port (default 7036)
  -S, --secure              Use TLS connection to Netprobe
  -k, --skip-verify         Skip certificate verification for Netprobe connections
  -e, --entity Entity       Send reports to Managed Entity (default "GDNA")
  -s, --sampler Sampler     Send reports to Sampler (default "GDNA")
  -R, --reset               Reset/Delete configured Dataviews on first run
```

## SEE ALSO

* [gdna](gdna.md)	 - Process Geneos License Usage Data
