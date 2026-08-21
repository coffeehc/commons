package pgdialect

import (
	"strconv"
	"strings"
)

func quoteIdentifier(identifier string) string {
	parts := strings.Split(identifier, ".")
	for index, part := range parts {
		if part == "*" {
			continue
		}
		parts[index] = `"` + strings.ReplaceAll(part, `"`, `""`) + `"`
	}
	return strings.Join(parts, ".")
}

func rewriteSQL(query string) string {
	var result strings.Builder
	result.Grow(len(query) + 16)
	placeholder := 0
	for index := 0; index < len(query); {
		switch query[index] {
		case '\'':
			index = copyQuoted(&result, query, index, '\'', '\'')
		case '"':
			index = copyQuoted(&result, query, index, '"', '"')
		case '`':
			index = copyBacktickIdentifier(&result, query, index)
		case '-':
			if index+1 < len(query) && query[index+1] == '-' {
				index = copyLineComment(&result, query, index)
				continue
			}
			result.WriteByte(query[index])
			index++
		case '/':
			if index+1 < len(query) && query[index+1] == '*' {
				index = copyBlockComment(&result, query, index)
				continue
			}
			result.WriteByte(query[index])
			index++
		case '$':
			if delimiter := dollarQuoteDelimiter(query[index:]); delimiter != "" {
				index = copyDollarQuote(&result, query, index, delimiter)
				continue
			}
			result.WriteByte(query[index])
			index++
		case '?':
			if index+1 < len(query) && query[index+1] == '?' {
				result.WriteByte('?')
				index += 2
				continue
			}
			if index+1 < len(query) && (query[index+1] == '|' || query[index+1] == '&') {
				result.WriteByte('?')
				index++
				continue
			}
			placeholder++
			result.WriteByte('$')
			result.WriteString(strconv.Itoa(placeholder))
			index++
		default:
			result.WriteByte(query[index])
			index++
		}
	}
	return result.String()
}

func copyQuoted(result *strings.Builder, query string, start int, quote byte, outputQuote byte) int {
	result.WriteByte(outputQuote)
	for index := start + 1; index < len(query); index++ {
		result.WriteByte(query[index])
		if quote == '\'' && query[index] == '\\' && index+1 < len(query) {
			result.WriteByte(query[index+1])
			index++
			continue
		}
		if query[index] != quote {
			continue
		}
		if index+1 < len(query) && query[index+1] == quote {
			result.WriteByte(query[index+1])
			index++
			continue
		}
		return index + 1
	}
	return len(query)
}

func copyBacktickIdentifier(result *strings.Builder, query string, start int) int {
	result.WriteByte('"')
	for index := start + 1; index < len(query); index++ {
		switch query[index] {
		case '`':
			if index+1 < len(query) && query[index+1] == '`' {
				result.WriteByte('`')
				index++
				continue
			}
			result.WriteByte('"')
			return index + 1
		case '"':
			result.WriteString(`""`)
		default:
			result.WriteByte(query[index])
		}
	}
	return len(query)
}

func copyLineComment(result *strings.Builder, query string, start int) int {
	for index := start; index < len(query); index++ {
		result.WriteByte(query[index])
		if query[index] == '\n' {
			return index + 1
		}
	}
	return len(query)
}

func copyBlockComment(result *strings.Builder, query string, start int) int {
	depth := 0
	for index := start; index < len(query); index++ {
		if index+1 < len(query) && query[index] == '/' && query[index+1] == '*' {
			depth++
			result.WriteString("/*")
			index++
			continue
		}
		if index+1 < len(query) && query[index] == '*' && query[index+1] == '/' {
			depth--
			result.WriteString("*/")
			index++
			if depth == 0 {
				return index + 1
			}
			continue
		}
		result.WriteByte(query[index])
	}
	return len(query)
}

func dollarQuoteDelimiter(query string) string {
	if len(query) < 2 || query[0] != '$' {
		return ""
	}
	for index := 1; index < len(query); index++ {
		character := query[index]
		if character == '$' {
			return query[:index+1]
		}
		if character != '_' && (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') {
			return ""
		}
	}
	return ""
}

func copyDollarQuote(result *strings.Builder, query string, start int, delimiter string) int {
	end := strings.Index(query[start+len(delimiter):], delimiter)
	if end < 0 {
		result.WriteString(query[start:])
		return len(query)
	}
	end += start + len(delimiter) + len(delimiter)
	result.WriteString(query[start:end])
	return end
}
