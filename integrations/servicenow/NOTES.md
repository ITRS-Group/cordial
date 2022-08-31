
Priority Rules (inc Impact / Urgency)
https://docs.servicenow.com/bundle/rome-it-service-management/page/product/incident-management/task/def-prio-lookup-rules.html


https://docs.servicenow.com/bundle/rome-application-development/page/integrate/inbound-rest/concept/c_TableAPI.html?cshalt=yes

sysparm_query

Encoded query used to filter the result set.
Syntax: sysparm_query=<col_name><operator><value>.
<col_name>: Name of the table column to filter against.
<operator>: Supports the following values:
=: Exactly matches <value>.
!=: Does not match <value>.
^: Logically AND multiple query statements.
^OR: Logically OR multiple query statements.
LIKE: <col_name> contains the specified string. Only works for <col_name> fields whose data type is string.
STARTSWITH: <col_name> starts with the specified string. Only works for <col_name> fields whose data type is string.
ENDSWITH: <col_name> ends with the specified string. Only works for <col_name> fields whose data type is string.
<value>: Value to match against.
All parameters are case-sensitive. Queries can contain more than one entry, such as sysparm_query=<col_name><operator><value>[<operator><col_name><operator><value>].

For example:

(sysparm_query=caller_id=javascript:gs.getUserID()^active=true)

Encoded queries also supports order by functionality. To sort responses based on certain fields, use the ORDERBY and ORDERBYDESC clauses in sysparm_query.

Syntax:
ORDERBY<col_name>
ORDERBYDESC<col_name>
For example: sysparm_query=active=true^ORDERBYnumber^ORDERBYDESCcategory

This query filters all active records and orders the results in ascending order by number, and then in descending order by category.

If part of the query is invalid, such as by specifying an invalid field name, the instance ignores the invalid part. It then returns rows using only the valid portion of the query. You can control this behavior using the property glide.invalid_query.returns_no_rows. Set this property to true to return no rows on an invalid query.




flow:

create

    ./client -short "new issue" -text "raising an issue against IBM-T42-DLG" -severity critical -search name="IBM-T42-DLG"

update

    ./client -text "some update 1" -search sys_id=b0cbf176c0a80009002b452bc33e2fc3 -severity warning

resolve

    state=6

    ./client -text "fixed by turning it off and on again" -search name="IBM-T42-DLG" state=6 close_code="Closed/Resolved by Caller" close_note="rebooted"

snooze -> in progress on-hold ?

    state=2 in progress

    state=3
    hold_reason=3 "Awaiting Problem" or 2="Awaiting Change"


    ./...