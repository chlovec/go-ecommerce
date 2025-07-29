package main

import (
	"bytes"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	args := []string{
		"-db-dsn=mock-dsn",
		"-db-max-open-conns=100",
		"-db-max-idle-conns=50",
		"-db-max-idle-time=20m",
	}
	var buf bytes.Buffer
	logger := newLogger(&buf)

	t.Run("should return 0 if server starts successfully", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()
		mock.ExpectClose()

		mockSQLOpen := func(driverName, dsn string) (*sql.DB, error) {
			return db, nil
		}

		exitCode := run(args, logger, mockSQLOpen)
		assert.Equal(t, exitCode, 0)

		logOutput := buf.String()
		assert.Contains(t, logOutput, "database connection pool established")

		buf.Reset()
	})

	t.Run("should fail if flag error", func(t *testing.T) {
		invalidArgs := []string{
			"-db-dsn=mock-dsn",
			"-db-max-open-conns=no-int",
			"-db-max-idle-conns=50",
			"-db-max-idle-time=20m",
		}

		mockSQLOpen := func(driverName, dsn string) (*sql.DB, error) {
			return nil, nil
		}

		exitCode := run(invalidArgs, logger, mockSQLOpen)
		assert.Equal(t, exitCode, 1)

		logOutput := buf.String()
		assert.Contains(
			t,
			logOutput,
			`"invalid value \"no-int\" for flag -db-max-open-conns: parse error"`,
		)

		buf.Reset()
	})

	t.Run("should return 1 if fails to establish database connection pool", func(t *testing.T) {
		mockSQLOpen := func(driverName, dsn string) (*sql.DB, error) {
			return nil, errors.New("database error")
		}

		exitCode := run(args, logger, mockSQLOpen)
		assert.Equal(t, exitCode, 1)

		logOutput := buf.String()
		assert.Contains(t, logOutput, "database error")

		buf.Reset()
	})
}

func TestOpenDB(t *testing.T) {
	cfg := config{}
	cfg.db.dsn = "postgres://user:pass@localhost/db"
	cfg.db.maxOpenConns = 10
	cfg.db.maxIdleConns = 5
	cfg.db.maxIdleTime = time.Minute

	t.Run("should establish database connection pool", func(t *testing.T) {
		// Setup sqlmock
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer db.Close()

		// Expect ping
		mock.ExpectPing()

		sqlOpen := func(driverName, dsn string) (*sql.DB, error) {
			assert.Equal(t, "postgres", driverName)
			assert.Equal(t, cfg.db.dsn, dsn)
			return db, nil
		}

		conn, err := openDB(cfg, sqlOpen)
		assert.NoError(t, err)
		assert.Equal(t, db, conn)
	})

	t.Run("should return sqlOpen error", func(t *testing.T) {
		expectedErr := errors.New("open failed")
		sqlOpen := func(driverName, dsn string) (*sql.DB, error) {
			return nil, expectedErr
		}

		conn, err := openDB(cfg, sqlOpen)
		assert.Nil(t, conn)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("should return ping error", func(t *testing.T) {
		// Setup sqlmock with ping monitoring enabled
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		assert.NoError(t, err)

		expectedPingErr := errors.New("ping failed")
		mock.ExpectPing().WillReturnError(expectedPingErr)

		sqlOpen := func(driverName, dsn string) (*sql.DB, error) {
			return db, nil
		}

		conn, err := openDB(cfg, sqlOpen)
		assert.Nil(t, conn)
		assert.Equal(t, expectedPingErr, err)
	})
}
