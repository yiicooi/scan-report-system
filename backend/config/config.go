package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	OSS      OSSConfig
	AI       AIConfig
}

type ServerConfig struct {
	Port string
	Mode string // debug | release
}

type DatabaseConfig struct {
	DSN string
}

type JWTConfig struct {
	Secret          string
	ExpireHours     int
	RefreshExpHours int
}

type OSSConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
	Domain          string
}

type AIConfig struct {
	Provider        string
	DeepSeekAPIKey  string
	DeepSeekBaseURL string
	DeepSeekModel   string
}

var Cfg Config

func Load() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("jwt.expireHours", 24)
	viper.SetDefault("jwt.refreshExpHours", 168)
	viper.SetDefault("ai.provider", "deepseek")
	viper.SetDefault("ai.deepSeekBaseURL", "https://api.deepseek.com")
	viper.SetDefault("ai.deepSeekModel", "deepseek-chat")

	envBindings := map[string]string{
		"server.port":         "SERVER_PORT",
		"server.mode":         "SERVER_MODE",
		"database.dsn":        "DATABASE_DSN",
		"jwt.secret":          "JWT_SECRET",
		"jwt.expireHours":     "JWT_EXPIRE_HOURS",
		"jwt.refreshExpHours": "JWT_REFRESH_EXP_HOURS",
		"oss.endpoint":        "OSS_ENDPOINT",
		"oss.accessKeyID":     "OSS_ACCESS_KEY_ID",
		"oss.accessKeySecret": "OSS_ACCESS_KEY_SECRET",
		"oss.bucketName":      "OSS_BUCKET_NAME",
		"oss.domain":          "OSS_DOMAIN",
		"ai.provider":         "AI_PROVIDER",
		"ai.deepSeekAPIKey":   "DEEPSEEK_API_KEY",
		"ai.deepSeekBaseURL":  "DEEPSEEK_BASE_URL",
		"ai.deepSeekModel":    "DEEPSEEK_MODEL",
	}
	for key, env := range envBindings {
		if err := viper.BindEnv(key, env); err != nil {
			log.Fatalf("failed to bind env %s: %v", env, err)
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("config file not found, using env/defaults: %v", err)
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}
}
