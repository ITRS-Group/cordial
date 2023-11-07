# Create Holiday Calendar Active Times

This is the start of a small tool to try to auto-generate holiday calendar Active Times in an include file.

It is only partially written.

The whole thing must be built and run in a container because of the Python 3.7 requirement for calling from a Go program. Do this with:

```bash
docker build . --tag=geneos-holidays
docker run -it --rm geneos-holidays
```

Then, inside the container run:

```bash
/app/holidays/holidays UK
```

Without an argument it outputs all holidays it can find.

Later versions will output an XML include file.

