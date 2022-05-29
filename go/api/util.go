package api

func AppendRoutes(routelists ...[]Route) []Route {
	result := []Route{}
	for _, routes := range routelists {
		result = append(result, routes...)
	}
	return result
}
