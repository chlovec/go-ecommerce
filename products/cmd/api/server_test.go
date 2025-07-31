package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockHTTPServer struct {
	mock.Mock
}

func (m *MockHTTPServer) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	err := args.Error(0)
	return err
}

func (m *MockHTTPServer) ListenAndServe() error {
	args := m.Called()
	return args.Error(0)
}

type safeBuffer struct {
	mu sync.Mutex
	b  *bytes.Buffer
}

func (s *safeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestServe(t *testing.T) {
	t.Run("should start and shut down gracefully", func(t *testing.T) {
		sb := &safeBuffer{b: &bytes.Buffer{}}
		logger := newLogger(sb)
		cfg := config{port: 8080, env: "test"}

		svr := newServer(cfg, logger)

		// Simulate manual shutdown
		go func() {
			time.Sleep(200 * time.Millisecond)
			process, _ := os.FindProcess(os.Getpid())
			err := process.Signal(os.Interrupt)
			assert.NoError(t, err)
		}()

		err := svr.Serve()
		assert.NoError(t, err)

		logOutput := sb.String()
		assert.Contains(t, logOutput, `"starting server" addr=:8080 env=test`)
		assert.Contains(t, logOutput, `"shutting down server" signal=interrupt`)
		assert.Contains(t, logOutput, `"stopped server" addr=:8080`)
	})

	t.Run("should start and not shutdown gracefully", func(t *testing.T) {
		sb := &safeBuffer{b: &bytes.Buffer{}}
		logger := newLogger(sb)
		cfg := config{port: 8080, env: "test"}
		mockSrv := new(MockHTTPServer)

		mockSrv.On("Shutdown", mock.Anything).Return(errors.New("server shutdown error"))
		mockSrv.On("ListenAndServe").Return(http.ErrServerClosed)

		svr := &Server{
			addr:       ":8080",
			config:     cfg,
			logger:     logger,
			httpServer: mockSrv,
		}

		go func() {
			time.Sleep(100 * time.Millisecond)
			process, _ := os.FindProcess(os.Getpid())
			err := process.Signal(os.Interrupt)
			assert.NoError(t, err)
		}()

		err := svr.Serve()
		assert.Error(t, err)
		assert.Equal(t, "server shutdown error", err.Error())

		logOutput := sb.String()
		assert.Contains(t, logOutput, `"starting server" addr=:8080 env=test`)
		assert.Contains(t, logOutput, `"shutting down server" signal=interrupt`)
		assert.NotContains(t, logOutput, `"stopped server" addr=:8080`)
	})

	t.Run("should fail with listen and serve error", func(t *testing.T) {
		sb := &safeBuffer{b: &bytes.Buffer{}}
		logger := newLogger(sb)
		cfg := config{port: 8080, env: "test"}
		mockSrv := new(MockHTTPServer)

		mockSrv.On("Shutdown", mock.Anything).Return(nil)
		mockSrv.On("ListenAndServe").Return(errors.New("unexpected listen and serve error"))

		svr := &Server{
			addr:       ":8080",
			config:     cfg,
			logger:     logger,
			httpServer: mockSrv,
		}

		go func() {
			time.Sleep(100 * time.Millisecond)
			process, _ := os.FindProcess(os.Getpid())
			err := process.Signal(os.Interrupt)
			assert.NoError(t, err)
		}()

		err := svr.Serve()
		assert.Error(t, err)
		assert.Equal(t, "unexpected listen and serve error", err.Error())

		logOutput := sb.String()
		assert.Contains(t, logOutput, `"starting server" addr=:8080 env=test`)
		assert.NotContains(t, logOutput, `"shutting down server" signal=interrupt`)
		assert.NotContains(t, logOutput, `"stopped server" addr=:8080`)
	})
}
