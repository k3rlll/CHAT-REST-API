package config

import (
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Metrics struct {
	Enabled bool   `yaml:"enabled" env:"METRICS_ENABLED" env-default:"true"`
	Port    int    `yaml:"port" env:"METRICS_PORT" env-default:"9090"`
	Host    string `yaml:"host" env:"METRICS_HOST" env-default:"localhost"`
}

type Server struct {
	Port        int           `yaml:"port" env:"SERVER_PORT" env-default:"8082"`
	Mode        string        `yaml:"mode" env:"SERVER_MODE" env-default:"debug"`
	Host        string        `yaml:"host" env:"SERVER_HOST" env-default:"localhost"`
	Timeout     time.Duration `yaml:"timeout" env:"SERVER_TIMEOUT" env-default:"15"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env:"SERVER_IDLE_TIMEOUT" env-default:"60"`
}

type Postgres struct {
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port     int    `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User     string `yaml:"user" env:"DB_USER" env-default:"postgres"`
	Password string `yaml:"password" env:"DB_PASSWORD"`
	DBName   string `yaml:"dbname" env:"DB_NAME" env-default:"postgres"`
}

type Kafka struct {
	Brokers       []string `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092"`
	Topic         string   `yaml:"topic" env:"KAFKA_TOPIC" env-default:"chat_messages"`
	ConsumerGroup string   `yaml:"consumer_group" env:"KAFKA_CONSUMER_GROUP" env-default:"chat_service_group"`
}

type MongoDB struct {
	URI      string `yaml:"uri" env:"MONGO_URI" env-default:"mongodb://localhost:27017"`
	Database string `yaml:"database" env:"MONGO_DATABASE" env-default:"chatdb"`
	Ports    int    `yaml:"ports" env:"MONGO_PORTS" env-default:"27017"`
}

type Redis struct {
	Host         string        `yaml:"host" env:"REDIS_HOST" env-default:"localhost"`
	Port         int           `yaml:"port" env:"REDIS_PORT" env-default:"6379"`
	Password     string        `yaml:"password" env:"REDIS_PASSWORD"`
	DB           int           `yaml:"db" env:"REDIS_DB" env-default:"0"`
	DialTimeout  time.Duration `yaml:"dial_timeout" env:"REDIS_DIAL_TIMEOUT" env-default:"5s"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env:"REDIS_READ_TIMEOUT" env-default:"3s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env:"REDIS_WRITE_TIMEOUT" env-default:"3s"`
	PoolSize     int           `yaml:"pool_size" env:"REDIS_POOL_SIZE" env-default:"10"`
	MinIdleConns int           `yaml:"min_idle_conns" env:"REDIS_MIN_IDLE_CONNS" env-default:"2"`
}

type Auth struct {
	TokenTTL time.Duration `yaml:"token_ttl" env:"AUTH_TOKEN_TTL" env-default:"15m"`
}

type Config struct {
	Env      string   `yaml:"env" env:"ENV" env-default:"development"`
	Server   Server   `yaml:"server"`
	Postgres Postgres `yaml:"postgres"`
	MongoDB  MongoDB  `yaml:"mongodb"`
	Redis    Redis    `yaml:"redis"`
	Kafka    Kafka    `yaml:"kafka"`
	Metrics  Metrics  `yaml:"metrics"`
	Auth     Auth     `yaml:"auth"`
}

type EnvConfig struct {
	ConfigPath string `env:"CONFIG_PATH"`
	SecretKey  string `env:"MY_SECRET_KEY"`
}

var configPathFlag string

func init() {
	flag.StringVar(&configPathFlag, "config", "", "Path to the config file")
}

func (e *EnvConfig) MySecretKey() string {
	return e.SecretKey
}

func MySecretKey() string {
	envConfig := &EnvConfig{
		SecretKey: "",
	}
	err := cleanenv.ReadEnv(envConfig)
	if err != nil {
		panic("cannot read env variables: " + err.Error())
	}
	return envConfig.MySecretKey()
}

func (c *Config) DatabaseDSN() string {
	return "postgres://" +
		c.Postgres.User + ":" +
		c.Postgres.Password + "@" +
		c.Postgres.Host + ":" +
		strconv.Itoa(c.Postgres.Port) + "/" +
		c.Postgres.DBName + "?sslmode=disable"
}

func (c *Config) MongoURI() string {
	return c.MongoDB.URI
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
	if !flag.Parsed() {
		flag.Parse()
	}

	res = configPathFlag

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	if res == "" {
		panic("config path is not provided")
	}

	return res
}
