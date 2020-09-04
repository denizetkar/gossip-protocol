package identity

import (
	"io/ioutil"
	"log"
	"regexp"
)

func Parse(path string) (identities []string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	regex := regexp.MustCompile("^[1-9A-F]{64}$")
	for _, value := range files {
		if regex.MatchString(value.Name()) {
			identities = append(identities, value.Name())
		}
	}

	return identities
}
