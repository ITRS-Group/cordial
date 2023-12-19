# `files2dv` - Files to Dataview(s)

When FKM, FTM, Statetracker or `grep` isn't quite enough... try `files2dv`. 

> ❗ This documentation is incomplete as the full configuration file format has not yet settled.


## Modes of Operation

* Toolkit Mode

    The program can run in single-shot mode and output a Geneos Toolkit compatible CSV of the selected Dataview. You can select which configured dataview using command line options, otherwise the first dataview defined in the configuration will be used.

* Push Mode - ⚠ This mode is not yet implemented

    The program can run as a background process, examining files and pushing data into Geneos using one of the two available Netprobe APIs, either REST or XML-RPC. The choice of API is governed by local requirements and limitations which are discussed below.

## File Modes

* `file`

    The `file` mode is the default. The program creates one row per file in the dataview. Each row can contain data about the file and the file contents plus a summary of the file status.

* `info`

    The `info` mode creates one row per file. Each row can contain metadata about matching files.

* `line` - ⚠ This is not yet implemented

    The `line` mode creates one row per matching line for all files. Data extracted from each line can be presented in columns.

### File Mode

In `file` mode the program will scan the given paths for files and then process the contents of each file in order to build a single row of data per file. If a path does not result in any matches then a row is created to show no matches unless this is ignored. Each file is processes, line by line, and for each column the program will look for a match and set the column contents based on that. Also, if no matches are found by the end of the file (or the line limit) then an optional failure string can be used instead.

### Info Mode


## Files

Each dataview has a list of paths to search for files. Each path entry can use wildcards (basic file wildcards and not regular expressions) supported by the Go filepath.Match method. Depending on the File Mode for the dataview and the configuration options, if a path entry matches no files it may or may not be reported as `NOT_FOUND`.



