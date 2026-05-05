# `geneos tls info`

This command displays information about TLS certificates, either in local files or from remote endpoints. It can show details about the certificate chain, including verification status, expiration dates, and more.

You can pass multiple files or endpoints to check, and the output can be formatted in different ways, including a human-readable table or a format suitable for further processing.

Once flags have been parsed, any remaining arguments are treated as file paths to check. These files can be in PEM, PFX/PKCS12 or Java Keystore format. If the file is a PFX/PKCS12 or Java Keystore, you can specify the password using the `--password` flag, otherwise you will be prompted to enter it for each file. If you specify a password as a flag then all password protected files must have the same password.

The `--connect`/`-c` flag allows you to specify remote endpoints to connect to and retrieve TLS certificate information. The endpoint should be a URL or in the format `host:port` (`port` defaults to 443 if not given). You can specify multiple endpoints by using the flag multiple times. You can also give the path to a file contraining a list of endpoints (in the same format for `--connect`), one per line, using the `--connect-file` flag.

The output format is controlled by the `--format`/`-f` flag. The default is `column`, which produces a human-readable table. The `toolkit` format produces output in a format suitable for ingestion by a Geneos Toolkit sampler, with headlines for total counts and rows of data for each certificate. Other formats are `csv` and `table`. `csv` doesn't include the headline counts, and `table` is similar to `column` but with borders around the table.

By default all certificates found will be listed. To only show leaf certificates, use the `--leaf`/`-L` flag, which will process but not display CA certificates.

By default a short format is used, which includes only the most important information. If you use the `--long`/`-l` flag, additional details about each certificate will be included in the output.

For certificate verification you can specify additional root certificate files using `--roots`/`-r`. These should be in PEM format. The system's default root certificates and the geneos `ca-bundle.pem` files will also be used for verification, and these cannot be disabled, if they exist.

A certificate is considered verified if it can be chained to a trusted root certificate and is not expired. The hostname is *NOT* used for verifiation. The output includes a "Verified" column which indicates whether the certificate is verified. If a private key is present in a file and it matches a certificate then the PrivateKeyMatched colum will show `true`. In general this will only apply to local files.

The total number of certificates processed, the number of verified certificates, the number of expired certificates, and the number of certificates expiring within 30 days are included in the output headlines when using the `toolkit` format.

## Toolkit Sampler Example

Here is an example of how you might use this command in a Geneos Toolkit sampler:

Set the `Sampler Script` to:

```
/usr/local/bin/geneos tls info --format toolkit --connect-file ./cert-paths.txt --long --leaf-only
```

And then under the (lower) Advanced tab, in the Script box, set Contents to a list of endpoints (one per line) and Filename to `./cert-paths.txt`

Attach this sampler to a Managed Entity where the `geneos` program is installed (edit the path above to suit) and it should display one row for each endpoint that can be connected. Endpoints that cannot be contacted will show an error in the `CommonName` column.
