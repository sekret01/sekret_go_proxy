package config

type Config struct {
	ConnectionConfig `yaml:"connection"`
	ModuleConfig     `yaml:"modules"`
	LogConfig        `yaml:"logs"`
	SecureConfig     `yaml:"secure"`
	AdminConfig      `yaml:"admin"`
}

type ConnectionConfig struct {
	LocalHost  string `yaml:"localhost"`
	RemoteHost string `yaml:"remotehost"`
}

type ModuleConfig struct {
	TransportType  string `yaml:"transport_type"`
	EncryptorType  string `yaml:"encryptor_type"`
	FramerType     string `yaml:"framer_type"`
	DispatcherType string `yaml:"dispatcher_type"`
	AuthType       string `yaml:"auth_type"`
}

type LogConfig struct {
	LoggerLevel string `yaml:"log_level"`
}

type SecureConfig struct {
	Key32Bytes string `yaml:"key_32_bytes"`
}

type AdminConfig struct {
	AdminHost string `yaml:"adminhost"`
	User      string `yaml:"user"`
	Password  string `yaml:"password"`
}
