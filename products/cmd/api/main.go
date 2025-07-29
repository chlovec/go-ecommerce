package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := loadConfig(os.Getenv)
	exitCode := run(cfg, logger, sql.Open)
	os.Exit(exitCode)
}

func run(
	cfg config,
	logger *slog.Logger,
	sqlOpen func(driverName string, dataSourceName string) (*sql.DB, error),
) int {
	//1. Create a db connection pool
	db, err := openDB(cfg, sqlOpen)
	if err != nil {
		logger.Error(err.Error())
		return 1
	}

	// Defer a call to db.Close() so that the connection pool is closed before the
	// main() function exits.
	defer db.Close()

	// Log a message to say that the connection pool has been successfully
	// established.
	logger.Info("database connection pool established")

	//2. Listen to and handle requests

	return 0
}

// The openDB() function returns a sql.DB connection pool that will be used by
// with the service to connect to the database and perform database operations.
func openDB(
	cfg config,
	sqlOpen func(driverName string, dataSourceName string) (*sql.DB, error),
) (*sql.DB, error) {
	// Use sql.Open() to create an empty connection pool, using the DSN from the config
	// struct.
	db, err := sqlOpen("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	// Set the maximum number of open (in-use + idle) connections in the pool.
	// Passing a value less than or equal to 0 will mean there is no limit.
	db.SetMaxOpenConns(cfg.db.maxOpenConns)

	// Set the maximum number of idle connections in the pool.
	// Passing a value less than or equal to 0 will mean there is no limit.
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	// Set the maximum idle timeout for connections in the pool.
	// Passing a duration less than or equal to 0 will mean that
	// connections are not closed due to their idle time.
	db.SetConnMaxIdleTime(cfg.db.maxIdleTime)

	// Create a context with a 5-second timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use PingContext() to establish a new connection to the database, passing in the
	// context we created above as a parameter. If the connection couldn't be
	// established successfully within the 5-second deadline, then this will return an
	// error. If we get this error, or any other, we close the connection pool and
	// return the error.
	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	// Return the sql.DB connection pool.
	return db, nil
}
