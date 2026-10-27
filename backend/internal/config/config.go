package config

import "os"

type Config struct {
	Port      string
	DataDir   string
	MLURL     string
	PublicURL string
}

func Load() Config {
	return Config{
		Port:      getenv("PORT", "8000"),
		DataDir:   getenv("DATA_DIR", "/data"),
		MLURL:     os.Getenv("ML_URL"),
		PublicURL: getenv("PUBLIC_URL", "http://localhost:8000"),
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
