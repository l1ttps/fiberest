package configs

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/subosito/gotenv"
)

// Config holds all application configuration.
//
// Precedence per key: process env > .env file > `default` tag.
// Only env vars (not .env entries) count as "explicitly set" for the
// conditional defaults resolved in NewConfig.
type Config struct {
	App struct {
		Port int    `env:"PORT" default:"3278" validate:"required,min=1,max=65535"`
		Env  string `env:"APP_ENV" default:"development" validate:"required,oneof=development production test"`
		Name string `env:"APP_NAME" default:"Fiberest" validate:"required"`
	}
	DB struct {
		Host         string        `env:"DB_HOST" default:"localhost" validate:"required"`
		Port         int           `env:"DB_PORT" default:"5432" validate:"required,min=1,max=65535"`
		User         string        `env:"DB_USER" default:"postgres"`
		Password     string        `env:"DB_PASSWORD"`
		Name         string        `env:"DB_NAME" default:"fiberest"`
		SSLMode      string        `env:"DB_SSLMODE" default:"disable" validate:"required,oneof=disable allow prefer require verify-ca verify-full"`
		MaxIdleConns int           `env:"DB_MAX_IDLE_CONNS" default:"10" validate:"min=0"`
		MaxOpenConns int           `env:"DB_MAX_OPEN_CONNS" default:"100" validate:"min=0"`
		MaxLifetime  time.Duration `env:"DB_MAX_LIFETIME" default:"1h" validate:"required"`
		LogLevel     string        `env:"DB_LOG_LEVEL" default:"silent" validate:"required,oneof=silent error warn info"`
		// AutoMigrate defaults to true outside production; set DB_AUTO_MIGRATE to override.
		AutoMigrate bool `env:"DB_AUTO_MIGRATE"`
	}
	CORS struct {
		AllowOrigins     []string `env:"CORS_ALLOW_ORIGINS"` // comma separated
		AllowCredentials bool     `env:"CORS_ALLOW_CREDENTIALS"`
	}
	Log struct {
		Level  string `env:"LOG_LEVEL" default:"info" validate:"required,oneof=debug info warn error"`
		Format string `env:"LOG_FORMAT" validate:"omitempty,oneof=json text"` // defaults to json in production, text otherwise
	}
	Auth struct {
		SessionDuration time.Duration `env:"SESSION_DURATION" default:"168h" validate:"required"`
		BcryptCost      int           `env:"BCRYPT_COST" default:"10" validate:"required,min=4,max=31"`
		// CookieSecure defaults to true in production; set SESSION_COOKIE_SECURE to override.
		CookieSecure bool `env:"SESSION_COOKIE_SECURE"`
	}
}

// NewConfig loads .env (when present), fills Config from the environment and validates it.
func NewConfig() (*Config, error) {
	if err := gotenv.Load(".env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	cfg := &Config{}
	if err := load(reflect.ValueOf(cfg).Elem()); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Conditional defaults that depend on APP_ENV.
	if cfg.Log.Format == "" {
		cfg.Log.Format = "text"
		if cfg.App.Env == "production" {
			cfg.Log.Format = "json"
		}
	}
	if len(cfg.CORS.AllowOrigins) == 0 && cfg.App.Env == "development" {
		cfg.CORS.AllowOrigins = []string{"*"}
	}
	if os.Getenv("DB_AUTO_MIGRATE") == "" {
		cfg.DB.AutoMigrate = cfg.App.Env != "production"
	}
	if os.Getenv("SESSION_COOKIE_SECURE") == "" {
		cfg.Auth.CookieSecure = cfg.App.Env == "production"
	}

	if err := validator.New().Struct(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}

var durationType = reflect.TypeOf(time.Duration(0))

func load(cfg reflect.Value) error {
	for i := 0; i < cfg.NumField(); i++ {
		section := cfg.Field(i)
		if section.Kind() != reflect.Struct {
			continue
		}
		st := section.Type()
		for j := 0; j < st.NumField(); j++ {
			field := st.Field(j)
			key := field.Tag.Get("env")
			if key == "" {
				continue
			}
			raw := os.Getenv(key)
			if raw == "" {
				raw = field.Tag.Get("default")
			}
			if raw == "" {
				continue
			}
			if err := setValue(section.Field(j), raw); err != nil {
				return fmt.Errorf("%s=%q: %w", key, raw, err)
			}
		}
	}
	return nil
}

func setValue(field reflect.Value, raw string) error {
	if field.Type() == durationType {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return err
		}
		field.SetInt(int64(d))
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		field.SetBool(b)
	case reflect.Slice:
		if field.Type().Elem().Kind() != reflect.String {
			return fmt.Errorf("unsupported slice type %s", field.Type())
		}
		out := field.Slice(0, 0)
		for _, part := range strings.Split(raw, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = reflect.Append(out, reflect.ValueOf(part))
			}
		}
		field.Set(out)
	default:
		return fmt.Errorf("unsupported type %s", field.Type())
	}
	return nil
}
