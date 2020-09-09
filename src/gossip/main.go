package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"parser/ini"
	"path/filepath"
)

var gossipWorkspacePath string

// init is an initialization function for 'main' package, called by Go.
func init() {
	gossipWorkspacePath, _ = os.Getwd()
}

func newCentralControllerFromConfigFile(configPath string) (*CentralController, error) {
	config, err := ini.ReadConfigFile(configPath)
	if err != nil {
		return nil, err
	}

	// Check if GLOBAL configurations exist.
	globalConfig, ok := config["GLOBAL"]
	if !ok {
		return nil, fmt.Errorf("GLOBAL section cannot be found in the config file: %s", configPath)
	}
	hostKeyPath, err := globalConfig.GetStringValue("hostkey")
	if err != nil {
		return nil, err
	}
	pubKeyPath, err := globalConfig.GetStringValue("pubkey")
	if err != nil {
		return nil, err
	}
	// Check if the configurations for gossip module exist
	gossipConfig, ok := config["gossip"]
	if !ok {
		return nil, fmt.Errorf("Gossip section cannot be found in the config file: %s", configPath)
	}
	// Check if the trusted identities path exists
	trustedIdentitiesPath, err := gossipConfig.GetStringValue("trusted_identities_path")
	if err != nil {
		return nil, err
	}
	// Check if the bootstrapper address exists
	bootstrapper, err := gossipConfig.GetStringValue("bootstrapper")
	if err != nil {
		return nil, err
	}
	// Check if the API address exists
	apiAddr, err := gossipConfig.GetStringValue("api_address")
	if err != nil {
		return nil, err
	}
	// Check if the P2P listening address exists
	p2pAddr, err := gossipConfig.GetStringValue("listen_address")
	if err != nil {
		return nil, err
	}
	// Check if the "cache size" exists
	cacheSize, err := gossipConfig.GetUint16Value("cache_size")
	if err != nil {
		return nil, err
	}
	// Check if the "degree" exists
	degree, err := gossipConfig.GetUint8Value("degree")
	if err != nil {
		return nil, err
	}
	// Check if the "maxTTL" exists
	maxTTL, err := gossipConfig.GetUint8Value("max_ttl")
	if err != nil {
		return nil, err
	}

	centralController, err := NewCentralController(
		trustedIdentitiesPath, hostKeyPath, pubKeyPath, bootstrapper, apiAddr, p2pAddr, cacheSize, degree, maxTTL,
	)
	if err != nil {
		return nil, err
	}

	return centralController, nil
}

func main() {
	// Set global logging settings.
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)

	// Take config file path as a command line argument.
	defaultConfigPath := filepath.Join(gossipWorkspacePath, "config", "config.ini")
	configPath := flag.String("config_path", defaultConfigPath, "a file path string for the configuration")
	// Create a central controller and run it.
	centralController, err := newCentralControllerFromConfigFile(*configPath)
	if err != nil {
		// Log the error and exit.
		log.Fatalln(err)
	}
	log.Println(centralController)
	//centralController.Run()
}
