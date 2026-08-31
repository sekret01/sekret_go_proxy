package logger

type LoggerHub interface {
	Registrate(logger Logger)
	SetLevel(lvl LoggerLevel)
	WithModule(module string) Logger
	Log(lvl LoggerLevel, msg string)
	Debug(msg string)
	Info(msg string)
	Warning(msg string)
	Error(msg string)
	Critical(msg string)
}

type Logger interface {
	Log(lvl LoggerLevel, msg string)
	Debug(msg string)
	Info(msg string)
	Warning(msg string)
	Error(msg string)
	Critical(msg string)
}

// CONST

type LoggerLevel byte

const (
	DEBUG    LoggerLevel = 0x01
	INFO     LoggerLevel = 0x02
	WARNING  LoggerLevel = 0x03
	ERROR    LoggerLevel = 0x04
	CRITICAL LoggerLevel = 0x05
)

const (
	DebugStr    = "DEBUG"
	InfoStr     = "INFO"
	WarningStr  = "WARNING"
	ErrorStr    = "ERROR"
	CriticalStr = "CRITICAL"
)

func GetLevelName(lvl LoggerLevel) string {
	switch lvl {
	case DEBUG:
		return DebugStr
	case INFO:
		return InfoStr
	case WARNING:
		return WarningStr
	case ERROR:
		return ErrorStr
	case CRITICAL:
		return CriticalStr
	default:
		return "UNKNOWN"
	}
}
