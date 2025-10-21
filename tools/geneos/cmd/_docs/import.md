Import each `SOURCE` to instances. With the `--common`/`-c` option imports are saved in the component sub-directory with the name passed as an argument. See examples below.

The `SOURCE` can be the path to a local file, a URL or '-' for `STDIN`. `SOURCE` may not be a directory.

If `SOURCE` is a file in the current directory then it must be prefixed with `"./"` to avoid being seen as an instance NAME to search for. Any file path with a directory separator already present does not need this precaution. The program will read from `STDIN` if the `SOURCE` '-' is given but this can only be used once and a destination DEST must be defined.

If `DEST` is given with a `SOURCE` then it must either be a plain file name or a descending relative path. An absolute or ascending path is an error.

Without an explicit `DEST` for the destination file only the base name of the `SOURCE` is used. If `SOURCE` is a URL then the file name for the resource from the remote web server is preferred over the last part of the URL.

If the `--common`/`-c` option is used then a TYPE must also be specified. Each component of TYPE has a base directory. That directory may contain, in addition to instances of that TYPE, a number of other directories that can be used for shared resources. These may be scripts, include files and so on. Using a TYPE `gateway` as an example and using a `--common config` option the destination for `SOURCE` would be `gateway/config`

Future releases may add support for directories and.or unarchiving of `tar.gz`/`zip` and other file archives.
