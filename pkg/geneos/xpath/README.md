# Geneos XPath package

The parser interprets Gateway names, managed-entity attributes, sampler types, row/headline names, and cell columns as structured fields. Predicates outside those supported forms are preserved verbatim on each path component, so expressions such as `dataview[(param("Group")="Gateway Info")]` are forwarded to the Gateway through parse, JSON marshal, and format round trips.

Local `Match` filtering only evaluates structured fields. The Gateway remains responsible for evaluating preserved predicates.

