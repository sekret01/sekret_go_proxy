package loggers

import (
	"fmt"
	"strings"
	"time"

	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type ConsoleLogger struct {
	logChan chan string
}

func (l *ConsoleLogger) Log(lvl logger.LoggerLevel, msg string) {
	timeNow := time.Now().Format(time.RFC3339Nano)
	log := "[" + logger.ColoredLevel(logger.GetLevelName(lvl)) + "] :: " + timeNow + " :: " + strings.Trim(msg, " \n") + "\n"
	select {
	case l.logChan <- log:
	default:
	}
}

func (l *ConsoleLogger) Debug(msg string)    { l.Log(logger.DEBUG, msg) }
func (l *ConsoleLogger) Info(msg string)     { l.Log(logger.INFO, msg) }
func (l *ConsoleLogger) Warning(msg string)  { l.Log(logger.WARNING, msg) }
func (l *ConsoleLogger) Error(msg string)    { l.Log(logger.ERROR, msg) }
func (l *ConsoleLogger) Critical(msg string) { l.Log(logger.CRITICAL, msg) }

func (l *ConsoleLogger) ChanenelReader() {
	for log := range l.logChan {
		fmt.Print(log)
	}
}

func NewConsoleLogger() logger.Logger {
	lg := &ConsoleLogger{
		logChan: make(chan string),
	}
	go lg.ChanenelReader()
	return lg
}
