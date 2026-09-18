package stringsx

import "strings"

func JoinURLPath(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// FieldsIgnoringQuote 以指定字符分割字符串，忽略单引号或双引号内的分割符
// 如："'bg_fee_0,old_bg_fee_0', 0, '5405', 'BETRCQP', '423609511', '15800', '1', 'WANG SHENGWEI ', '1', 2"
func FieldsIgnoringQuote(s string, sep rune) []string {
	isInSingleQuote := false
	fields := strings.FieldsFunc(s, func(r rune) bool {
		if r == '\'' || r == '"' {
			isInSingleQuote = !isInSingleQuote
			return false
		}

		if !isInSingleQuote && r == sep {
			return true
		}

		return false
	})

	var newFields []string
	for _, field := range fields {
		newFields = append(newFields, strings.Trim(field, `'" `))
	}

	return newFields
}
