package generator

import "strings"

func getServiceName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		fileName := parts[len(parts)-1]
		if fileParts := strings.SplitAfter(fileName, "."); len(fileParts) > 0 &&
			fileParts[len(fileParts)-1] == "service" {
			return fileName
		}
	}

	return ""
}
