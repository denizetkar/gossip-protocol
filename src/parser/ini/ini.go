// Package ini contains an implementation of '.ini' config file parser.
package ini

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
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

// GetStringValue is a funtion to retrieve a value of type 'string'
// with key 'key' from the section 'keyValueDict'.
func (keyValueDict KeyValueDict) GetStringValue(key string) (string, error) {
	value, ok := keyValueDict[key]
	if !ok {
		return "", fmt.Errorf("the key %q cannot be found in the section", key)
	}
	return value, nil
}

// GetUint8Value is a funtion to retrieve a value of type 'uint8'
// with key 'key' from the section 'keyValueDict'.
func (keyValueDict KeyValueDict) GetUint8Value(key string) (uint8, error) {
	valueStr, ok := keyValueDict[key]
	if !ok {
		return 0, fmt.Errorf("the key %q cannot be found in the section", key)
	}
	value, err := strconv.ParseUint(valueStr, 10, 8)
	if err != nil {
		return 0, err
	}
	return uint8(value), nil
}

// GetUint16Value is a funtion to retrieve a value of type 'uint16'
// with key 'key' from the section 'keyValueDict'.
func (keyValueDict KeyValueDict) GetUint16Value(key string) (uint16, error) {
	valueStr, ok := keyValueDict[key]
	if !ok {
		return 0, fmt.Errorf("the key %q cannot be found in the section", key)
	}
	value, err := strconv.ParseUint(valueStr, 10, 16)
	if err != nil {
		return 0, err
	}
	return uint16(value), nil
}

// GetUint32Value is a funtion to retrieve a value of type 'uint32'
// with key 'key' from the section 'keyValueDict'.
func (keyValueDict KeyValueDict) GetUint32Value(key string) (uint32, error) {
	valueStr, ok := keyValueDict[key]
	if !ok {
		return 0, fmt.Errorf("the key %q cannot be found in the section", key)
	}
	value, err := strconv.ParseUint(valueStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(value), nil
}

// GetUint64Value is a funtion to retrieve a value of type 'uint64'
// with key 'key' from the section 'keyValueDict'.
func (keyValueDict KeyValueDict) GetUint64Value(key string) (uint64, error) {
	valueStr, ok := keyValueDict[key]
	if !ok {
		return 0, fmt.Errorf("the key %q cannot be found in the section", key)
	}
	value, err := strconv.ParseUint(valueStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}

// GetInt8Value is a funtion to retrieve a value of type 'int8'
// with key 'key' from the section 'keyValueDict'.
func (keyValueDict KeyValueDict) GetInt8Value(key string) (int8, error) {
	valueStr, ok := keyValueDict[key]
	if !ok {
		return 0, fmt.Errorf("the key %q cannot be found in the section", key)
	}
	value, err := strconv.ParseInt(valueStr, 10, 8)
	if err != nil {
		return 0, err
	}
	return int8(value), nil
}

// GetInt16Value is a funtion to retrieve a value of type 'int16'
// with key 'key' from the section 'keyValueDict'.
func (keyValueDict KeyValueDict) GetInt16Value(key string) (int16, error) {
	valueStr, ok := keyValueDict[key]
	if !ok {
		return 0, fmt.Errorf("the key %q cannot be found in the section", key)
	}
	value, err := strconv.ParseInt(valueStr, 10, 16)
	if err != nil {
		return 0, err
	}
	return int16(value), nil
}

// GetInt32Value is a funtion to retrieve a value of type 'int32'
// with key 'key' from the section 'keyValueDict'.
func (keyValueDict KeyValueDict) GetInt32Value(key string) (int32, error) {
	valueStr, ok := keyValueDict[key]
	if !ok {
		return 0, fmt.Errorf("the key %q cannot be found in the section", key)
	}
	value, err := strconv.ParseInt(valueStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(value), nil
}

// GetInt64Value is a funtion to retrieve a value of type 'int64'
// with key 'key' from the section 'keyValueDict'.
func (keyValueDict KeyValueDict) GetInt64Value(key string) (int64, error) {
	valueStr, ok := keyValueDict[key]
	if !ok {
		return 0, fmt.Errorf("the key %q cannot be found in the section", key)
	}
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
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

// ReadConfigFile is a convenience function to directly read a
// *.ini config file and return the configuration.
func ReadConfigFile(configPath string) (map[string]KeyValueDict, error) {
	configFile, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer configFile.Close()
	scanner := bufio.NewScanner(configFile)
	config := Parse(scanner)

	return config, nil
}
