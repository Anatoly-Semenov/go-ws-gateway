package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Kafka  KafkaConfig  `mapstructure:"kafka"`
	Redis  RedisConfig  `mapstructure:"redis"`
	Logger LoggerConfig `mapstructure:"logger"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"readTimeout"`
	WriteTimeout time.Duration `mapstructure:"writeTimeout"`
}

type KafkaConfig struct {
	BootstrapServers string   `mapstructure:"bootstrapServers"`
	GroupID          string   `mapstructure:"groupId"`
	Topics           []string `mapstructure:"topics"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type LoggerConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func LoadConfig(path string) (*Config, error) {
	v := viper.New()

	setDefaults(v)

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(path)
	v.AddConfigPath(".")

	v.SetEnvPrefix("WSGATEWAY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.readTimeout", 10*time.Second)
	v.SetDefault("server.writeTimeout", 10*time.Second)

	v.SetDefault("kafka.bootstrapServers", "localhost:9092")
	v.SetDefault("kafka.groupId", "ws-gateway")
	v.SetDefault("kafka.topics", []string{"balance-updates", "user-blocks", "notifications"})

	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "json")
}
