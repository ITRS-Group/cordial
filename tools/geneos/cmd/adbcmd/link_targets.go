package adbcmd

import "strings"

type linkTarget struct {
	start int
	end   int
	kind  string
	value string
}

func findLinkTargets(value string) []linkTarget {
	targets := make([]linkTarget, 0)
	for index := 0; index < len(value); {
		if !isIdentifierStart(value[index]) {
			index++
			continue
		}

		start := index
		for index < len(value) && isIdentifierPart(value[index]) {
			index++
		}
		if index == len(value) || value[index] != '(' {
			continue
		}

		end, ok := matchingParen(value, index)
		if !ok {
			break
		}
		targets = append(targets, linkTarget{
			start: start,
			end:   end,
			kind:  value[start:index],
			value: value[index+1 : end-1],
		})
		index = end
	}
	return targets
}

func rewriteLinkTargets(value string, rewrite func(kind, target string) string) string {
	targets := findLinkTargets(value)
	if len(targets) == 0 {
		return value
	}

	var result strings.Builder
	position := 0
	for _, target := range targets {
		result.WriteString(value[position:target.start])
		result.WriteString(target.kind)
		result.WriteByte('(')
		result.WriteString(rewrite(target.kind, target.value))
		result.WriteByte(')')
		position = target.end
	}
	result.WriteString(value[position:])
	return result.String()
}

func matchingParen(value string, open int) (int, bool) {
	depth := 0
	var quote byte
	for index := open; index < len(value); index++ {
		char := value[index]
		if char == '\\' && index+1 < len(value) {
			index++
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		switch char {
		case '\'', '"':
			quote = char
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return index + 1, true
			}
		}
	}
	return 0, false
}

func isIdentifierStart(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char == '_'
}

func isIdentifierPart(char byte) bool {
	return isIdentifierStart(char) || char >= '0' && char <= '9'
}
