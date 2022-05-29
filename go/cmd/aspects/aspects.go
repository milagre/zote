package aspects

func Prefix(prefix string, attr string) string {
	if prefix == "" {
		return attr
	}

	return prefix + "-" + attr
}
