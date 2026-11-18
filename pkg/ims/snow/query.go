package snow

import "strings"

// EscapeQuery returns value escaped for use as the value of a condition
// in a ServiceNow encoded query (the `sysparm_query` parameter).
//
// In an encoded query a `^` separates conditions, and is also the start
// of the `^OR`, `^NQ` and `^EQ` operators, so a value from a request
// that contains one would end the condition and could add others. A
// literal `^` in a value is written as `^^`. Nothing else in a value is
// special, and in particular `\` is not an escape. The result still has
// to be URL encoded, as for any query parameter, which the request
// building already does.
func EscapeQuery(value string) string {
	return strings.ReplaceAll(value, "^", "^^")
}

// EscapeQueryValues returns a copy of values with each of the values
// escaped using [EscapeQuery], for use as the lookup table when
// expanding a configured query template.
func EscapeQueryValues(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	escaped := make(map[string]string, len(values))
	for k, v := range values {
		escaped[k] = EscapeQuery(v)
	}
	return escaped
}
