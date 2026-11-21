# Financial centre holiday calendars

> [!IMPORTANT]
>
> These holiday calendars were built using AI/LLM-assisted collection and conversion processes and have not been validated by humans. They must be treated as unverified reference data, not as authoritative schedules. Exercise care before operational use: verify every relevant date, observance, jurisdiction, market and timezone against current official sources, and confirm that the calendar is suitable for its intended schedule.
>
> Review the [holiday calendar refresh runbook](RUNBOOK.md) for how dates are obtained, interpreted, converted and rated, including the process's limitations. The runbook describes the collection workflow but does not provide human validation.

Snapshot gathered **2026-11-20**. Public holiday and exchange calendars
are listed separately. Dates are local to the timezone shown. Tables use
`DD-MM` for day-first dates (including the US as requested), `MM-DD` where
month-first is customary in East Asia, and `YYYY-MM-DD` for Canada. The date
column identifies the format. Coverage is the current year plus up to five years ahead
(no dates after 2031-11-20 in this snapshot). A missing year is not a
statement that there are no holidays.

The lists cover full-day holidays only. Where published, observed or
compensatory dates are included as specified by the source. Normal
weekends, local or industry-specific holidays unless noted, shortened
sessions, settlement-only holidays and emergency closures are not inferred.
Review the linked sources before use and whenever the authorities issue
updates. Each Geneos calendar starts with its **Annually recurring** dates,
encoded as `DD/MM` (for example, `25/12`). Year-specific holidays and
substitute dates follow in chronological order as `DD/MM/YYYY`. An
observed date that replaces the nominal date (rather than adding to it)
is not encoded as annually recurring.

## Source confidence

These are qualitative **source-format and extraction-confidence** notes,
assessed on **2026-11-20**, not probabilities that a holiday is correct.
An official machine-readable calendar is explicitly intended for automated
use and generally reduces transcription and layout ambiguity. It can still
be stale, incomplete, published for a different jurisdiction or market, or
interpreted incorrectly. A PDF or web page can be legally authoritative
while being less reliable to extract automatically.

- **High:** An identified official JSON or CSV calendar with explicit dated
  records. Check the schema, character encoding, jurisdiction and horizon,
  and interpret any observance or market-scope fields.

- **Medium:** An official human-readable HTML table, notice, statutory rule
  or exchange schedule. Dates, layout, footnotes and scope require human
  interpretation; a web page is not treated as a structured feed merely
  because it is HTML.

- **Low:** Dates depend on secondary HTML tables or reproduced documents,
  particularly where the official dated source could not be retrieved.
  Verify against the relevant authority before operational use.

For PDFs, a directly checked text-based official schedule would normally be
**Medium**, with additional column, footnote and text-extraction checks.
Scanned/image PDFs or OCR-only dates warrant **Low** until manually
corroborated. The Malaysian PDF below was unavailable: its dates were taken
from a secondary web listing, not extracted from that PDF.

The assessment uses the source representations recorded with this
snapshot. A page link without an identified machine-readable resource is
not upgraded to High on the assumption that an API exists behind it.
Mixed-source calendars have separate ratings where the collection record
supports that distinction. Ratings do not extend the published horizon,
validate annual-recurrence choices, or make estimated dates authoritative.
Each regional document records the source format and confidence beside
its calendars. The same notes remain in each XML calendar's source comment.

### Low-confidence calendars are disabled

Every calendar containing Low-confidence source data is supplied with
`disabled="true"` on its named `<activeTime>`, including mixed-source calendars
whose other dates have higher confidence. Merely referencing it does not
enable the distributed definition.

To use one, copy the active time into your own writable Gateway setup or site
include. Review the sources and ensure the calendar is suitable for your
jurisdiction, market and monitoring schedule. Edit dates manually where needed
before enabling the reviewed local copy. Give the copy a distinct name and
reference that name from your schedule, leaving the read-only distributed
calendar disabled. Regional documentation and each affected XML comment
identify this required review.

## Regional documentation

| Region | Documentation | XML include / status |
| --- | --- | --- |
| Africa (AF) | [Africa](holiday-calendars-af.md) | No verified calendars in this snapshot. |
| Asia excluding India (AS) | [Asia excluding India](holiday-calendars-as.md) | `holidays.layer0.as.xml` |
| Australia (AU) | [Australia](holiday-calendars-au.md) | `holidays.layer0.au.xml` |
| Europe including GB (EU) | [Europe including GB](holiday-calendars-eu.md) | `holidays.layer0.eu.xml` |
| India (IN) | [India](holiday-calendars-in.md) | `holidays.layer0.in.xml` |
| Middle East (ME) | [Middle East](holiday-calendars-me.md) | No verified calendars in this snapshot. |
| North America (NA) | [North America](holiday-calendars-na.md) | `holidays.layer0.na.xml` |
| South America (SA) | [South America](holiday-calendars-sa.md) | `holidays.layer0.sa.xml` |

These are geographic documentation/include labels, not ISO country codes
or political membership designations. EU includes GB; AS excludes India
and Australia; NA includes Mexico. India has city bank and NSE cash-market
calendars with retained source research. ME and AF are separate regions with
no encoded calendars in this snapshot.

Regional indexes and country sections are alphabetical by country or
territory, then public holidays, bank holidays, collective leave and exchange closures.
The XML country groups use the same order, so public and exchange calendars stay
together for each jurisdiction.

## Refresh process

The [monthly refresh runbook](RUNBOOK.md) describes source retrieval, date
conversion, confidence assessment and validation for maintainers and LLM
agents. It is included in the archive as `docs/RUNBOOK.md` and `docs/RUNBOOK.pdf`
so users can inspect the collection process as well as the resulting dates.

## Optional installation

Holidays are distributed separately as `geneos-holiday-includes-vYYYY.MM.DD.tar.gz`.
Their version is the date the source data was fetched, not the Cordial release
or archive build date. This snapshot is `v2026.11.20`, recorded in
the source tree's `pkg/holidays/VERSION`. Month and day are always two digits. Updating documentation
or rebuilding the archive does not change the snapshot version.
They are not part of the standard ITRS includes or the files installed by
`geneos init` or a no-argument `geneos includes install`.

```shell
geneos include install /path/to/geneos-holiday-includes-v2026.11.20.tar.gz
```

The archive uses the installer-compatible `geneos-includes-v<version>/`
wrapper, containing a `holidays-v<version>/` directory of `holidays.*.xml`
files, `scripts/install-includes.sh`, and a `docs/` directory with this guide,
regional Markdown and PDFs, and the refresh runbook in both formats.
The archive filename differs from its internal wrapper intentionally; the
installer validates the contents, not the filename.

Installation creates `gateway/includes/holidays-v<version>/` and missing
stable links such as `gateway/includes/holidays.layer0.in.xml`. It does
not add Gateway include references or enable calendars. Existing links are
preserved unless `--update` is supplied. For later upgrades, use
`geneos include install --update ARCHIVE` or
`geneos include update --prefix holidays`. The latter selects the
newest installed holiday snapshot; add `--version vYYYY.MM.DD`
to select a different installed version. A full
`--version holidays-v<version>` remains supported without `--prefix`.
Updates without either option still select the embedded ITRS corpus version,
which is versioned separately from Cordial and holidays.

Use `geneos include list --prefix holidays --version` to inspect installed
holiday versions, and `geneos include delete --prefix holidays` to remove
unused holiday versions. List and delete default to `--prefix itrs`; use
`--prefix '*'` for all sources. Installation infers the source from the
archive unless `--prefix` is explicitly supplied as a source check.

From the repository root, `make holidays` builds the optional archive into
`release-<cordial-version>/`, using `pkg/holidays/VERSION` and Docker to package the XML and
Markdown/PDF documentation. The full release pipeline also publishes this
archive and includes it in `SHA256SUMS`. The standard include archive remains
available through `make includes`.

### Installation without the `geneos` command

Extract the archive into a separate directory, change into its top-level
wrapper, and run the supplied shell installer with your Geneos home:

```shell
tar -xzf /path/to/geneos-holiday-includes-v2026.11.20.tar.gz
cd geneos-includes-v2026.11.20
sh scripts/install-includes.sh /path/to/geneos
```

The shared installer detects the `holidays` source, installs its XML read-only
into a new version directory, and creates only missing top-level holiday
links. Existing files, links (including dangling ones), standard ITRS includes
and site configuration are left alone. There are no configuration templates
in the holiday archive, so it does not create or migrate site files.
The installer refuses an already-installed version rather than overwriting it.
It does not add Gateway references, enable calendars or update existing links.
Use `geneos include install --update ARCHIVE` for managed upgrades and
Gateway-reference checks. The shell installer assumes a trusted, already
extracted archive; it does not replace the CLI's archive validation.

## Use in Geneos

Add only the XML includes listed above that your Gateway needs.

Under Active Times, look in **Layer 0 - Holidays** for the country
groups and reference a **named calendar**, not its group, alongside
your normal schedule. For example:

```xml
<activeTime name="London monitoring hours">
    <scheduledPeriod>
        <startTime>08:00</startTime>
        <endTime>18:00</endTime>
        <days>
            <monday>true</monday>
            <tuesday>true</tuesday>
            <wednesday>true</wednesday>
            <thursday>true</thursday>
            <friday>true</friday>
        </days>
    </scheduledPeriod>
    <activeTime ref="Holidays - GB England and Wales - Public"/>
    <timezone><Europe>London</Europe></timezone>
</activeTime>
```

The include provides override periods, not a standalone business-hours
schedule. Referencing both public and exchange calendars suppresses both
sets of dates; choose only the exchange calendar if public holidays on
which trading continues should remain active. Exchange calendars use the
venue's timezone, which need not match your office's.
Legacy Netprobe-side sampler scheduling does not support calendar timezones.
To use calendar timezones with samplers, also include
`itrs.layer0.foundation.xml`, which enables **Gateway-controlled sampler
active times**. If you do not use the foundation include, enable the same
option in the Gateway setup under **Active Times**:

```xml
<activeTimes>
    <gatewayControlledProbeActiveTimes>true</gatewayControlledProbeActiveTimes>
</activeTimes>
```

If your setup already has an `<activeTimes>` section, add the option there
before the active time definitions rather than creating a second section.
