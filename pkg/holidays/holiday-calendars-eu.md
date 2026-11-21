# Europe including GB (EU) holiday calendars

> [!IMPORTANT]
>
> These calendars were built with AI/LLM assistance and have not been validated by humans. Verify dates and suitability before use, and review the [refresh runbook](RUNBOOK.md).

Snapshot gathered **2026-11-20**. See the [overall holiday calendar guide](README.md) for date formats, coverage limits, source-confidence grades and Geneos scheduling examples.

Gateway include: `holidays.layer0.eu.xml`. Under **Layer 0 - Holidays**, the **Layer 0 - EU - Country** groups hold the named calendars.

EU is a geographic include label here, not a statement of European Union membership: the United Kingdom is included. Euronext Amsterdam and Paris share one named exchange calendar, documented under Netherlands and referenced from France. England and Wales bank holidays match the published LSE full-day closure horizon in this snapshot, but that equivalence does not cover half-days or trade-reporting exceptions.

## Calendars

| Country or territory | Calendar |
| --- | --- |
| France | [`Holidays - FR Metropolitan - Public`](#holidays---fr-metropolitan---public) |
| Germany | [`Holidays - DE Hesse - Public`](#holidays---de-hesse---public) |
| Germany | [`Holidays - DE Xetra and Frankfurt - Exchange`](#holidays---de-xetra-and-frankfurt---exchange) |
| Netherlands | [`Holidays - NL - Public`](#holidays---nl---public) |
| Netherlands | [`Holidays - Euronext Amsterdam and Paris - Exchange`](#holidays---euronext-amsterdam-and-paris---exchange) |
| United Kingdom | [`Holidays - GB England and Wales - Public`](#holidays---gb-england-and-wales---public) |

## France

### Public holidays

#### `Holidays - FR Metropolitan - Public`

Centre: Paris, France.

Recorded source format: Official government JSON calendar API. Source confidence: High; metropolitan jurisdiction and coverage.

Timezone: `Europe/Paris`. Source: [French government calendar API, metropolitan France](https://calendrier.api.gouv.fr/jours-feries/metropole.json).

**Annually recurring**

| Date (DD-MM) | Holiday / closure |
| --- | --- |
| 01-01 | New Year's Day |
| 01-05 | Labour Day |
| 08-05 | Victory Day |
| 14-07 | National Day |
| 15-08 | Assumption Day |
| 01-11 | All Saints' Day |
| 11-11 | Armistice Day |
| 25-12 | Christmas Day |

| Year | Date (DD-MM) | Holiday / closure |
| --- | --- | --- |
| 2026 | 06-04 | Easter Monday |
| 2026 | 14-05 | Ascension Day |
| 2026 | 25-05 | Whit Monday |
| 2027 | 29-03 | Easter Monday |
| 2027 | 06-05 | Ascension Day |
| 2027 | 17-05 | Whit Monday |
| 2028 | 17-04 | Easter Monday |
| 2028 | 25-05 | Ascension Day |
| 2028 | 05-06 | Whit Monday |
| 2029 | 02-04 | Easter Monday |
| 2029 | 10-05 | Ascension Day |
| 2029 | 21-05 | Whit Monday |
| 2030 | 22-04 | Easter Monday |
| 2030 | 30-05 | Ascension Day |
| 2030 | 10-06 | Whit Monday |
| 2031 | 14-04 | Easter Monday |
| 2031 | 22-05 | Ascension Day |
| 2031 | 02-06 | Whit Monday |

The shared [Euronext calendar](#holidays---euronext-amsterdam-and-paris---exchange)
under Netherlands also covers Paris.

## Germany

### Public holidays

#### `Holidays - DE Hesse - Public`

Centre: Frankfurt, Germany.

Recorded source format: Official HTML statutory guidance. Source confidence: Medium; date-rule interpretation.

Timezone: `Europe/Berlin`. Source: [Hesse Ministry of the Interior](https://innen.hessen.de/buerger-staat/feiertage).

**Annually recurring**

| Date (DD-MM) | Holiday / closure |
| --- | --- |
| 01-01 | New Year's Day |
| 01-05 | Labour Day |
| 03-10 | German Unity Day |
| 25-12 | Christmas Day |
| 26-12 | Second Christmas Day |

| Year | Date (DD-MM) | Holiday / closure |
| --- | --- | --- |
| 2026 | 03-04 | Good Friday |
| 2026 | 06-04 | Easter Monday |
| 2026 | 14-05 | Ascension Day |
| 2026 | 25-05 | Whit Monday |
| 2026 | 04-06 | Corpus Christi |

### Exchange closures

#### `Holidays - DE Xetra and Frankfurt - Exchange`

Centre: Frankfurt, Germany.

Recorded source format: Official human-readable exchange schedule. Source confidence: Medium; venue-specific exceptions.

Timezone: `Europe/Berlin`. Source: [Deutsche Börse trading calendar](https://www.cashmarket.deutsche-boerse.com/cash-en/trading/trading-calendar-and-trading-hours).

**Annually recurring**

| Date (DD-MM) | Holiday / closure |
| --- | --- |
| 01-01 | New Year's Day |
| 01-05 | Labour Day |
| 24-12 | Christmas Eve |
| 25-12 | Christmas Day |
| 26-12 | Second Christmas Day |
| 31-12 | New Year's Eve |

| Year | Date (DD-MM) | Holiday / closure |
| --- | --- | --- |
| 2026 | 03-04 | Good Friday |
| 2026 | 06-04 | Easter Monday |
| 2027 | 26-03 | Good Friday |
| 2027 | 29-03 | Easter Monday |
| 2028 | 14-04 | Good Friday |
| 2028 | 17-04 | Easter Monday |
| 2029 | 30-03 | Good Friday |
| 2029 | 02-04 | Easter Monday |
| 2030 | 19-04 | Good Friday |
| 2030 | 22-04 | Easter Monday |
| 2031 | 11-04 | Good Friday |
| 2031 | 14-04 | Easter Monday |

## Netherlands

### Public holidays

#### `Holidays - NL - Public`

Centre: Amsterdam, Netherlands.

Recorded source format: Official HTML public-holiday guidance. Source confidence: Medium; dates and scope.

Timezone: `Europe/Amsterdam`. Source: [Government of the Netherlands](https://www.government.nl/documents/question-and-answer/work/public-holidays-in-the-netherlands).

**Annually recurring**

| Date (DD-MM) | Holiday / closure |
| --- | --- |
| 01-01 | New Year's Day |
| 05-05 | Liberation Day |
| 25-12 | Christmas Day |
| 26-12 | Boxing Day |

| Year | Date (DD-MM) | Holiday / closure |
| --- | --- | --- |
| 2026 | 03-04 | Good Friday |
| 2026 | 05-04 | Easter Sunday |
| 2026 | 06-04 | Easter Monday |
| 2026 | 27-04 | King's Day |
| 2026 | 14-05 | Ascension Day |
| 2026 | 24-05 | Whit Sunday |
| 2026 | 25-05 | Whit Monday |
| 2027 | 26-03 | Good Friday |
| 2027 | 28-03 | Easter Sunday |
| 2027 | 29-03 | Easter Monday |
| 2027 | 27-04 | King's Day |
| 2027 | 06-05 | Ascension Day |
| 2027 | 16-05 | Whit Sunday |
| 2027 | 17-05 | Whit Monday |

### Exchange closures

#### `Holidays - Euronext Amsterdam and Paris - Exchange`

Centre: Amsterdam, Netherlands.

Recorded source format: Official human-readable schedules linked from an HTML page. Source confidence: Medium; market scope and closure footnotes.

Timezone: `Europe/Amsterdam` (same local clock as Paris). Source:
[Euronext cash and derivatives trading calendars](https://www.euronext.com/en/trading/trading-hours-holidays).

| Year | Date (DD-MM) | Holiday / closure |
| --- | --- | --- |
| 2026 | 01-01 | New Year's Day |
| 2026 | 03-04 | Good Friday |
| 2026 | 06-04 | Easter Monday |
| 2026 | 01-05 | Labour Day |
| 2026 | 25-12 | Christmas Day |
| 2027 | 01-01 | New Year's Day |
| 2027 | 26-03 | Good Friday |
| 2027 | 29-03 | Easter Monday |

## United Kingdom

### Public holidays

#### `Holidays - GB England and Wales - Public`

Centre: London, United Kingdom.

Recorded source format: Official GOV.UK JSON feed; LSE HTML applicability cross-check. Source confidence: High for bank-holiday dates, Medium for exchange applicability.

Timezone: `Europe/London`. Source: [GOV.UK bank holiday feed](https://www.gov.uk/bank-holidays.json).

**Annually recurring:** None on a fixed date. The calendar follows the
published bank holiday dates. For example, Christmas Day is observed on
December 27 in 2027, so a yearless `25/12` would incorrectly add an
override on December 25 as well.

| Year | Date (DD-MM) | Holiday / closure |
| --- | --- | --- |
| 2026 | 01-01 | New Year's Day |
| 2026 | 03-04 | Good Friday |
| 2026 | 06-04 | Easter Monday |
| 2026 | 04-05 | Early May bank holiday |
| 2026 | 25-05 | Spring bank holiday |
| 2026 | 31-08 | Summer bank holiday |
| 2026 | 25-12 | Christmas Day |
| 2026 | 28-12 | Boxing Day (substitute) |
| 2027 | 01-01 | New Year's Day |
| 2027 | 26-03 | Good Friday |
| 2027 | 29-03 | Easter Monday |
| 2027 | 03-05 | Early May bank holiday |
| 2027 | 31-05 | Spring bank holiday |
| 2027 | 30-08 | Summer bank holiday |
| 2027 | 27-12 | Christmas Day (substitute) |
| 2027 | 28-12 | Boxing Day (substitute) |
| 2028 | 03-01 | New Year's Day (substitute) |
| 2028 | 14-04 | Good Friday |
| 2028 | 17-04 | Easter Monday |
| 2028 | 01-05 | Early May bank holiday |
| 2028 | 29-05 | Spring bank holiday |
| 2028 | 28-08 | Summer bank holiday |
| 2028 | 25-12 | Christmas Day |
| 2028 | 26-12 | Boxing Day |

London Stock Exchange on-exchange non-trading days match this list in the
published horizon, so reuse the public calendar. The [LSE business days
table](https://www.londonstockexchange.com/en/resources/markets-products/business-days)
also specifies half-days and trade-reporting exceptions that are not encoded.
