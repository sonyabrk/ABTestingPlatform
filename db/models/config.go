package models

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		DBName   string `yaml:"dbname"`
		SSLMode  string `yaml:"sslmode"`
	} `yaml:"database"`

	Logging struct {
		Level      string `yaml:"level"`       // уровень логирования (info/error)
		File       string `yaml:"file"`        // путь к файлу логов
		MaxSize    int64  `yaml:"max_size"`    // макс. размер файла (байты)
		MaxBackups int    `yaml:"max_backups"` // макс. количество бэкапов
	} `yaml:"logging"`
}

func LoadConfig(path string) (*Config, error) {
	config := &Config{}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)
	if err := d.Decode(config); err != nil {
		return nil, err
	}

	// установка значений по умолчанию для логирования
	if config.Logging.MaxSize == 0 {
		config.Logging.MaxSize = 10 * 1024 * 1024 // 10MB по умолчанию
	}
	if config.Logging.MaxBackups == 0 {
		config.Logging.MaxBackups = 5 // 5 бэкапов по умолчанию
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "info" // info по умолчанию
	}

	return config, nil
}
