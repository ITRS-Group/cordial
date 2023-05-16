# dv2email - Dataview To HTML

`dv2email` is a lightweight program to capture and convert Geneos
Dataviews to HTML suitable for inclusion in email and other messaging
systems.

## Usage

```bash
dv2email get --url https://mygateway.local:7038 -k --rows 'core*' --css http://server.local/css/mycss.css '//dataview[(@name="DataviewOne")]'
```

The order of the columns is not fixed and so you should normally list the columns in the order you want them shown:

```bash
dv2email get --url https://mygateway.local:7038 -k --rows 'core*' --columns rowname,col1,col2 '//dataview[(@name="DataviewOne")]'
```

HTML template, default or input

Note that CSS support is limited in gmail clients (including GSuite domains) and you should adjust your template and CSS options accordingly. The option -i tries to inline the CSS but increases the size of the HTML.
