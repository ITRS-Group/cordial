# `geneos tls export`

Exports either the local signer private key, certificate and root certificate suitable for importing into another cordial `geneos` installation or, if you specify either component TYPE or instance NAMEs then the private key, instance certificate and other certificates up to and including the trust root for matching instances is output instead.

The default output is to the console but you can specify a file destination using the `--dest`/`-D` option.
