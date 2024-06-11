# `files2dv` - Files to Dataview(s)

When FKM, FTM, Statetracker or `grep` isn't quite enough... try `files2dv`. 

> ❗ This documentation is incomplete as the full configuration file format has not yet settled.

## Modes of Operation

* Toolkit Mode

    The program can run in single-shot mode and output a Geneos Toolkit compatible CSV of the selected Dataview. You can select which of the configured dataview using command line options, otherwise the first dataview in the configuration will be used.

* Push Mode - 💡 This mode is not yet implemented

    The program can run as a background process, examining files and pushing data into Geneos using one of the two available Netprobe APIs, either REST or XML-RPC. The choice of API is governed by local requirements and limitations which are discussed below.

## File Operations

The program runs as follows:

* For each dataview in the configuration, the program iterates through the list of given paths
* Each path can contain wildcards and Geneos style `<today...>` date specifiers
* The resulting list of file paths / names are then checked to extract the configured column values
* You can limit the type of files that are included using the `types` option, which can contain `file`, `directory`, `symlink` or `other`. If you do not specify any `types` then all are included

When a column specifier does not contain a `match` entry then the column value is set using file metadata.

When no column contains a `match` option then the file contents are not accessed and only file metadata is used to construct the columns.

However, if any column specifier contains a `match` clause then the file is opened for reading and scanned, line at a time, and the first matching line in the file is used to fill in the `value` for that column, using regular expression groups to capture specific text that can be interpolated into `value`. Once the value of a column is set, it is no longer checked against further lines in the file. If all columns containing a `match` clause have had values set then the scanning of the file is considered complete and the program moves on to the next row/file.

If you specify an `ignore-lines` list of regular expressions then any lines that match any of these will be immediately skipped and the program moves on to the next line. If, after processing all the lines in the file (up to `max-lines`, if given), any columns that have not matched the given regexp then the `fail` option is used to fill in the column value.

Each line in the file is checked as many times as there are columns that have a `match` clause (and the value has not already been set by a previous match). This is in contrast to the Geneos FKM plugin that moves on to the next line in a file as soon as a match is found (unless an obscure FKM option is used). This means that `match` clauses can overlap, but also there is no way to skip to the next line once a match is found.

### Unmatched File Paths

Each dataview has a list of paths to search for files. Each path can use wildcards (basic file wildcards, aka "globs", and not regular expressions) supported by the Go filepath.Match method. Depending on the File Mode for the dataview and the configuration options, if a path entry matches no files it may or may not be reported as `NOT_FOUND`.

## Configuration Variables

The following variables are available for each file path:

| Variable      | Description |
|---------------|-------------|
| `${fullpath}` | The full (absolute) path to the file, or the underlying path text value if no files match. This is recommended for the first column (the Geneos row name) as it should always be unique with no duplicates. The full path is obtained using the Go `filepath.Abs()` function and if there is an error processing the path then `${fullpath}` is set to the same as the `${path}` |
| `${path}`     | The path to the file, or the underlying path text if there are no matches |
| `${status}`   | The status of the file, which should be `OK` when all is well. The other built-in values are `NO_MATCH`, `NOT_FOUND`, `ACCESS_DENIED`, `INVALID` and `UNKNOWN_ERROR`.<br><br>In `file` mode the `${status}` value can also be set by the configuration item `on-fail.status` which is triggered is any column containing a `fail` clause and the match fails for all lines in the file |
| `${pattern}`  | This is the original path/pattern used for this row. When multiple files match a path/pattern then the `${path}` and `${fullpath}` are set based on the underlying file, while `${pattern}` is the unchanged text value in the configuration file |
| `${filename}` | The `${filename}` is the base file name for the row. It may be empty if the path contains wildcard pattern(s) and does not match a file. If the path does not contain wildcard pattern(s) then `${filename}` will contain a value |

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




