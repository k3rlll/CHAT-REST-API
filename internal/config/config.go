package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPServer struct {
	Port        int           `yaml:"port" env:"SERVER_PORT" env-default:"8082"`
	Mode        string        `yaml:"mode" env:"SERVER_MODE" env-default:"debug"`
	Host        string        `yaml:"host" env:"SERVER_HOST" env-default:"localhost"`
	Timeout     time.Duration `yaml:"timeout" env:"SERVER_TIMEOUT" env-default:"15"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env:"SERVER_IDLE_TIMEOUT" env-default:"60"`
}

type Config struct {
	Env      string     `yaml:"env" env:"ENV" env-default:"development"`
	Server   HTTPServer `yaml:"server"`
	Database struct {
		Type     string `yaml:"type"`
		Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
		Port     int    `yaml:"port" env:"DB_PORT" env-default:"8081"`
		User     string `yaml:"user" env:"DB_USER" env-default:"postgres"`
		Password string `yaml:"password" env:"DB_PASSWORD"`
	} `yaml:"database"`
}

func MustLoadConfig() *Config {
	path := fetchConfigPath()
	if path == "" {
		panic("config path is not provided")
	}
	return MustLoadPath(path)
}

func MustLoadPath(configPath string) *Config {

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("cannot read config: " + err.Error())
	}

	return &cfg
}

func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "Path to config file")
	flag.Parse()

	if res != "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
