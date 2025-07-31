package main

import (
	"flag"
	"strconv"
	"time"
)

type config struct {
	env          string
	port         int
	idleTimeout  time.Duration
	readTimeout  time.Duration
	WriteTimeout time.Duration
	db           struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
}

// The loadConfig() function returns configuration data for running the product service.
// It reads the command line flags passed upon starting the service. If any of the flags
// is not provided, it defaults to reading it from the environment variables.
func loadConfig(args []string, getEnv func(key string) string) (config, error) {
	var cfg config

	// Use a new FlagSet for isolation
	fs := flag.NewFlagSet("config", flag.ContinueOnError)

	fs.IntVar(&cfg.port, "port", getIntEnv(getEnv, "SERVER_PORT", 4000), "API server port")
	fs.StringVar(
		&cfg.env,
		"env",
		getEnv("ENV"),
		"Environment (development|staging|production)",
	)
	fs.DurationVar(&cfg.idleTimeout, "svr-idle-timeout", time.Minute, "API server idle timeout")
	fs.DurationVar(&cfg.readTimeout, "svr-read-timeout", 5*time.Second, "API server idle timeout")
	fs.DurationVar(&cfg.WriteTimeout, "svr-write-timeout", 10*time.Second, "API server write timeout")

	//Read db configurations
	fs.StringVar(&cfg.db.dsn, "db-dsn", getEnv("PRODUCTS_DB_DSN"), "PostgreSQL DSN")
	fs.IntVar(
		&cfg.db.maxOpenConns,
		"db-max-open-conns",
		getIntEnv(getEnv, "DB_MAX_OPEN_CONN", 25),
		"PostgreSQL max open connections",
	)
	fs.IntVar(
		&cfg.db.maxIdleConns,
		"db-max-idle-conns",
		getIntEnv(getEnv, "DB_MAX_IDLE_CONN", 25),
		"PostgreSQL max idle connections",
	)
	fs.DurationVar(
		&cfg.db.maxIdleTime,
		"db-max-idle-time",
		time.Duration(getIntEnv(getEnv, "DB_MAX_IDLE_TIME", 15))*time.Minute,
		"PostgreSQL max connection idle time",
	)

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}

	return cfg, nil
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
