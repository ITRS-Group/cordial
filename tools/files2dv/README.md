# `files2dv` - Files to Dataview(s)

When FKM, FTM, Statetracker or `grep` isn't quite enough... try `files2dv`. 

> ❗ This documentation is incomplete as the full configuration file format has not yet settled.

## Modes of Operation

* Toolkit Mode

    The program can run in single-shot mode and output a Geneos Toolkit compatible CSV of the selected Dataview. You can select which dataview from the configuration using command line options, otherwise the first dataview defined in the configuration will be used.

* Push Mode

    > ⚠ This mode is not yet implemented.

    The program can run as a background process, examining files and pushing data into Geneos using one of the two available Netprobe APIs, either REST or XML-RPC. The choice of API is governed by local requirements and limitations which are discussed below.

## File Modes

* `file`

    The `file` mode is the default. The program creates one row per file in the dataview. Each row can contain data about the file and the file contents.

* `info`

    The `info` mode creates one row per file. Each row can contain metadata about the file without opening the file.

* `line`

    The `line` mode creates one row per matching line for all files. Data extracted from each line can be presented in columns.

    > ⚠ This is not yet implemented.

## Files

Each dataview has a list of paths to match for files. Each path entry can use wildcards (not regexps) supported by the Go filepath.Match method. Depending on the File Mode for the dataview, if a path entry matches no files it may or may not be reported as `NOT_FOUND`.

