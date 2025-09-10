package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// уровни логирования
const (
	LevelDebug = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// уровни логирования в строчном формате
var levelNames = map[int]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
	LevelFatal: "FATAL",
}

// структура логгера с поддержкой уровней и ротацией
type Logger struct {
	*log.Logger
	level      int
	file       *os.File
	mu         sync.Mutex // "замок" для безопасности при работе из нескольких потоков
	logPath    string
	maxSize    int64
	maxBackups int
}

var globalLogger *Logger

// инициализация логгера по умолчанию
func init() {
	// по умолчанию логируется в stdout с уровнем INFO
	globalLogger = &Logger{
		Logger:     log.New(os.Stdout, "", 0),
		level:      LevelInfo,
		logPath:    "",
		maxSize:    10 * 1024 * 1024,
		maxBackups: 5,
	}
	// очистка стандартных флагов log.LstdFlags
	globalLogger.SetFlags(0)
}

// создание нового экземпляра логгера
func New(logPath string, level int) (*Logger, error) {
	var output io.Writer = os.Stdout
	var file *os.File
	var err error

	// если есть путь к файлу, то пишем в него
	if logPath != "" {
		// создание директории, если таковой нет
		dir := filepath.Dir(logPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("не удалось создать директорию для логов: %w", err)
		}
		// открытие файла для запси
		file, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("не удалось открыть файл логов: %w", err)
		}
		output = file
	}

	logger := &Logger{
		Logger:     log.New(output, "", 0),
		level:      level,
		file:       file,
		logPath:    logPath,
		maxSize:    10 * 1024 * 1024, // 10 MB по умолчанию
		maxBackups: 5,
	}
	logger.SetFlags(0)
	return logger, nil
}

// инициализация глобального логгера
func InitGlobal(logPath string, level int) error {
	logger, err := New(logPath, level)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// установка уровня логгирования
func (l *Logger) SetLevel(level int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// установка макс. размера файла лога перед ротацией
func (l *Logger) SetMaxSize(size int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maxSize = size
}

// установка макс. кол-во файлов бэкапа
func (l *Logger) SetMaxBackups(count int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maxBackups = count
}
