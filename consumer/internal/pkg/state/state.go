package state

import (
	"io/ioutil"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

type StateConfig struct {
	Sensors map[string]string `yaml:"sensors"`
}

func ReadState() (StateConfig, error) {
	state := StateConfig{}
	viper.SetConfigName("state")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		return state, err
	}
	err = viper.Unmarshal(&state)
	if err != nil {
		return state, err
	}
	return state, nil
}

func WriteState(state StateConfig) error {
	d, err := yaml.Marshal(&state)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("state.yml", d, 0644)
	if err != nil {
		return err
	}
	return nil
}
