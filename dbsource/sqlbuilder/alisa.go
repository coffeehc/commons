package sqlbuilder

import (
	"strings"
)

type aliasDefinition struct {
	tableAlisa string
	alisa      string
	joinAlisa  string
	openJoin   bool
}

func (impl aliasDefinition) handle(colName string) string {
	colName = strings.TrimSpace(colName)
	if impl.openJoin {
		if strings.Index(colName, ".") > 0 {
			if impl.tableAlisa != "" {
				tablePrefix := strings.TrimSuffix(impl.tableAlisa, ".") + "."
				if strings.HasPrefix(colName, tablePrefix) {
					colName = impl.joinAlisa + strings.TrimPrefix(colName, tablePrefix)
				}
			}
		} else {
			colName = impl.alisa + colName
		}
	}
	return quoteIdentifier(colName)
}

func (impl aliasDefinition) handleProjection(projection string) string {
	projection = strings.TrimSpace(projection)
	if !isIdentifierReference(projection) {
		return projection
	}
	return impl.handle(projection)
}

func quoteIdentifier(identifier string) string {
	parts := strings.Split(identifier, ".")
	for index, part := range parts {
		part = strings.TrimSpace(part)
		if part == "*" {
			continue
		}
		if isQuotedIdentifierPart(part) {
			parts[index] = part
			continue
		}
		parts[index] = "`" + strings.ReplaceAll(part, "`", "``") + "`"
	}
	return strings.Join(parts, ".")
}

func isIdentifierReference(identifier string) bool {
	parts := strings.Split(identifier, ".")
	for _, part := range parts {
		if part != "*" && !isIdentifierPart(part) && !isQuotedIdentifierPart(part) {
			return false
		}
	}
	return true
}

func isQuotedIdentifierPart(part string) bool {
	if len(part) < 2 {
		return false
	}
	return part[0] == '`' && part[len(part)-1] == '`' || part[0] == '"' && part[len(part)-1] == '"'
}

func isIdentifierPart(part string) bool {
	if part == "" {
		return false
	}
	for index := 0; index < len(part); index++ {
		character := part[index]
		if character == '_' || character == '$' || character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || index > 0 && character >= '0' && character <= '9' {
			continue
		}
		return false
	}
	return true
}
