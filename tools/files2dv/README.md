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

In `file` mode the program will scan the given paths for files and then scan the contents of each file in order to build a single row of data for each file. If a path does not result in any matches then a status row is created to show no matches unless the configuration value `ignore-file-errors` includes `match`.

Each file is scanned, line by line, and for each unfilled column the program will look for a match and set the column contents based on that. Once a column has a value it will not be updated further. If no matches are found by the end of the file (or the line limit) then an optional failure string can be used instead.

### Info Mode

In `info` mode the program will scan the given paths and add a row for each file found. 

### Configuration Variables

The following variables are available for each path in all modes:

| Variable      | Description |
|---------------|-------------|
| `${fullpath}` | The full (absolute) path to the file, or the underlying path if no files match. This is recommended for the first column (the Geneos row name) as it should always be unique with no duplicates. The full path is obtained using the Go `filepath.Abs()` function and if there is an error processing the path then `${fullpath}` is set to the same as the `${path}`. |
| `${path}`     | The path to the file, or the underlying path if there are no matches. |
| `${status}`   | The status of the file, which should be `OK` when all is well. The other built-in values are `NO_MATCH`, `NOT_FOUND`, `ACCESS_DENIED`, `INVALID` and `UNKNOWN_ERROR`.<br><br>In `file` mode the `${status}` value can also be set by the configuration item `on-fail.status` which is triggered is any column contains a `fail` key and the match fails. |
| `${pattern}`  | This is the original path/pattern used for this row. When multiple files match a path/pattern then the `${path}` and `${fullpath}` are set based on the underlying file, while `${pattern}` is the unchanged value in the configuration file. |
| `${filename}` | The `${filename}` is the base file name for the row. It may be empty if the path contains wildcard pattern(s) and does not match a file. If the path does not contain wildcard pattern(s) then `${filename}` will contain a value. |

The variables below only have values when files are found:

| Variable      | Description |
|---------------|-------------|
| `${type}` | The file type. One of `file`, `symlink`, `directory` or `other` |
| `${size}` | The file size, in bytes |
| `${modtime}` | The last modification time, in ISO8601 format. The time-zone will depend on the local time-zone of the server |
| `${mode}` | The file mode in POSIX/UNIX format |
| `${device}` | The filesystem ID based on the local OS value. On Linux this is a decimal value while on Windows this is in hexadeciaml, to align with local conventions |

The variables below only have values when files are found on a Linux system:

| Variable      | Description |
|---------------|-------------|
|  `${uid}` / `${gid}` | The numeric values for the UID and GID of the file |
| `${user}` / `${group}` | The human-readable user and group owners of the file. If these cannot be found then these are set to the UID and GID, as above |
| `${inode}` | The inode of the file, in decimal format |

The variables below only have values when files are found on a Windows system:

| Variable      | Description |
|---------------|-------------|
| `${sid}` | The SID of the file owner |
| `${owner}` | The account name for the owner of the file in `DOMAIN\USERNAME` format. If the account name cannot be found then the SID is used. |
| `${index}` | The 64-bit Windows File Index ID in hexadecimal format |


## Files

Each dataview has a list of paths to search for files. Each path entry can use wildcards (basic file wildcards and not regular expressions) supported by the Go filepath.Match method. Depending on the File Mode for the dataview and the configuration options, if a path entry matches no files it may or may not be reported as `NOT_FOUND`.



