package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // embed tz database so Asia/Ho_Chi_Minh works on minimal images

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	Shop     ShopConfig     `yaml:"shop"`
	App      AppConfig      `yaml:"app"`
	Media    MediaConfig    `yaml:"media"`
	Notify   NotifyConfig   `yaml:"notifications"`
}

// NotifyConfig: push notifications (Firebase Cloud Messaging) and the daily
// planting reminders. Without credentials, reminders are still stored and
// listed in the app, just not pushed.
type NotifyConfig struct {
	// Path of the Firebase service-account JSON key (FCM_CREDENTIALS_FILE).
	CredentialsFile string `yaml:"fcm_credentials_file"`
	// The same key, base64-encoded (FCM_CREDENTIALS_BASE64): handier in an
	// .env file. Used when CredentialsFile is empty.
	CredentialsBase64 string `yaml:"fcm_credentials_base64"`
	// Hour of the day (shop timezone) from which reminders go out (default 7).
	ReminderHour int `yaml:"reminder_hour"`
	// How often the scheduler checks for due reminders (default 10m).
	CheckInterval time.Duration `yaml:"check_interval"`
	// Disabled turns the reminder scheduler off (NOTIFY_DISABLED=true).
	Disabled bool `yaml:"disabled"`
}

// AppConfig holds business-level settings.
type AppConfig struct {
	// Timezone used to group revenue by day and to interpret date-only filters (from/to).
	// Timestamps are stored in UTC (DSN must use loc=UTC).
	Timezone string `yaml:"timezone"`
}

// MediaConfig controls where uploaded files are stored and how their public URL is built.
type MediaConfig struct {
	UploadDir     string `yaml:"upload_dir"`
	PublicBaseURL string `yaml:"public_base_url"` // e.g. https://api.example.com (no trailing slash)
}

// Location returns the configured business timezone (falls back to UTC+7).
func (c *Config) Location() *time.Location {
	if loc, err := time.LoadLocation(c.App.Timezone); err == nil {
		return loc
	}
	return time.FixedZone("ICT", 7*60*60)
}

type ServerConfig struct {
	Addr         string        `yaml:"addr"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

type AuthConfig struct {
	JWTSecret string        `yaml:"jwt_secret"`
	TokenTTL  time.Duration `yaml:"token_ttl"`
}

type ShopConfig struct {
	Name     string `yaml:"name"`
	Phone    string `yaml:"phone"`
	Currency string `yaml:"currency"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyEnvOverrides()
	cfg.setDefaults()
	if err := cfg.validateProduction(os.Getenv("APP_ENV")); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// validateProduction refuses unsafe settings when APP_ENV=production: a
// guessable JWT secret lets anyone mint tokens, and plain-http public URLs
// would be blocked by the app.
func (c *Config) validateProduction(appEnv string) error {
	if appEnv != "production" {
		return nil
	}
	var problems []string
	secret := c.Auth.JWTSecret
	if len(secret) < 32 || strings.Contains(strings.ToLower(secret), "change-me") {
		problems = append(problems, "JWT_SECRET must be a random value of at least 32 characters (e.g. `openssl rand -hex 32`)")
	}
	if c.Database.DSN == "" {
		problems = append(problems, "DB_DSN is required")
	}
	if !strings.HasPrefix(c.Media.PublicBaseURL, "https://") {
		problems = append(problems, "PUBLIC_BASE_URL must be the https URL of the API (e.g. https://7b.truongdx.dev)")
	}
	if len(problems) > 0 {
		return fmt.Errorf("unsafe production config:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("DB_DSN"); v != "" {
		c.Database.DSN = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		c.Auth.JWTSecret = v
	}
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		c.Server.Addr = v
	}
	if v := os.Getenv("APP_TIMEZONE"); v != "" {
		c.App.Timezone = v
	}
	if v := os.Getenv("UPLOAD_DIR"); v != "" {
		c.Media.UploadDir = v
	}
	if v := os.Getenv("PUBLIC_BASE_URL"); v != "" {
		c.Media.PublicBaseURL = v
	}
	if v := os.Getenv("FCM_CREDENTIALS_FILE"); v != "" {
		c.Notify.CredentialsFile = v
	}
	if v := os.Getenv("FCM_CREDENTIALS_BASE64"); v != "" {
		c.Notify.CredentialsBase64 = v
	}
	if v := os.Getenv("REMINDER_HOUR"); v != "" {
		if h, err := strconv.Atoi(v); err == nil {
			c.Notify.ReminderHour = h
		}
	}
	if v := os.Getenv("NOTIFY_DISABLED"); v != "" {
		c.Notify.Disabled = v == "1" || strings.EqualFold(v, "true")
	}
}

func (c *Config) setDefaults() {
	if c.Server.Addr == "" {
		c.Server.Addr = ":2170"
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 15 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 15 * time.Second
	}
	if c.Auth.TokenTTL == 0 {
		c.Auth.TokenTTL = 24 * time.Hour
	}
	if c.Shop.Currency == "" {
		c.Shop.Currency = "VND"
	}
	if c.App.Timezone == "" {
		c.App.Timezone = "Asia/Ho_Chi_Minh"
	}
	if c.Media.UploadDir == "" {
		c.Media.UploadDir = "uploads"
	}
	if c.Notify.ReminderHour <= 0 || c.Notify.ReminderHour > 23 {
		c.Notify.ReminderHour = 7
	}
	if c.Notify.CheckInterval <= 0 {
		c.Notify.CheckInterval = 10 * time.Minute
	}
	c.Media.PublicBaseURL = strings.TrimRight(c.Media.PublicBaseURL, "/")
	if c.Media.PublicBaseURL == "" {
		c.Media.PublicBaseURL = "http://localhost" + c.Server.Addr
	}
}
