# `gdna fetch`

The `gdna fetch` command collects a new set of license token data from the configured sources.

The fetch command will normally work through the configured license usage sources and load the transformed data into the SQLite database. If the program configuration has been set, usually during report development, to make temporary tables actually not temporary then these are not updated during fetching of data. This behaviour can be changed using the `--post-process`/`-p` flag, which executes the routines required to create and update these intermediate tables. Another flags, `--time`/`-T`, is also for diagnostics and development of reports and this sets the timestamp of data imported from files to the current time rather than the modification time of the file. Note that this flag only affects plain files sources from the `licd` TCP endpoint and not reporting summary files, which have a timestamp in the filename.

