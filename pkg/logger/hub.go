package logger

import "sync"

type Hub struct {
	loggers []Logger
	buffer  LogBuffer
	level   LoggerLevel
	mu      sync.Mutex
}

func (h *Hub) Registrate(logger Logger) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.loggers = append(h.loggers, logger)
}

func (h *Hub) RegisterBuffer(buffer LogBuffer) {
	h.buffer = buffer
}

func (h *Hub) GetLogs() []LogMessage {
	if h.buffer == nil {
		return []LogMessage{}
	}
	return h.buffer.GetLogs()
}

func (h *Hub) SetLevel(lvl LoggerLevel) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.level = lvl
}

func (h *Hub) WithModule(module string) Logger {
	return &ModuleLogger{
		hub:    h,
		module: module,
	}
}

func (h *Hub) Log(lvl LoggerLevel, msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if lvl >= h.level {
		for _, logger := range h.loggers {
			logger.Log(lvl, msg)
		}
	}
}

func (h *Hub) Debug(msg string)    { h.Log(DEBUG, msg) }
func (h *Hub) Info(msg string)     { h.Log(INFO, msg) }
func (h *Hub) Warning(msg string)  { h.Log(WARNING, msg) }
func (h *Hub) Error(msg string)    { h.Log(ERROR, msg) }
func (h *Hub) Critical(msg string) { h.Log(CRITICAL, msg) }

var hub *Hub = nil

func GetLoggerHub() LoggerHub {
	if hub == nil {
		hub = &Hub{
			loggers: []Logger{},
			level:   DEBUG,
		}
	}
	return hub
}
