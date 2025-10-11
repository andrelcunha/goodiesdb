package server

import "os"

type Config struct {
	Host     string
	Port     string
	Password string
	UseRDB   bool
	UseAOF   bool
	Version  string
	DataDir  string
}

func NewConfig() *Config {
	return &Config{
		Port:     "6379",
		Password: "guest",
		UseRDB:   true,
		UseAOF:   true,
		DataDir:  "data",
	}
}

// LoadFromEnv loads the configuration from environment variables
func (c *Config) LoadFromEnv() {
	if host := os.Getenv("HOST"); host != "" {
		c.Host = host
	}
	if port := os.Getenv("PORT"); port != "" {
		c.Port = port
	}
	if password := os.Getenv("PASSWORD"); password != "" {
		c.Password = password
	}
	if useRDB := os.Getenv("USE_RDB"); useRDB != "" {
		c.UseRDB = useRDB == "true"
	}
	if useAOF := os.Getenv("USE_AOF"); useAOF != "" {
		c.UseAOF = useAOF == "true"
	}
	if dataDir := os.Getenv("DATA_DIR"); dataDir != "" {
		c.DataDir = dataDir
	}
}
