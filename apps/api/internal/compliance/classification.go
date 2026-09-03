package compliance

import "strings"

type DataClass string

const (
	ClassPublic    DataClass = "PUBLIC"
	ClassInternal  DataClass = "INTERNAL"
	ClassPII       DataClass = "PII"
	ClassSensitive DataClass = "SENSITIVE"
)

func ClassifyField(name, value string) DataClass {
	lower := strings.ToLower(name)
	if containsPII(value) {
		return ClassPII
	}
	switch {
	case strings.Contains(lower, "password"), strings.Contains(lower, "secret"), strings.Contains(lower, "token"):
		return ClassSensitive
	case strings.Contains(lower, "email"), strings.Contains(lower, "phone"), strings.Contains(lower, "name"):
		return ClassPII
	case strings.Contains(lower, "internal"), strings.Contains(lower, "org"):
		return ClassInternal
	default:
		return ClassPublic
	}
}

func ClassifyFields(fields map[string]string) map[string]DataClass {
	out := make(map[string]DataClass, len(fields))
	for k, v := range fields {
		out[k] = ClassifyField(k, v)
	}
	return out
}
