package config

type Config struct {
	LocalHost  string `yaml:"localhost"`
	RemoteHost string `yaml:"remotehost"`

	TransportType string `yaml:"transport_type"`
	EncryptorType string `yaml:"encryptor_type"`
	FramerType    string `yaml:"framer_type"`

	LoggerLevel string `yaml:"log_level"`

	Key32Bytes string `yaml:"key_32_bytes"`
}
