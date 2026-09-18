package loggers

import (
	"sync"
	"time"

	"github.com/sekret01/sekret_go_proxy/pkg/logger"
)

type LogMessage struct {
	Time  string `json:"time"`
	Level string `json:"level"`
	Msg   string `json:"msg"`
	Count int    `json:"count"`
}

type BufferLogger struct {
	mu     sync.Mutex
	buffer []*LogMessage
	maxLen int
}

func (l *BufferLogger) Log(lvl logger.LoggerLevel, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.buffer[len(l.buffer)-1].Msg == msg {
		l.buffer[len(l.buffer)-1].Count++
		return
	}
	timeNow := time.Now().Format(time.RFC3339Nano)
	newLog := &LogMessage{
		Time:  timeNow,
		Level: logger.GetLevelName(lvl),
		Msg:   msg,
		Count: 1,
	}
	l.buffer = append(l.buffer, newLog)
	l.validateLen()
}

func (l *BufferLogger) Debug(msg string)    { l.Log(logger.DEBUG, msg) }
func (l *BufferLogger) Info(msg string)     { l.Log(logger.INFO, msg) }
func (l *BufferLogger) Warning(msg string)  { l.Log(logger.WARNING, msg) }
func (l *BufferLogger) Error(msg string)    { l.Log(logger.ERROR, msg) }
func (l *BufferLogger) Critical(msg string) { l.Log(logger.CRITICAL, msg) }

func (l *BufferLogger) validateLen() {
	if len(l.buffer) > l.maxLen {
		delta := len(l.buffer) - l.maxLen
		for i := 0; i < delta; i++ {
			l.buffer[i] = nil
		}
		l.buffer = l.buffer[delta:]
	}
}

func NewBufferLogger() logger.Logger {
	return &BufferLogger{
		buffer: []*LogMessage{},
		maxLen: 100,
	}
}
