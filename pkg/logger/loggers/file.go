package loggers

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type FileLogger struct {
	mu      sync.Mutex
	logChan chan string
}

func (l *FileLogger) Log(lvl logger.LoggerLevel, msg string) {
	timeNow := time.Now().Format(time.RFC3339Nano)
	l.mu.Lock()
	defer l.mu.Unlock()
	log := "[" + logger.GetLevelName(lvl) + "] :: " + timeNow + " :: " + strings.Trim(msg, " \n") + "\n"
	l.logChan <- log
}

func (l *FileLogger) Debug(msg string)    { l.Log(logger.DEBUG, msg) }
func (l *FileLogger) Info(msg string)     { l.Log(logger.INFO, msg) }
func (l *FileLogger) Warning(msg string)  { l.Log(logger.WARNING, msg) }
func (l *FileLogger) Error(msg string)    { l.Log(logger.ERROR, msg) }
func (l *FileLogger) Critical(msg string) { l.Log(logger.CRITICAL, msg) }

func (l *FileLogger) ChanenelReader() {
	file, err := os.OpenFile("logs/logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	canLog := err == nil
	if canLog {
		defer file.Close()
	}
	for log := range l.logChan {
		if canLog {
			if _, err := file.WriteString(log); err != nil {
				continue
			}
		}
	}
}

func NewFileLogger() logger.Logger {
	lg := &FileLogger{
		logChan: make(chan string),
	}
	go lg.ChanenelReader()
	return lg
}
