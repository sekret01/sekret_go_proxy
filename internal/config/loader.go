package config

import (
	"os"
	"fmt"
	"gopkg.in/yaml.v3"
)

func LoadConfig(configPath string) (*Config, error) {
	fmt.Println("start read config")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	fmt.Println("read, start load yaml")
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	fmt.Println("end, return config")
	return &config, nil
}
