# `gdna`

The `gdna` program combines an Extract-Transform-Load (ETL) and reporting tool that collects Geneos license usage data to generate reports of license utilisation which are expressed as levels of monitoring coverage. The reports are fully configurable and are built using SQLite queries.

The data used is from either `licd` CSV reports, which are described [here](https://docs.itrsgroup.com/docs/geneos/current/administration/licence-daemon/index.html#csv-files) or, where file access to the `licd` working directory is available, License Summary reports generated by newer released of `licd`.

In normal operations - after installation, configuration and commissioning - simply use the `gdna start -D` command to run the program as a background process executing functions based on the schedules in the configuration file. Other commands allow 

The configuration file is normally located in `${HOME}/.config/geneos/gdna.yaml` but can placed anywhere using a path to the file with the `--config`/`-f` option. If there is a `gdna.defaults.yaml` file in the same directory then it is loaded first. This second file can be used, for example, to house the relatively large SQL queries for custom reports while leaving the day-to-day configuration file small and easy to manage.

The program logs various actions to a file in the working directory but this can be changed in the configuration `logging` section or with the `--logfile`/`-l` option, which is set to a dash (`-`) send logging output to STDERR, which is normally your terminal. Using this latter option with the `gdna start -D` is not possible and logging will then go nowhere. Other features of logging, such as auto-rolling log files as they grow and more, can be configured in the `gdna.yaml` file.


## Commands

| Command | Description |
|-------|-------|
| [`gdna email`](gdna_email.md)	 | Email reports |
| [`gdna fetch`](gdna_fetch.md)	 | Fetch usage data |
| [`gdna list`](gdna_list.md)	 | List available reports |
| [`gdna report`](gdna_report.md)	 | Run ad hoc report(s) |
| [`gdna restart`](gdna_restart.md)	 | Restart background GDNA process |
| [`gdna start`](gdna_start.md)	 | Start cycling though fetch, report etc. |
| [`gdna stop`](gdna_stop.md)	 | Stop background GDNA process |
| [`gdna version`](gdna_version.md)	 | Show program version |

### Options

```text
  -f, --config FILE    Use configuration file FILE
  -l, --logfile file   Write logs to file. Use '-' for console or /dev/null for none (default "docs.log")
```

## SEE ALSO
