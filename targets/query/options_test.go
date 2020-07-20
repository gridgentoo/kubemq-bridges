package query

import (
	"github.com/kubemq-hub/kubemq-bridges/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOptions_parseOptions(t *testing.T) {

	tests := []struct {
		name     string
		cfg      config.Metadata
		wantOpts options
		wantErr  bool
	}{
		{
			name: "valid options",
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
			wantOpts: options{
				host:                  "localhost",
				port:                  50000,
				clientId:              "client_id",
				authToken:             "some-auth token",
				concurrency:           1,
				defaultChannel:        "some-channel",
				defaultTimeoutSeconds: 100,
			},
			wantErr: false,
		},
		{
			name: "invalid options - bad port",
			cfg: config.Metadata{
				Name: "kubemq-target",
				Kind: "",
				Properties: map[string]string{
					"host": "localhost",
					"port": "-1",
				},
			},
			wantOpts: options{},
			wantErr:  true,
		},
		{
			name: "invalid options - bad concurrency",
			cfg: config.Metadata{
				Name: "kubemq-target",
				Kind: "",
				Properties: map[string]string{
					"host":            "localhost",
					"port":            "50000",
					"client_id":       "client_id",
					"auth_token":      "some-auth token",
					"default_channel": "some-channel",
					"concurrency":     "-1",
				},
			},
			wantOpts: options{},
			wantErr:  true,
		},
		{
			name: "invalid options - bad default timeout seconds",
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
					"default_timeout_seconds": "-1",
				},
			},
			wantOpts: options{},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOpts, err := parseOptions(tt.cfg)
			if tt.wantErr {
				require.Error(t, err)

			} else {
				require.NoError(t, err)

			}
			require.EqualValues(t, tt.wantOpts, gotOpts)
		})
	}
}
