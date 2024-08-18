package internal

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func Test_initViper(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantURL string
	}{
		{
			name: "pass",
			config: `
grafana:
  url: http://grafana.example.com
`,
			wantURL: "http://grafana.example.com",
		},
		{
			name: "invalid config file",
			config: `
not-a-valid-yaml-file
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "*.yaml")
			configFilename = tmpFile.Name()
			require.NoError(t, err)
			_, _ = tmpFile.Write([]byte(tt.config))
			_ = tmpFile.Close()
			defer func() { _ = os.Remove(tmpFile.Name()) }()

			v := viper.New()
			initViper(v)
			assert.Equal(t, v.ConfigFileUsed(), configFilename)
			assert.Equal(t, tt.wantURL, v.GetString("grafana.url"))
		})
	}
}
