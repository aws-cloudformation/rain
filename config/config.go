package config

import (
	"fmt"
	"os"
	"path"

	"github.com/aws-cloudformation/rain/console/text"
	"github.com/spf13/viper"
)

const (
	configName = "config"
	configDir  = ".rain"
)

var Debug = false
var Profile = ""
var Region = ""

func Debugf(message string, parts ...interface{}) {
	if Debug {
		fmt.Println(text.Orange("DEBUG: " + fmt.Sprintf(message, parts...)))
	}
}

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		// No place like home
		panic(err)
	}

	viper.SetConfigName(configName)
	viper.AddConfigPath(path.Join(home, configDir))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(fmt.Errorf("Failed to read config file: %s", err.Error()))
		} else {
			Debugf("Creating new config file")

			err = os.Mkdir(path.Join(home, configDir), 0755)
			if err != nil && !os.IsExist(err) {
				panic(fmt.Errorf("Failed to create config directory: %s", err.Error()))
			}

			f, err := os.Create(path.Join(home, configDir, configName) + ".yaml")
			if err != nil {
				panic(fmt.Errorf("Failed to create config file: %s", err.Error()))
			}
			f.Chmod(0644)
			f.Close()

			err = viper.WriteConfig()
			if err != nil {
				panic(fmt.Errorf("Failed to write config file: %s", err.Error()))
			}
		}
	} else {
		Debugf("Loaded config file")
	}
}
