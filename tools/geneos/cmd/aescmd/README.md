# `geneos aes` Subsystem

The `aes` subsystem allows you to manage AES256 keyfiles and perform
encryption and decryption.


The `geneos aes` commands provide tools to manage Geneos AES256 key
files as [documented
here](https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/gateway_secure_passwords.htm).

In addition to the functionality built-in to Geneos as described in the
Gateway documentation these encoded password can also be included in
configuration files so that plain text passwords and other credentials
are not visible to users.

* `geneos aes new [-k KEYFILE] [-I] [TYPE] [NAME]`

  Create a new keyfile. With no arguments a new keyfile printed on
  STDOUT. If the import option (`-I`) is given then the keyfile is
  copied to the component keyfile directory (e.g.
  `gateway/gateway_shared/keyfiles`) with a name made of the CRC32
  checksum of the file and an `.aes` extension. The file is also copied
  to remote hosts and all matching instances have their keyfile
  parameters set to use this file. Any instances with an existing
  keyfile setting have that moved to `prevkeyfile`.

* `geneos aes ls [-c] [-j [-i]] [TYPE] [NAME]`

  List configured keyfiles in Geneos instances. The CRC32 column is
  provided as a visual aid to human users to identify common keyfiles.
  
  Note: If a keyfile is configured then the component - currently only
  Gateways - are started with the keyfile on the command line. This may
  cause start-up issues if the keyfile has just been added or changed
  and your Gateway is earlier than GA5.14.0 or there is an existing
  `cache/` directory in the Gateway working directory. To resolve this
  you may have to remove the `cache/` directory (use the `geneos clean`
  command with the `-F` full-clean option) or start the Gateway with a
  `-skip-cache` option which can be set with `geneos set -k
  options=-skip-cache` and so on.

* `geneos aes encode [-k KEYFILE] [-p PASSWORD] [-s SOURCE] [-e] [TYPE]
  [NAME]`

  Encode a plain text PASSWORD or SOURCE using the keyfile given or the
  keyfiles configured for all matching instances or the user's default
  keyfile. If instances share the same keyfile then the same output will
  be generated for each. If neither a string or a source path is given
  then the user is prompted to enter a password. The SOURCE can be a
  local file or a URL. The `-e` option set the output to be in
  "expandable" form, which includes the path to the keyfile used, ready
  for copying directly into configuration files that support
  ExpandString() values.

* `geneos aes decode [-e STRING] [-k KEYFILE] [-v KEYFILE] [-p PASSWORD]
  [-s SOURCE] [TYPE] [NAME]`

  Decode the ExpandString format STRING (with embedded keyfile path) or
  the encoded PASSWORD or the SOURCE using the provided keyfile (or
  previous keyfile) or using the keyfiles for matching instances or the
  user's default keyfile. The first valid UTF-8 decoded text is output
  and further processing stops. The encoded text can be prefixed with
  the Geneos `+encs+` text, which will be removed if present. The SOURCE
  can be a local file or a URL.

* `geneos aes import [-k FILE|URL|-] [-H host] [TYPE] [NAME...]`

  Import a keyfile

* `geneos aes set [-k FILE|URL|-] [-C CRC32] [-N] [TYPE] [NAME...]`

  Update the existing keyfile in use by rotating the currently
  configured keyfile to previous-keyfile. Requires GA6.x.
  