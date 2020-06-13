package ini

import (
	"bufio"
	"encoding/json"
	"regexp"
)

// KeyValueDict is a map from string to string
type KeyValueDict map[string]string

func (keyValueDict KeyValueDict) String() string {
	b, err := json.MarshalIndent(keyValueDict, "", "  ")
	if err != nil {
		panic(err)
	}

	return string(b)
}

var sectionRe = regexp.MustCompile(`^\s*\[\s*([^\s]*)\s*\]\s*$`)
var keyPairRe = regexp.MustCompile(`^\s*(.+?)\s*=\s*(.+?)\s*$`)

// Parse is a function for extracting sections and key-value pairs
// from a '.ini' file.
func Parse(scanner *bufio.Scanner) map[string]KeyValueDict {
	var sectionName string
	config := map[string]KeyValueDict{}

	for scanner.Scan() {
		line := scanner.Text()

		submatches := sectionRe.FindStringSubmatch(line)
		if len(submatches) >= 2 {
			sectionName = submatches[1]
			if _, exists := config[sectionName]; !exists {
				config[sectionName] = KeyValueDict{}
			}
			continue
		}

		submatches = keyPairRe.FindStringSubmatch(line)
		if len(submatches) >= 3 {
			config[sectionName][submatches[1]] = submatches[2]
		}
	}

	return config
}
