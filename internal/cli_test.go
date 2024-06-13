package internal

import (
	"bytes"
	"errors"
	"flag"
	"github.com/gosimple/slug"
	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

func TestExportDashboards(t *testing.T) {
	tests := []struct {
		name   string
		config func() *viper.Viper
	}{
		{
			name: "unfiltered",
			config: func() *viper.Viper {
				v := viper.New()
				return v
			},
		},
		{
			name: "filtered",
			config: func() *viper.Viper {
				v := viper.New()
				v.Set("folders", "folder 1")
				return v
			},
		},
		{
			name: "override",
			config: func() *viper.Viper {
				v := viper.New()
				v.Set("namespace", "application")
				v.Set("grafana.operator.label.value", "local-grafana")
				v.Set("folders", "folder 1")
				return v
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp, err := makeExporter(tt.config(), slog.Default())
			require.NoError(t, err)
			exp.client = fakeClient{}

			var buf bytes.Buffer
			require.NoError(t, exp.exportDashboards(&buf))

			gp := filepath.Join("testdata", slug.Make(t.Name())+".yaml")
			if *update {
				require.NoError(t, os.WriteFile(gp, buf.Bytes(), 0644))
			}
			golden, err := os.ReadFile(gp)
			require.NoError(t, err)
			assert.Equal(t, string(golden), buf.String())
		})
	}
}

func TestExportDataSources(t *testing.T) {
	v := viper.New()
	v.Set("namespace", "monitoring")
	v.Set("grafana.operator.label.value", "local-grafana")

	exp, err := makeExporter(v, slog.Default())
	require.NoError(t, err)
	exp.client = fakeClient{}
	var buf bytes.Buffer
	require.NoError(t, exp.exportDataSources(&buf))

	gp := filepath.Join("testdata", slug.Make(t.Name())+".yaml")
	if *update {
		require.NoError(t, os.WriteFile(gp, buf.Bytes(), 0644))
	}
	golden, err := os.ReadFile(gp)
	require.NoError(t, err)
	assert.Equal(t, string(golden), buf.String())
}

var _ Fetcher = fakeClient{}

type fakeClient struct{}

func (f fakeClient) Dashboards() ([]gapi.FolderDashboardSearchResponse, error) {
	return []gapi.FolderDashboardSearchResponse{
		{UID: "1", Type: "dash-db", FolderTitle: "folder 1", Title: "db 1"},
		{UID: "2", Type: "dash-db", FolderTitle: "folder 2", Title: "db 2"},
	}, nil
}

func (f fakeClient) DashboardByUID(uid string) (*gapi.Dashboard, error) {
	switch uid {
	case "1", "2":
		return &gapi.Dashboard{Model: map[string]interface{}{"foo": "bar"}}, nil
	default:
		return nil, errors.New("dashboard not found")
	}
}

func (f fakeClient) DataSources() ([]*gapi.DataSource, error) {
	return []*gapi.DataSource{
		{
			Name: "Prometheus",
			URL:  "http://prometheus",
		},
	}, nil
}
