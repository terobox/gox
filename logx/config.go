package logx

import "strings"

// Config 日志配置
type Config struct {
	Level         string `json:"level" yaml:"level" mapstructure:"level"`
	Dir           string `json:"dir" yaml:"dir" mapstructure:"dir"`
	Filename      string `json:"filename" yaml:"filename" mapstructure:"filename"`
	MaxSize       int    `json:"max_size" yaml:"max_size" mapstructure:"max_size"`
	MaxAge        int    `json:"max_age" yaml:"max_age" mapstructure:"max_age"`
	MaxBackups    int    `json:"max_backups" yaml:"max_backups" mapstructure:"max_backups"`
	Compress      bool   `json:"compress" yaml:"compress" mapstructure:"compress"`
	ConsoleOutput bool   `json:"console_output" yaml:"console_output" mapstructure:"console_output"`
	DisableFile   bool   `json:"disable_file" yaml:"disable_file" mapstructure:"disable_file"`
}

// DefaultConfig 返回一套合理的默认配置
func DefaultConfig() *Config {
	return &Config{
		Level:         "info",
		Dir:           "logs",
		Filename:      "app.log",
		MaxSize:       100,
		MaxAge:        30,
		MaxBackups:    10,
		Compress:      true,
		ConsoleOutput: true,
		DisableFile:   false,
	}
}

func (c *Config) applyDefaults() {
	def := DefaultConfig()
	if strings.TrimSpace(c.Level) == "" {
		c.Level = def.Level
	}
	if strings.TrimSpace(c.Dir) == "" {
		c.Dir = def.Dir
	}
	if strings.TrimSpace(c.Filename) == "" {
		c.Filename = def.Filename
	}
	if c.MaxSize <= 0 {
		c.MaxSize = def.MaxSize
	}
	if c.MaxAge <= 0 {
		c.MaxAge = def.MaxAge
	}
	if c.MaxBackups <= 0 {
		c.MaxBackups = def.MaxBackups
	}
}
