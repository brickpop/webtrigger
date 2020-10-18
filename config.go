// +build linux darwin freebsd netbsd openbsd

package main

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v2"
)

// DefaultPort is used whtn no valid port is found on the config file
const DefaultPort = 5000

// Trigger holds the settings of a trigger
type Trigger struct {
	ID      string `yaml:"id"`
	Token   string `yaml:"token"`
	Script  string `yaml:"script"`
	Timeout int    `yaml:"timeout"`
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
	var ids []string

	for idx, trigger := range conf.Triggers {
		// Check name
		if trigger.ID == "" {
			return errors.Errorf("[CONFIG] The trigger at index %d has no ID defined", idx)
		}
		for _, prevID := range ids {
			if prevID == trigger.ID {
				return errors.Errorf("[CONFIG] The trigger %s is defined multiple times", prevID)
			}
		}
		ids = append(ids, trigger.ID)

		// Check token
		if trigger.Token == "" {
			return errors.Errorf("[CONFIG] The trigger %s has no token", trigger.ID)
		}
		if len(trigger.Token) < 6 {
			return errors.Errorf("[CONFIG] The token for trigger %s should have at least 6 characters", trigger.ID)
		}

		// Check script
		info, err := os.Stat(trigger.Script)
		if os.IsNotExist(err) {
			return errors.Errorf("[CONFIG] The script file does not exist: %s", trigger.Script)
		}
		if info.IsDir() {
			return errors.Errorf("[CONFIG] The script path is a directory: %s", trigger.Script)
		}
		err = unix.Access(trigger.Script, unix.X_OK)
		if err != nil {
			return errors.Errorf("[CONFIG] The script is not executable: %s", trigger.Script)
		}
	}

	return nil
}
