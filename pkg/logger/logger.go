package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
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

// закрытие файла лога
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// проверка, нужно ли делать ротацию логов
func (l *Logger) shouldRotate() bool {
	if l.file == nil || l.logPath == "" {
		return false
	}
	info, err := l.file.Stat()
	if err != nil {
		return false
	}
	return info.Size() >= l.maxSize
}

// ротация лог-файлов
func (l *Logger) rotate() error {
	if l.file == nil {
		return nil
	}
	// хакрытие текущего файла
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("ошибка при закрытии файла лога: %w", err)
	}
	for i := l.maxBackups - 1; i > 0; i-- {
		oldPath := fmt.Sprintf("%s.%d", l.logPath, i)
		newPath := fmt.Sprintf("%s.%d", l.logPath, i+1)
		if _, err := os.Stat(oldPath); err == nil {
			if i >= l.maxBackups {
				// удаление самого старого, если превышен лимит
				os.Remove(oldPath)
			} else {
				os.Rename(oldPath, newPath)
			}
		}
	}
	// переименовываем текущий файл
	if _, err := os.Stat(l.logPath); err == nil {
		os.Rename(l.logPath, l.logPath+".1")
	}
	// создаем новый файл
	file, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("не удалось создать новый файл лога: %w", err)
	}

	l.file = file
	l.SetOutput(file)
	return nil
}

// внутренний метод логирования
func (l *Logger) log(level int, callerDepth int, format string, args ...interface{}) {
	// не логируем, если уровень меньше установленного
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	// проверка надобности ротации
	if l.shouldRotate() {
		if err := l.rotate(); err != nil {
			fmt.Printf("Ошибка ротации логов: %v\n", err)
		}
	}
	// получение информации о вызывающем коде
	_, file, line, ok := runtime.Caller(callerDepth)
	if !ok {
		file = "unknown"
		line = 0
	} else {
		// оставляем только имя файла
		file = filepath.Base(file)
	}

	// форматируем сообщение
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelName := levelNames[level]
	message := fmt.Sprintf(format, args...)
	// формируем строку лога
	logEntry := fmt.Sprintf("%s [%s] %s:%d %s", timestamp, levelName, file, line, message)
	// выводим лог
	l.Output(0, logEntry)
}

// Debug логирует сообщение уровня DEBUG
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, 3, format, args...)
}

// Info логирует сообщение уровня INFO
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, 3, format, args...)
}

// Warn логирует сообщение уровня WARN
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, 3, format, args...)
}

// Error логирует сообщение уровня ERROR
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, 3, format, args...)
}

// Fatal логирует сообщение уровня FATAL и завершает программу
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LevelFatal, 3, format, args...)
	os.Exit(1)
}

// ниже глобальные функции для удобства логирования уровней

func Debug(format string, args ...interface{}) {
	globalLogger.log(LevelDebug, 4, format, args...)
}
func Info(format string, args ...interface{}) {
	globalLogger.log(LevelInfo, 4, format, args...)
}
func Warn(format string, args ...interface{}) {
	globalLogger.log(LevelWarn, 4, format, args...)
}
func Error(format string, args ...interface{}) {
	globalLogger.log(LevelError, 4, format, args...)
}
func Fatal(format string, args ...interface{}) {
	globalLogger.log(LevelFatal, 4, format, args...)
	os.Exit(1)
}

// получение глобального экземпляра логгера
func GetGlobal() *Logger {
	return globalLogger
}
