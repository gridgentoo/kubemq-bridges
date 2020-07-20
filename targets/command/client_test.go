package command

import (
	"context"
	"fmt"
	"github.com/kubemq-hub/kubemq-bridges/config"
	"github.com/kubemq-hub/kubemq-bridges/types"
	"github.com/kubemq-io/kubemq-go"

	"github.com/stretchr/testify/require"
	"testing"

	"time"
)

type mockCommandReceiver struct {
	host           string
	port           int
	channel        string
	executionDelay time.Duration
	executionError error
	executionTime  int64
}

func (m *mockCommandReceiver) run(ctx context.Context, t *testing.T) error {
	client, err := kubemq.NewClient(ctx,
		kubemq.WithAddress(m.host, m.port),
		kubemq.WithClientId("response-id"),
		kubemq.WithTransportType(kubemq.TransportTypeGRPC),
		kubemq.WithCheckConnection(true))
	if err != nil {
		return err
	}
	errCh := make(chan error, 1)
	commandCh, err := client.SubscribeToCommands(ctx, m.channel, "", errCh)
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case cmd := <-commandCh:
				time.Sleep(m.executionDelay)
				cmdResponse := client.R().SetRequestId(cmd.Id).SetResponseTo(cmd.ResponseTo).SetExecutedAt(time.Unix(m.executionTime, 0))
				if m.executionError != nil {
					cmdResponse.SetError(m.executionError)
				}
				err := cmdResponse.Send(ctx)
				require.NoError(t, err)
			case err := <-errCh:
				require.NoError(t, err)
			case <-ctx.Done():
				return
			}

		}
	}()
	time.Sleep(time.Second)
	return nil
}

func TestClient_Do(t *testing.T) {
	tests := []struct {
		name         string
		cfg          config.Metadata
		mockReceiver *mockCommandReceiver
		req          *types.Request
		wantResp     *types.Response
		wantErr      bool
	}{
		{
			name: "request",
			cfg: config.Metadata{
				Name: "kubemq-target",
				Kind: "",
				Properties: map[string]string{
					"host": "localhost",
					"port": "50000",
				},
			},
			mockReceiver: &mockCommandReceiver{
				host:           "localhost",
				port:           50000,
				channel:        "commands",
				executionDelay: 0,
				executionError: nil,
				executionTime:  1000,
			},
			req: types.NewRequest().
				SetData([]byte("data")).
				SetMetadataKeyValue("id", "id").
				SetMetadataKeyValue("channel", "commands").
				SetMetadataKeyValue("timeout_seconds", "5"),
			wantResp: types.NewResponse().
				SetMetadataKeyValue("result", "ok").
				SetMetadataKeyValue("id", "id"),
			wantErr: false,
		},
		{
			name: "request with execution error",
			cfg: config.Metadata{
				Name: "kubemq-target",
				Kind: "",
				Properties: map[string]string{
					"host": "localhost",
					"port": "50000",
				},
			},
			mockReceiver: &mockCommandReceiver{
				host:           "localhost",
				port:           50000,
				channel:        "commands",
				executionDelay: 0,
				executionError: fmt.Errorf("error"),
				executionTime:  0,
			},
			req: types.NewRequest().
				SetData([]byte("data")).
				SetMetadataKeyValue("id", "id").
				SetMetadataKeyValue("channel", "commands").
				SetMetadataKeyValue("timeout_seconds", "5"),
			wantResp: nil,
			wantErr:  true,
		},
		{
			name: "request error - empty body",
			cfg: config.Metadata{
				Name: "kubemq-target",
				Kind: "",
				Properties: map[string]string{
					"host": "localhost",
					"port": "50000",
				},
			},
			mockReceiver: &mockCommandReceiver{
				host:           "localhost",
				port:           50000,
				channel:        "commands",
				executionDelay: 0,
				executionError: nil,
				executionTime:  0,
			},
			req: types.NewRequest().
				SetMetadataKeyValue("id", "id").
				SetMetadataKeyValue("channel", "commands").
				SetMetadataKeyValue("timeout_seconds", "5"),
			wantResp: nil,
			wantErr:  true,
		},
		{
			name: "request error - bad metadata - no channel",
			cfg: config.Metadata{
				Name: "kubemq-target",
				Kind: "",
				Properties: map[string]string{
					"host": "localhost",
					"port": "50000",
				},
			},
			mockReceiver: &mockCommandReceiver{
				host:           "localhost",
				port:           50000,
				channel:        "commands",
				executionDelay: 0,
				executionError: nil,
				executionTime:  0,
			},
			req: types.NewRequest().
				SetMetadataKeyValue("id", "id").
				SetMetadataKeyValue("channel", "").
				SetMetadataKeyValue("timeout_seconds", "-1"),
			wantResp: nil,
			wantErr:  true,
		},
		{
			name: "request error - bad metadata - invalid timeout seconds",
			cfg: config.Metadata{
				Name: "kubemq-target",
				Kind: "",
				Properties: map[string]string{
					"host": "localhost",
					"port": "50000",
				},
			},
			mockReceiver: &mockCommandReceiver{
				host:           "localhost",
				port:           50000,
				channel:        "commands",
				executionDelay: 0,
				executionError: nil,
				executionTime:  0,
			},
			req: types.NewRequest().
				SetMetadataKeyValue("id", "id").
				SetMetadataKeyValue("channel", "commands").
				SetMetadataKeyValue("timeout_seconds", "-1"),
			wantResp: nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := tt.mockReceiver.run(ctx, t)
			require.NoError(t, err)
			target := New()
			err = target.Init(ctx, tt.cfg)
			require.NoError(t, err)
			gotResp, err := target.Do(ctx, tt.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.EqualValues(t, tt.wantResp, gotResp)
		})
	}
}

func TestClient_Init(t *testing.T) {

	tests := []struct {
		name    string
		cfg     config.Metadata
		wantErr bool
	}{
		{
			name: "init",
			cfg: config.Metadata{
				Name: "kubemq-target",
				Kind: "",
				Properties: map[string]string{
					"host":                    "localhost",
					"port":                    "50000",
					"client_id":               "client_id",
					"auth_token":              "some-auth token",
					"default_channel":         "some-channel",
					"concurrency":             "1",
					"default_timeout_seconds": "100",
				},
			},
			wantErr: false,
		},
		{
			name: "init - error",
			cfg: config.Metadata{
				Name: "kubemq-target",
				Kind: "",
				Properties: map[string]string{
					"host": "localhost",
					"port": "-1",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			c := New()

			if err := c.Init(ctx, tt.cfg); (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.EqualValues(t, tt.cfg.Name, c.Name())
		})
	}
}
