// Package identity contains an implementation for parsing
// trusted identities out of a given folder path.
package identity

import (
	"io/ioutil"
	"log"
	"regexp"
)

// Parse method extracts the identity of every trusted peer
// out of a given folder path.
func Parse(path string) (identities []string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	regex := regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	for _, value := range files {
		if regex.MatchString(value.Name()) {
			identities = append(identities, value.Name())
		}
	}

	return identities
}
