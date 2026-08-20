package configs

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// ponytail: keep viper global for legacy Get*/GetString backward compat instead of pure struct-only config; upgrade path is inject viper instance or pass Config to callers and drop viper when typed fields cover all usages.

// Config holds all application configurations (typed + legacy viper delegation).
type Config struct {
	App struct {
		Port int    `mapstructure:"PORT" validate:"required,min=1,max=65535"`
		Env  string `mapstructure:"APP_ENV" validate:"required,oneof=development production test"`
		Name string `mapstructure:"APP_NAME" validate:"required"`
	}
	DB struct {
		Host         string        `mapstructure:"DB_HOST" validate:"required"`
		Port         int           `mapstructure:"DB_PORT" validate:"required,min=1,max=65535"`
		User         string        `mapstructure:"DB_USER"`
		Password     string        `mapstructure:"DB_PASSWORD"`
		Name         string        `mapstructure:"DB_NAME"`
		SSLMode      string        `mapstructure:"DB_SSLMODE" validate:"required,oneof=disable allow prefer require verify-ca verify-full"`
		MaxIdleConns int           `mapstructure:"DB_MAX_IDLE_CONNS" validate:"min=0"`
		MaxOpenConns int           `mapstructure:"DB_MAX_OPEN_CONNS" validate:"min=0"`
		MaxLifetime  time.Duration `mapstructure:"DB_MAX_LIFETIME" validate:"required"`
		LogLevel     string        `mapstructure:"DB_LOG_LEVEL" validate:"required,oneof=silent error warn info"`
	}
	CORS struct {
		AllowOrigins     []string `mapstructure:"CORS_ALLOW_ORIGINS"`
		AllowCredentials bool     `mapstructure:"CORS_ALLOW_CREDENTIALS"`
	}
	Log struct {
		Level  string `mapstructure:"LOG_LEVEL" validate:"required,oneof=debug info warn error"`
		Format string `mapstructure:"LOG_FORMAT" validate:"required,oneof=json text"`
	}
	Auth struct {
		SessionDuration time.Duration `mapstructure:"SESSION_DURATION" validate:"required"`
		BcryptCost      int           `mapstructure:"BCRYPT_COST" validate:"required,min=4,max=31"`
	}
}

// NewConfig creates a new Config instance using Viper with typed defaults and validation.
func NewConfig() (*Config, error) {
	// Static defaults (conditional ones handled after unmarshal)
	viper.SetDefault("PORT", 3278)
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_NAME", "Fiberest")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", 5432)
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "")
	viper.SetDefault("DB_NAME", "fiberest")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("DB_MAX_IDLE_CONNS", 10)
	viper.SetDefault("DB_MAX_OPEN_CONNS", 100)
	viper.SetDefault("DB_MAX_LIFETIME", time.Hour)
	viper.SetDefault("DB_LOG_LEVEL", "silent")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("SESSION_DURATION", 168*time.Hour)
	viper.SetDefault("BCRYPT_COST", 10)
	viper.SetDefault("CORS_ALLOW_CREDENTIALS", false)
	// LOG_FORMAT and CORS_ALLOW_ORIGINS are conditional on APP_ENV — set after env resolution

	viper.SetEnvPrefix("")
	viper.AutomaticEnv()

	viper.SetConfigFile(".env")
	_ = viper.ReadInConfig()

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}
	// viper defaults for nested mapstructure keys may not unmarshal reliably;
	// backfill zero values from viper.Get so typed fields and legacy Get* stay in sync.
	if cfg.App.Port == 0 {
		cfg.App.Port = viper.GetInt("PORT")
	}
	if cfg.App.Env == "" {
		cfg.App.Env = viper.GetString("APP_ENV")
	}
	if cfg.App.Name == "" {
		cfg.App.Name = viper.GetString("APP_NAME")
	}
	if cfg.DB.Host == "" {
		cfg.DB.Host = viper.GetString("DB_HOST")
	}
	if cfg.DB.Port == 0 {
		cfg.DB.Port = viper.GetInt("DB_PORT")
	}
	if cfg.DB.SSLMode == "" {
		cfg.DB.SSLMode = viper.GetString("DB_SSLMODE")
	}
	if cfg.DB.MaxIdleConns == 0 {
		cfg.DB.MaxIdleConns = viper.GetInt("DB_MAX_IDLE_CONNS")
	}
	if cfg.DB.MaxOpenConns == 0 {
		cfg.DB.MaxOpenConns = viper.GetInt("DB_MAX_OPEN_CONNS")
	}
	if cfg.DB.MaxLifetime == 0 {
		cfg.DB.MaxLifetime = viper.GetDuration("DB_MAX_LIFETIME")
	}
	if cfg.DB.LogLevel == "" {
		cfg.DB.LogLevel = viper.GetString("DB_LOG_LEVEL")
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = viper.GetString("LOG_LEVEL")
	}
	if cfg.Auth.SessionDuration == 0 {
		cfg.Auth.SessionDuration = viper.GetDuration("SESSION_DURATION")
	}
	if cfg.Auth.BcryptCost == 0 {
		cfg.Auth.BcryptCost = viper.GetInt("BCRYPT_COST")
	}
	if cfg.DB.User == "" {
		cfg.DB.User = viper.GetString("DB_USER")
	}
	if cfg.DB.Password == "" {
		cfg.DB.Password = viper.GetString("DB_PASSWORD")
	}
	if cfg.DB.Name == "" {
		cfg.DB.Name = viper.GetString("DB_NAME")
	}
	cfg.CORS.AllowCredentials = viper.GetBool("CORS_ALLOW_CREDENTIALS")
	if s := viper.GetString("CORS_ALLOW_ORIGINS"); s != "" && len(cfg.CORS.AllowOrigins) == 0 {
		cfg.CORS.AllowOrigins = []string{s}
	}

	// Conditional defaults that depend on APP_ENV
	if cfg.Log.Format == "" {
		if s := viper.GetString("LOG_FORMAT"); s != "" {
			cfg.Log.Format = s
		} else if cfg.App.Env == "production" {
			cfg.Log.Format = "json"
		} else {
			cfg.Log.Format = "text"
		}
		viper.Set("LOG_FORMAT", cfg.Log.Format)
	}
	if len(cfg.CORS.AllowOrigins) == 0 {
		// Only default to * in development; otherwise leave empty (locked down)
		if cfg.App.Env == "development" && !viper.IsSet("CORS_ALLOW_ORIGINS") {
			cfg.CORS.AllowOrigins = []string{"*"}
			viper.Set("CORS_ALLOW_ORIGINS", "*")
		}
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Get retrieves a configuration value by key (legacy, delegates to viper).
func (c *Config) Get(key string) interface{} {
	return viper.Get(key)
}

// GetString retrieves a string configuration value by key (legacy).
func (c *Config) GetString(key string) string {
	return viper.GetString(key)
}

// GetInt retrieves an integer configuration value by key (legacy).
func (c *Config) GetInt(key string) int {
	return viper.GetInt(key)
}

// GetBool retrieves a boolean configuration value by key (legacy).
func (c *Config) GetBool(key string) bool {
	return viper.GetBool(key)
}
