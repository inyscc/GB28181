package config

import (
	"github.com/spf13/viper"
)

// MySQLOptions 定义MySQL数据库的配置选项
type MySQLOptions struct {
	Host                  string `json:"host,omitempty" mapstructure:"host"`
	Port                  string `json:"port" mapstructure:"port"`
	Username              string `json:"username,omitempty" mapstructure:"username"`
	Password              string `json:"password,omitempty" mapstructure:"password"`
	Database              string `json:"database,omitempty" mapstructure:"database"`
	MaxIdleConnections    int    `json:"max-idle-connections,omitempty" mapstructure:"max-idle-connections"`
	MaxOpenConnections    int    `json:"max-open-connections,omitempty" mapstructure:"max-open-connections"`
	MaxConnectionLifeTime int64  `json:"max-connection-life-time,omitempty" mapstructure:"max-connection-life-time"`
	LogLevel              int    `json:"log-level,omitempty" mapstructure:"log-level"`
}

func NewMySQLOptions() *MySQLOptions {
	m := &MySQLOptions{
		Host:                  "127.0.0.1",
		Port:                  "3306",
		MaxIdleConnections:    100,
		MaxOpenConnections:    100,
		MaxConnectionLifeTime: 10,
		LogLevel:              1,
	}
	_ = viper.UnmarshalKey("mysql", m)
	return m
}
