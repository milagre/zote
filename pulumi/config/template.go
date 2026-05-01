package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/milagre/zote/pulumi/env"
)

var envRefPattern = regexp.MustCompile(`\$\{env\.([a-zA-Z_][a-zA-Z0-9_]*)\}`)
var escapedEnvRef = regexp.MustCompile(`\$\$\{`)

func renderTemplate(src string, e env.Env) (string, error) {
	const sentinel = "\x00ZOTE_ESCAPED_DOLLAR\x00"
	work := escapedEnvRef.ReplaceAllString(src, sentinel)

	var firstErr error
	out := envRefPattern.ReplaceAllStringFunc(work, func(match string) string {
		field := envRefPattern.FindStringSubmatch(match)[1]
		value, err := lookupEnvField(e, field)
		if err != nil && firstErr == nil {
			firstErr = err
		}

		return value
	})
	if firstErr != nil {
		return "", firstErr
	}

	return strings.ReplaceAll(out, sentinel, "${"), nil
}

func lookupEnvField(e env.Env, field string) (string, error) {
	switch field {
	case "type":
		return e.Type, nil
	case "tier":
		return e.Tier, nil
	case "name":
		return e.Name, nil
	case "id":
		return e.ID(), nil
	case "root":
		return e.Root, nil
	case "prefix":
		return e.Prefix, nil
	case "is_dev":
		return boolString(e.IsDev()), nil
	case "is_local":
		return boolString(e.IsLocal()), nil
	case "lb_type":
		return e.LBType(), nil
	default:
		return "", fmt.Errorf("unknown env field %q in template", field)
	}
}

func boolString(b bool) string {
	if b {
		return "true"
	}

	return "false"
}
