package server

import (
	"strings"
)

type HttpPathMapping struct {
	HttpPath string
	FilePath string
}

func GetHttpPath(path string) string {
	// Create base path (http://foobar:5000/<base>)
	pathSplit := strings.Split(path, "/")
	var base string
	if len(pathSplit) > 0 {
		base = pathSplit[len(pathSplit)-1]
	} else {
		base = path
	}

	return "/" + base
}

func GetDefaultMappings(filePaths []string) []HttpPathMapping {
	mappings := []HttpPathMapping{}

	for _, f := range filePaths {
		m := HttpPathMapping{
			HttpPath: GetHttpPath(f),
			FilePath: f,
		}

		mappings = append(mappings, m)
	}

	return mappings
}
