package config

type Config struct {
	LocalHost  string `yaml:"localhost"`
	RemoteHost string `yaml:"remotehost"`
	AdminHost  string `yaml:"adminhost"`

	TransportType  string `yaml:"transport_type"`
	EncryptorType  string `yaml:"encryptor_type"`
	FramerType     string `yaml:"framer_type"`
	DispatcherType string `yaml:"dispatcher_type"`
	AuthType       string `yaml:"auth_type"`

	LoggerLevel string `yaml:"log_level"`

	Key32Bytes string `yaml:"key_32_bytes"`
}
