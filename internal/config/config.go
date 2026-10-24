package config

type Config struct {
	ConnectionConfig `yaml:"connection"`
	ModuleConfig     `yaml:"modules"`
	LogConfig        `yaml:"logs"`
	AdminConfig      `yaml:"admin"`
	VPN              `yaml:"vpn"`
}

type ConnectionConfig struct {
	ConnectionType string `yaml:"connection_type"`
	LocalHost      string `yaml:"localhost"`
	RemoteHost     string `yaml:"remotehost"`
}

type ModuleConfig struct {
	TransportType  string   `yaml:"transport_type"`
	EncryptorType  string   `yaml:"encryptor_type"`
	FramerType     string   `yaml:"framer_type"`
	DispatcherType string   `yaml:"dispatcher_type"`
	AuthType       string   `yaml:"auth_type"`
	HandshakeSteps []string `yaml:"handshake_steps"`
}

type LogConfig struct {
	LoggerLevel string `yaml:"log_level"`
}

type AdminConfig struct {
	AdminHost      string `yaml:"adminhost"`
	Authentication string `yaml:"authentication"`
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	AutoRunProxy   bool   `yaml:"auto_run_proxy"`
}

type VPN struct {
	Subnet string `yaml:"subnet"`
}
