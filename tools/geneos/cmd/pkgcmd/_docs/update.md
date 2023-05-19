Update the symlink from the default base name of the package to
the best match for VERSION. The default base directory is `active_prod`
and is normally linked to the latest version of a component type in the
packages directory. VERSION can either be a semantic version style name or
(the default if not given) `latest`.

If TYPE is not supplied, all supported component types are updated to VERSION.

Update will stop all matching instances of the each type before
updating the link and starting them up again, but only if the
instance uses the same basename.

The matching of VERSION is based on directory names of the form:

`[GA]X.Y.Z`

Where X, Y, Z are each ordered in ascending numerical order. If a
directory starts `GA` it will be selected over a directory with the
same numerical versions. All other directories name formats will
result in unexpected behaviour. If multiple installed versions
match then the lexically latest match will be used. The chosen
match may be much higher than that given on the command line as
only installed packages are used in the search.

If a basename for the symlink does not already exist it will be created,
so it important to check the spelling carefully.
