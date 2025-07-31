package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	t.Run("should load config from flags", func(t *testing.T) {
		args := []string{
			"-env=test",
			"-port=8080",
			"-svr-idle-timeout=1s",
			"-svr-read-timeout=2s",
			"-svr-write-timeout=5s",
			"-db-dsn=mock-dsn",
			"-db-max-open-conns=100",
			"-db-max-idle-conns=50",
			"-db-max-idle-time=20m",
		}

		mockGetEnv := func(key string) string {
			return ""
		}

		expectedConfig := config{}
		expectedConfig.idleTimeout = time.Second
		expectedConfig.readTimeout = 2 * time.Second
		expectedConfig.WriteTimeout = 5 * time.Second
		expectedConfig.env = "test"
		expectedConfig.port = 8080
		expectedConfig.db.dsn = "mock-dsn"
		expectedConfig.db.maxOpenConns = 100
		expectedConfig.db.maxIdleConns = 50
		expectedConfig.db.maxIdleTime = 20 * time.Minute

		actualConfig, err := loadConfig(args, mockGetEnv)
		assert.NoError(t, err)
		assert.Equal(t, expectedConfig, actualConfig)
	})

	t.Run("should error if any flag is invalid", func(t *testing.T) {
		args := []string{
			"-db-dsn=mock-dsn",
			"-db-max-open-conns=100",
			"-db-max-idle-conns=no-int",
			"-db-max-idle-time=20m",
		}

		mockGetEnv := func(key string) string {
			return ""
		}

		expectedConfig := config{}
		actualConfig, err := loadConfig(args, mockGetEnv)
		assert.Error(t, err)
		assert.Equal(
			t,
			"invalid value \"no-int\" for flag -db-max-idle-conns: parse error",
			err.Error(),
		)
		assert.Equal(t, expectedConfig, actualConfig)
	})

	t.Run("should load config from env", func(t *testing.T) {
		args := []string{}

		mockGetEnv := func(key string) string {
			switch key {
			case "PRODUCTS_DB_DSN":
				return "env-dsn"
			case "DB_MAX_OPEN_CONN":
				return "30"
			case "DB_MAX_IDLE_CONN":
				return "15"
			case "DB_MAX_IDLE_TIME":
				return "10"
			case "SERVER_PORT":
				return "5000"
			case "ENV":
				return "test server 2"
			default:
				return ""
			}
		}

		expectedConfig := config{}
		expectedConfig.idleTimeout = time.Minute
		expectedConfig.readTimeout = 5 * time.Second
		expectedConfig.WriteTimeout = 10 * time.Second
		expectedConfig.env = "test server 2"
		expectedConfig.port = 5000
		expectedConfig.db.dsn = "env-dsn"
		expectedConfig.db.maxOpenConns = 30
		expectedConfig.db.maxIdleConns = 15
		expectedConfig.db.maxIdleTime = 10 * time.Minute

		actualConfig, err := loadConfig(args, mockGetEnv)
		assert.NoError(t, err)
		assert.Equal(t, expectedConfig, actualConfig)
	})

	t.Run("should default to env if flag is not provided", func(t *testing.T) {
		args := []string{}

		mockGetEnv := func(key string) string {
			switch key {
			case "PRODUCTS_DB_DSN":
				return "env-dsn"
			case "DB_MAX_OPEN_CONN":
				return "30"
			case "DB_MAX_IDLE_CONN":
				return "15"
			case "DB_MAX_IDLE_TIME":
				return "10"
			case "SERVER_PORT":
				return "5000"
			case "ENV":
				return "test server 2"
			default:
				return ""
			}
		}

		expectedConfig := config{}
		expectedConfig.env = "test server 2"
		expectedConfig.port = 5000
		expectedConfig.idleTimeout = time.Minute
		expectedConfig.readTimeout = 5 * time.Second
		expectedConfig.WriteTimeout = 10 * time.Second
		expectedConfig.db.dsn = "env-dsn"
		expectedConfig.db.maxOpenConns = 30
		expectedConfig.db.maxIdleConns = 15
		expectedConfig.db.maxIdleTime = 10 * time.Minute

		actualConfig, err := loadConfig(args, mockGetEnv)
		assert.NoError(t, err)
		assert.Equal(t, expectedConfig, actualConfig)
	})

	t.Run("should load default values if no flags and no envs", func(t *testing.T) {
		args := []string{}

		mockGetEnv := func(key string) string {
			return ""
		}

		expectedConfig := config{}
		expectedConfig.env = ""
		expectedConfig.port = 4000
		expectedConfig.idleTimeout = time.Minute
		expectedConfig.readTimeout = 5 * time.Second
		expectedConfig.WriteTimeout = 10 * time.Second
		expectedConfig.db.dsn = ""
		expectedConfig.db.maxOpenConns = 25
		expectedConfig.db.maxIdleConns = 25
		expectedConfig.db.maxIdleTime = 15 * time.Minute

		actualConfig, err := loadConfig(args, mockGetEnv)
		assert.NoError(t, err)
		assert.Equal(t, expectedConfig, actualConfig)
	})
}

func TestGetIntEnv(t *testing.T) {
	mockGetEnv := func(key string) string {
		switch key {
		case "PRODUCTS_DB_DSN":
			return "env-dsn"
		case "DB_MAX_OPEN_CONN":
			return "30"
		default:
			return ""
		}
	}

	const defaultValue = 10

	t.Run("should return valid int", func(t *testing.T) {
		actualValue := getIntEnv(mockGetEnv, "DB_MAX_OPEN_CONN", defaultValue)
		assert.Equal(t, 30, actualValue)
	})

	t.Run("should return default value if variable is not int", func(t *testing.T) {
		actualValue := getIntEnv(mockGetEnv, "PRODUCTS_DB_DSN", defaultValue)
		assert.Equal(t, defaultValue, actualValue)
	})

	t.Run("should return default value if variable is missing", func(t *testing.T) {
		actualValue := getIntEnv(mockGetEnv, "MISSING_VAR", defaultValue)
		assert.Equal(t, defaultValue, actualValue)
	})
}
