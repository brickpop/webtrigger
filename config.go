package main

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// DefaultPort is used whtn no valid port is found on the config file
const DefaultPort = 5000

// Trigger holds the settings of a trigger
type Trigger struct {
	ID     string `yaml:"id"`
	Token  string `yaml:"token"`
	Script string `yaml:"script"`
}

// Config holds the trigger definitions
type Config struct {
	Port     int       `yaml:"port"`
	Triggers []Trigger `yaml:"triggers"`
}

// ReadConfig parses `config.toml` and returns a struct with the desired config
func ReadConfig(filePath string) (Config, error) {
	conf := Config{}

	// Read config.toml
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return conf, err
	}

	err = yaml.Unmarshal([]byte(data), &conf)
	if err != nil {
		return conf, err
	}

	if conf.Port == 0 {
		conf.Port = DefaultPort
	}
	if len(conf.Triggers) < 1 {
		return conf, errors.Errorf("No triggers are defined")
	}

	err = checkScripts(conf)
	if err != nil {
		return conf, err
	}

	return conf, nil
}

func checkScripts(conf Config) error {
	for _, trigger := range conf.Triggers {
		info, err := os.Stat(trigger.Script)
		if os.IsNotExist(err) {
			return errors.Errorf("[CONFIG] The script file does not exist: %s", trigger.Script)
		}
		if info.IsDir() {
			return errors.Errorf("[CONFIG] The script path is a directory: %s", trigger.Script)
		}
	}

	return nil
}
