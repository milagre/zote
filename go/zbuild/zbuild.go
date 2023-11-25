package zbuild

import _ "embed"

//go:generate sh -c "printf %s $(git rev-parse --short HEAD) > hash.txt"
//go:embed hash.txt
var hash string

//go:generate sh -c "printf %s $(date -u +'%Y-%m-%dT%H:%M:%SZ') > timestamp.txt"
//go:embed timestamp.txt
var timestamp string

//go:generate sh -c "printf %s $(git describe --tags --abbrev=0 2> /dev/null || git rev-parse --abbrev-ref HEAD) > version.txt"
//go:embed version.txt
var version string

func Hash() string {
	return hash
}

func Timestamp() string {
	return timestamp
}

func Version() string {
	return version
}
