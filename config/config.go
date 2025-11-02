package config

import (
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Env                string            `mapstructure:"env"`
	LogLevel           string            `mapstructure:"log_level"`
	ServiceName        string            `mapstructure:"service_name"`
	Port               string            `mapstructure:"port"`
	Version            string            `mapstructure:"version"`
	RenderPagePath     string            `mapstructure:"render_page_path"`
	HeadlessBrowser    bool              `mapstructure:"headless_browser"`
	BrowsersCount      int               `mapstructure:"browsers_count"`
	EnableLeakless     bool              `mapstructure:"enable_leakless"`
	TakeScreenshot     bool              `mapstructure:"take_screenshot"`
	BrowserWait        time.Duration     `mapstructure:"browser_wait"`
	RenderTimeout      time.Duration     `mapstructure:"render_timeout"`
	HttpServerSettings *HttpServerConfig `mapstructure:"http_server"`
}

type HttpServerConfig struct {
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

func MustLoad() *Config {
	viper.AddConfigPath(path.Join("."))
	viper.SetConfigName("config")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		slog.Error("can't initialize config file.", slog.String("err", err.Error()))
		os.Exit(1)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		slog.Error("error unmarshalling viper config.", slog.String("err", err.Error()))
		os.Exit(1)
	}

	return &cfg
}
