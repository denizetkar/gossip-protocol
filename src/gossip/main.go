package main

import (
	"bufio"
	"fmt"
	"os"
	"parser/ini"
	"path/filepath"
)

var gossipWorkspacePath string

// init is an initialization function for 'main' package, called by Go.
func init() {
	gossipWorkspacePath, _ = os.Getwd()
}

func main() {

	configFile, err := os.Open(filepath.Join(gossipWorkspacePath, "config", "config.ini"))
	if err != nil {
		panic(err)
	}
	defer configFile.Close()
	scanner := bufio.NewScanner(configFile)

	config := ini.Parse(scanner)
	fmt.Println(config)
}
