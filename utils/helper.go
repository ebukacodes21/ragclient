package utils

import "strings"

func SanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "(", "_")
	name = strings.ReplaceAll(name, ")", "_")
	return name
}
