package config

type Config struct {
	LocalHost  string `yaml:"localhost"`
	LocalPort  int    `yaml:"localport"`
	RemoteHost string `yaml:"remotehost"`
	RemotePort int    `yaml:"remoteport"`
}
