package logger

type ColorString string

const (
	Reset  ColorString = "\033[0m"
	Red    ColorString = "\033[31m"
	Green  ColorString = "\033[32m"
	Yellow ColorString = "\033[33m"
	Blue   ColorString = "\033[34m"
	Purple ColorString = "\033[35m"
	Cyan   ColorString = "\033[36m"
	White  ColorString = "\033[37m"
)

func Colored(color ColorString, msg string) string {
	result := ""
	switch color {
	case Red:
		result += string(Red)
	case Green:
		result += string(Green)
	case Yellow:
		result += string(Yellow)
	case Blue:
		result += string(Blue)
	case Purple:
		result += string(Purple)
	case Cyan:
		result += string(Cyan)
	case White:
		result += string(White)
	case Reset:
		result += string(Reset)
	default:
		result += string(Reset)
	}
	result += msg + string(Reset)
	return result
}

func ColoredLevel(level string) string {
	switch level {
	case DebugStr:
		return Colored(Cyan, level)
	case InfoStr:
		return Colored(Green, level)
	case WarningStr:
		return Colored(Yellow, level)
	case ErrorStr:
		return Colored(Red, level)
	case CriticalStr:
		return Colored(Red, level)
	default:
		return level
	}
}
