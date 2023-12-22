package zaspect

import (
	"fmt"
)

func Prefix(prefix string, attr string) string {
	if prefix == "" {
		return attr
	}

	return prefix + "-" + attr
}

func Format(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
