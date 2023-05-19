Import non-instance certificates. A root certificate is one where the
subject is the same as the issuer. All other certificates are imported
as signing certs. Only the last one, if multiple are given, is used.
Private keys must be supplied, either as individual files on in the
certificate files and cannot be password protected. Only certificates
with matching private keys are imported.
