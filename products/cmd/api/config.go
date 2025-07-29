package main

import (
	"flag"
	"strconv"
	"time"
)

type config struct {
	db struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
}

// The loadConfig() function returns configuration data for running the product service.
// It reads the command line flags passed upon starting the service. If any of the flags
// is not provided, it defaults to reading it from the environment variables.
func loadConfig(getEnv func(key string) string) config {
	var cfg config

	//Read db configurations
	flag.StringVar(&cfg.db.dsn, "db-dsn", getEnv("PRODUCTS_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(
		&cfg.db.maxOpenConns,
		"db-max-open-conns",
		getIntEnv(getEnv, "DB_MAX_OPEN_CONN", 25),
		"PostgreSQL max open connections",
	)
	flag.IntVar(
		&cfg.db.maxIdleConns,
		"db-max-idle-conns",
		getIntEnv(getEnv, "DB_MAX_IDLE_CONN", 25),
		"PostgreSQL max idle connections",
	)
	flag.DurationVar(
		&cfg.db.maxIdleTime,
		"db-max-idle-time",
		time.Duration(getIntEnv(getEnv, "DB_MAX_IDLE_TIME", 25))*time.Minute,
		"PostgreSQL max connection idle time",
	)

	return cfg
}

func getIntEnv(getEnv func(key string) string, key string, defaultValue int) int {
	valStr := getEnv(key)
	if valStr == "" {
		return defaultValue
	}

	valInt, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultValue
	}

	return valInt
}
