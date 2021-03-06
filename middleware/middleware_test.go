package middleware

import (
	"context"
	"fmt"
	"github.com/kubemq-io/kubemq-bridges/config"
	"github.com/kubemq-io/kubemq-bridges/pkg/logger"
	"github.com/kubemq-io/kubemq-bridges/pkg/metrics"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
	"time"
)

type mockTarget struct {
	setResponse interface{}
	setError    error
	delay       time.Duration
	executed    int
}

func (m *mockTarget) Do(ctx context.Context, request interface{}) (interface{}, error) {
	time.Sleep(m.delay)
	return m.setResponse, m.setError
}

func TestClient_RateLimiter(t *testing.T) {
	tests := []struct {
		name             string
		mock             *mockTarget
		meta             config.Metadata
		timeToRun        time.Duration
		wantMaxExecution int
		wantErr          bool
	}{
		{
			name: "100 per seconds",
			mock: &mockTarget{
				setResponse: nil,
				setError:    nil,
				delay:       0,
				executed:    0,
			},
			meta: map[string]string{
				"rate_per_second": "100",
			},
			timeToRun:        time.Second,
			wantMaxExecution: 110,
			wantErr:          false,
		},
		{
			name: "unlimited",
			mock: &mockTarget{
				setResponse: nil,
				setError:    nil,
				delay:       0,
				executed:    0,
			},
			meta: map[string]string{},

			timeToRun:        time.Second,
			wantMaxExecution: math.MaxInt32,
			wantErr:          false,
		},
		{
			name: "bad rpc",
			mock: &mockTarget{
				setResponse: nil,
				setError:    nil,
				delay:       0,
				executed:    0,
			},
			meta: map[string]string{
				"rate_per_second": "-100",
			},
			timeToRun:        time.Second,
			wantMaxExecution: 0,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeToRun)
			defer cancel()
			rl, err := NewRateLimitMiddleware(tt.meta)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			md := Chain(tt.mock, RateLimiter(rl))
			for {
				select {
				case <-ctx.Done():
					goto done
				default:

				}
				_, _ = md.Do(ctx, "request")
			}
		done:
			require.LessOrEqual(t, tt.mock.executed, tt.wantMaxExecution)

		})
	}
}

func TestClient_Retry(t *testing.T) {
	log := logger.NewLogger("TestClient_Retry")
	tests := []struct {
		name        string
		mock        *mockTarget
		meta        config.Metadata
		wantRetries int
		wantErr     bool
	}{
		{
			name: "no retry options",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    1,
			},
			meta:        map[string]string{},
			wantRetries: 1,
			wantErr:     false,
		},
		{
			name: "retry with success",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    1,
			},
			meta:        map[string]string{},
			wantRetries: 1,
			wantErr:     false,
		},
		{
			name: "3 retries fixed delay",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    3,
			},
			meta: map[string]string{
				"retry_attempts":           "3",
				"retry_delay_milliseconds": "100",
				"retry_delay_type":         "fixed",
			},

			wantRetries: 3,
			wantErr:     false,
		},
		{
			name: "3 retries back-off delay",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    3,
			},
			meta: map[string]string{
				"retry_attempts":           "3",
				"retry_delay_milliseconds": "100",
				"retry_delay_type":         "back-off",
			},
			wantRetries: 3,
			wantErr:     false,
		},
		{
			name: "3 retries random delay",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    3,
			},
			meta: map[string]string{
				"retry_attempts":           "3",
				"retry_delay_milliseconds": "200",
				"retry_delay_type":         "random",
			},
			wantRetries: 3,
			wantErr:     false,
		},
		{
			name: "3 retries random with jitter delay",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    3,
			},
			meta: map[string]string{
				"retry_attempts":                "3",
				"retry_delay_milliseconds":      "200",
				"retry_max_jitter_milliseconds": "200",
				"retry_delay_type":              "random",
			},
			wantRetries: 3,
			wantErr:     false,
		},
		{
			name: "bad rate limiter - attempts",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    3,
			},
			meta: map[string]string{
				"retry_attempts":                "-3",
				"retry_delay_milliseconds":      "200",
				"retry_max_jitter_milliseconds": "200",
				"retry_delay_type":              "random",
			},
			wantRetries: 3,
			wantErr:     true,
		},
		{
			name: "bad rate limiter - retry_delay_milliseconds",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    3,
			},
			meta: map[string]string{
				"retry_attempts":                "3",
				"retry_delay_milliseconds":      "-200",
				"retry_max_jitter_milliseconds": "200",
				"retry_delay_type":              "random",
			},
			wantRetries: 3,
			wantErr:     true,
		},
		{
			name: "bad rate limiter - retry_max_jitter_milliseconds",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    3,
			},
			meta: map[string]string{
				"retry_attempts":                "3",
				"retry_delay_milliseconds":      "200",
				"retry_max_jitter_milliseconds": "-200",
				"retry_delay_type":              "random",
			},
			wantRetries: 3,
			wantErr:     true,
		},
		{
			name: "bad rate limiter - retry_delay_type",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    3,
			},
			meta: map[string]string{
				"retry_attempts":                "3",
				"retry_delay_milliseconds":      "200",
				"retry_max_jitter_milliseconds": "200",
				"retry_delay_type":              "bad-type",
			},
			wantRetries: 3,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			r, err := NewRetryMiddleware(tt.meta, log)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			md := Chain(tt.mock, Retry(r))
			resp, err := md.Do(ctx, "request")
			if tt.mock.setError != nil {
				require.Error(t, err)
				require.EqualValues(t, tt.wantRetries, tt.mock.executed)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}

		})
	}
}

func TestClient_Metric(t *testing.T) {
	exporter, err := metrics.NewExporter()
	require.NoError(t, err)

	tests := []struct {
		name       string
		mock       *mockTarget
		cfg        config.BindingConfig
		wantReport *metrics.Report
		wantErr    bool
	}{
		{
			name: "no error request",
			mock: &mockTarget{
				setResponse: "response",
				setError:    nil,
				delay:       0,
				executed:    0,
			},
			cfg: config.BindingConfig{

				Name: "b-1",
				Sources: config.Spec{
					Kind:        "sk",
					Connections: nil,
				},
				Targets: config.Spec{
					Kind:        "tk",
					Connections: nil,
				},
				Properties: map[string]string{},
			},
			wantReport: &metrics.Report{
				Key:            "b-1-sn-sk-tn-tk",
				Binding:        "b-1",
				SourceKind:     "sk",
				TargetKind:     "tk",
				RequestCount:   1,
				RequestVolume:  16,
				ResponseCount:  1,
				ResponseVolume: 16,
				ErrorsCount:    0,
			},
			wantErr: false,
		},
		{
			name: "error request",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    0,
			},
			cfg: config.BindingConfig{
				Name: "b-2",
				Sources: config.Spec{
					Kind:        "sk",
					Connections: nil,
				},
				Targets: config.Spec{
					Kind:        "tk",
					Connections: nil,
				},
				Properties: map[string]string{},
			},
			wantReport: &metrics.Report{
				Key:            "b-2-sn-sk-tn-tk",
				Binding:        "b-2",
				SourceKind:     "sk",
				TargetKind:     "tk",
				RequestCount:   1,
				RequestVolume:  16,
				ResponseCount:  0,
				ResponseVolume: 0,
				ErrorsCount:    1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			m, err := NewMetricsMiddleware(tt.cfg, exporter)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			md := Chain(tt.mock, Metric(m))
			_, _ = md.Do(ctx, "data")
			storedReport := exporter.Store.Get(tt.wantReport.Key)
			require.EqualValues(t, tt.wantReport, storedReport)
		})
	}
}

func TestClient_Log(t *testing.T) {

	tests := []struct {
		name    string
		mock    *mockTarget
		meta    config.Metadata
		wantErr bool
	}{
		{
			name: "no log level",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    0,
			},
			meta:    map[string]string{},
			wantErr: false,
		},
		{
			name: "debug level",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    0,
			},
			meta: map[string]string{
				"log_level": "debug",
			},
			wantErr: false,
		},
		{
			name: "info level",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    0,
			},
			meta: map[string]string{
				"log_level": "info",
			},
			wantErr: false,
		},
		{
			name: "info level - 2",
			mock: &mockTarget{
				setResponse: nil,
				setError:    nil,
				delay:       0,
				executed:    0,
			},
			meta: map[string]string{
				"log_level": "info",
			},
			wantErr: false,
		},
		{
			name: "error level",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    0,
			},
			meta: map[string]string{
				"log_level": "error",
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			mock: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
				executed:    0,
			},
			meta: map[string]string{
				"log_level": "bad-level",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			log, err := NewLogMiddleware("test", tt.meta)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			md := Chain(tt.mock, Log(log))
			_, _ = md.Do(ctx, "request")
		})
	}
}

func TestClient_Chain(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	mock := &mockTarget{
		setResponse: nil,
		setError:    fmt.Errorf("some-error"),
		delay:       0,
		executed:    0,
	}
	meta := map[string]string{
		"log_level":                     "debug",
		"rate_per_second":               "1",
		"retry_attempts":                "3",
		"retry_delay_milliseconds":      "100",
		"retry_max_jitter_milliseconds": "100",
		"retry_delay_type":              "fixed",
	}
	log, err := NewLogMiddleware("test", meta)
	require.NoError(t, err)
	rl, err := NewRateLimitMiddleware(meta)
	require.NoError(t, err)
	retry, err := NewRetryMiddleware(meta, logger.NewLogger("retry-logger"))
	require.NoError(t, err)
	md := Chain(mock, RateLimiter(rl), Retry(retry), Log(log))
	start := time.Now()
	_, _ = md.Do(ctx, "request")
	d := time.Since(start)
	require.GreaterOrEqual(t, d.Milliseconds(), 2*time.Second.Milliseconds())
}
