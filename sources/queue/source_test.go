package queue

import (
	"context"
	"fmt"
	"github.com/kubemq-hub/kubemq-bridges/config"
	"github.com/kubemq-hub/kubemq-bridges/middleware"
	"github.com/kubemq-hub/kubemq-bridges/pkg/logger"
	"github.com/kubemq-hub/kubemq-bridges/pkg/uuid"
	"github.com/kubemq-io/kubemq-go"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type mockTarget struct {
	setResponse interface{}
	setError    error
	delay       time.Duration
}

func (m *mockTarget) Do(ctx context.Context, request interface{}) (interface{}, error) {
	time.Sleep(m.delay)
	return m.setResponse, m.setError
}
func setupSource(ctx context.Context, targets []middleware.Middleware, ch string, maxRequeue string) (*Source, error) {
	s := New()

	err := s.Init(ctx, config.Metadata{
		"address":      "0.0.0.0:50000",
		"client_id":    "some-client-id",
		"auth_token":   "",
		"channel":      ch,
		"batch_size":   "1",
		"wait_timeout": "60",
		"sources":      "2",
		"max_requeue":  maxRequeue,
	}, config.Metadata{})
	if err != nil {
		return nil, err
	}
	err = s.Start(ctx, targets, logger.NewLogger("source"))
	if err != nil {
		return nil, err
	}
	time.Sleep(time.Second)
	return s, nil
}
func sendQueueMessage(t *testing.T, ctx context.Context, req *kubemq.QueueMessage, sendChannel string) error {
	client, err := kubemq.NewClient(ctx,
		kubemq.WithAddress("localhost", 50000),
		kubemq.WithClientId(uuid.New().String()),
		kubemq.WithTransportType(kubemq.TransportTypeGRPC))

	if err != nil {
		return err
	}

	_, err = client.SetQueueMessage(req).SetChannel(sendChannel).Send(ctx)

	return err
}

func TestClient_processQueue(t *testing.T) {
	tests := []struct {
		name        string
		target      middleware.Middleware
		respChannel string
		req         *kubemq.QueueMessage
		sendCh      string
		maxRequeue  string
		wantErr     bool
	}{
		{
			name: "request",
			target: &mockTarget{
				setResponse: nil,
				setError:    nil,
				delay:       0,
			},
			req:        kubemq.NewQueueMessage().SetBody([]byte("some-data")),
			sendCh:     uuid.New().String(),
			maxRequeue: "0",
			wantErr:    false,
		},
		{
			name: "request with target error",
			target: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
			},
			req:        kubemq.NewQueueMessage().SetBody([]byte("some-data")),
			wantErr:    false,
			sendCh:     uuid.New().String(),
			maxRequeue: "0",
		},
		{
			name: "request with target error and requeue",
			target: &mockTarget{
				setResponse: nil,
				setError:    fmt.Errorf("some-error"),
				delay:       0,
			},
			req:        kubemq.NewQueueMessage().SetBody([]byte("some-data")),
			wantErr:    false,
			sendCh:     uuid.New().String(),
			maxRequeue: "1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			c, err := setupSource(ctx, []middleware.Middleware{tt.target}, tt.sendCh, tt.maxRequeue)
			require.NoError(t, err)
			defer func() {
				_ = c.Stop()
			}()
			err = sendQueueMessage(t, ctx, tt.req, tt.sendCh)
			if tt.wantErr {
				require.Error(t, err)

			} else {
				require.NoError(t, err)
			}
			time.Sleep(time.Second)
		})
	}
}

func TestClient_Init(t *testing.T) {

	tests := []struct {
		name       string
		connection config.Metadata
		wantErr    bool
	}{
		{
			name: "init",
			connection: config.Metadata{
				"address":      "localhost:50000",
				"client_id":    "",
				"auth_token":   "some-auth token",
				"channel":      "some-channel",
				"batch_size":   "1",
				"wait_timeout": "60",
				"sources":      "2",
				"max_requeue":  "1",
			},
			wantErr: false,
		},
		{
			name: "init - error",
			connection: config.Metadata{
				"address": "localhost",
			},
			wantErr: true,
		},
		{
			name: "init - bad channel",
			connection: config.Metadata{
				"address":      "localhost:50000",
				"client_id":    "",
				"auth_token":   "some-auth token",
				"channel":      "",
				"batch_size":   "1",
				"wait_timeout": "60",
				"sources":      "2",
				"max_requeue":  "1",
			},
			wantErr: true,
		},
		{
			name: "init - bad batch size",
			connection: config.Metadata{
				"address":      "localhost:50000",
				"client_id":    "",
				"auth_token":   "some-auth token",
				"channel":      "some-channel",
				"batch_size":   "-1",
				"wait_timeout": "60",
				"sources":      "2",
				"max_requeue":  "1",
			},
			wantErr: true,
		}, {
			name: "init - bad wait timeout",
			connection: config.Metadata{
				"address":      "localhost:50000",
				"client_id":    "",
				"auth_token":   "some-auth token",
				"channel":      "some-channel",
				"batch_size":   "1",
				"wait_timeout": "-1",
				"sources":      "2",
				"max_requeue":  "1",
			},
			wantErr: true,
		},
		{
			name: "init - bad sources",
			connection: config.Metadata{
				"address":      "localhost:50000",
				"client_id":    "",
				"auth_token":   "some-auth token",
				"channel":      "some-channel",
				"batch_size":   "1",
				"wait_timeout": "1",
				"sources":      "-1",
				"max_requeue":  "1",
			},
			wantErr: true,
		},
		{
			name: "init - bad max requeue",
			connection: config.Metadata{
				"address":      "localhost:50000",
				"client_id":    "",
				"auth_token":   "some-auth token",
				"channel":      "some-channel",
				"batch_size":   "1",
				"wait_timeout": "1",
				"sources":      "1",
				"max_requeue":  "-1",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			c := New()
			if err := c.Init(ctx, tt.connection, config.Metadata{}); (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
			}

		})
	}
}
