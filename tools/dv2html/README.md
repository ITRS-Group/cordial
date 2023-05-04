# dv2html - Dataview To HTML

`dv2html` is a lightweight program to capture and convert Geneos
Dataviews to HTML suitable for inclusion in email and other messaging
systems.

## Usage

```bash
dv2html get --url https://mygateway.local:7038 -k --rows 'core*' --css http://server.local/css/mycss.css '//dataview[(@name="DataviewOne")]'
```

HTML template, default or input
