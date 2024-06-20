package zapi

import (
	"fmt"
	"net/url"
)

func AppendRoutes(routelists ...[]Route) []Route {
	result := []Route{}
	for _, routes := range routelists {
		result = append(result, routes...)
	}
	return result
}

func Pathf(template string, params ...string) string {
	parts := make([]interface{}, len(params))
	for i, p := range params {
		parts[i] = fmt.Sprintf("{%s}", p)
	}

	return fmt.Sprintf(template, parts...)
}

func RenderPathf(source Source, template string, params ...interface{}) string {
	parts := make([]any, len(params))
	for i, p := range params {
		s := fmt.Sprintf("%s", p)
		parts[i] = url.PathEscape(s)
	}

	path := fmt.Sprintf(template, parts...)
	return (&url.URL{
		Path: source.Root,
	}).JoinPath(path).Path
}
