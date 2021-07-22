package srm

import "strings"

var upperCaseLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ" // No support for unicode upper case letters.
func isPublic(name string) bool {
	return strings.ContainsAny(name[0:1], upperCaseLetters)
}

func camelToSnake(text string) string {
	var (
		results = []string{""}
	)

	for len(text) > 0 {
		i := strings.IndexAny(text, upperCaseLetters)
		if i == 0 {
			results[len(results)-1] += strings.ToLower(text[0:1])
			text = text[1:]
			continue
		}

		if i < 0 {
			results[len(results)-1] += text
			break
		}

		results[len(results)-1] += text[0:i]
		results = append(results, "")
		text = text[i:]
	}
	return strings.Join(results, "_")
}
