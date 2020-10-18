// +build linux darwin freebsd netbsd openbsd

package main

import (
	"io/ioutil"
	"os"
	"sync"

	"github.com/kballard/go-shellquote"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v2"
)

// DefaultPort is used whtn no valid port is found on the config file
const DefaultPort = 5000

// Trigger holds the settings of a trigger
type Trigger struct {
	ID        string `yaml:"id"`
	Token     string `yaml:"token"`
	Command   string `yaml:"command"`
	Timeout   int    `yaml:"timeout"`
	Status    TriggerStatus
	WaitGroup *sync.WaitGroup
}

// Config holds the trigger definitions
type Config struct {
	Port     int       `yaml:"port"`
	Triggers []Trigger `yaml:"triggers"`
}

// TriggerStatus contains the supported trigger statuses
type TriggerStatus int

const (
	// StatusUnstarted is the default trigger state
	StatusUnstarted TriggerStatus = iota
	// StatusRunning indicated a trigger in progress
	StatusRunning
	// StatusDone indicates a successfully completed trigger
	StatusDone
	// StatusFailed indicates that the last execution failed
	StatusFailed
)

func (d TriggerStatus) String() string {
	return [...]string{"unstarted", "running", "done", "failed"}[d]
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

	err = checkConfig(conf)
	if err != nil {
		return conf, err
	}

	return conf, nil
}

func checkConfig(conf Config) error {
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

		// Check the command
		commandItems, err := shellquote.Split(trigger.Command)
		if err != nil {
			return errors.Errorf("[CONFIG] %s\n%s", err, trigger.Command)
		}

		executableFile := commandItems[0]
		info, err := os.Stat(executableFile)
		if os.IsNotExist(err) {
			return errors.Errorf("[CONFIG] The script file does not exist: %s", executableFile)
		} else if info.IsDir() {
			return errors.Errorf("[CONFIG] The script path is a directory: %s", executableFile)
		}
		err = unix.Access(executableFile, unix.X_OK)
		if err != nil {
			return errors.Errorf("[CONFIG] The script is not executable: %s", executableFile)
		}
	}

	return nil
}
