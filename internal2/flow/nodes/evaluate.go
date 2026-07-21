package nodes

import "strings"

// Evaluate runs a simple operator check on a field value. Used by loop nodes.
func Evaluate(op, fieldVal, compareVal string) bool {
	switch op {
	case "contains":
		return strings.Contains(fieldVal, compareVal)
	case "equals":
		return strings.TrimSpace(fieldVal) == strings.TrimSpace(compareVal)
	case "not_empty":
		return strings.TrimSpace(fieldVal) != ""
	case "is_json":
		return strings.HasPrefix(strings.TrimSpace(fieldVal), "{") || strings.HasPrefix(strings.TrimSpace(fieldVal), "[")
	default:
		return strings.Contains(fieldVal, compareVal)
	}
}
