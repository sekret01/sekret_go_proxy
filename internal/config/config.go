package config

type Config struct {
	LocalHost  string `yaml:"localhost"`
	LocalPort  int    `yaml:"localport"`
	RemoteHost string `yaml:"remotehost"`
	RemotePort int    `yaml:"remoteport"`

	TransportType string `yaml:"transport_type"`
	EncryptorType string `yaml:"encryptor_type"`
	FramerType    string `yaml:"framer_type"`

	LoggerLevel string `yaml:"log_level"`

	Key32Bytes string `yaml:"key_32_bytes"`
}
