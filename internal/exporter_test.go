package internal

import (
	"bytes"
	"errors"
	"flag"
	"github.com/gosimple/slug"
	"github.com/grafana/grafana-openapi-client-go/client/dashboards"
	"github.com/grafana/grafana-openapi-client-go/client/datasources"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/grafana-openapi-client-go/models"
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
		args   []string
	}{
		{
			name:   "unfiltered",
			config: viper.New,
		},
		{
			name:   "filtered by name",
			config: viper.New,
			args:   []string{"db 1"},
		},
		{
			name: "filtered by folder",
			config: func() *viper.Viper {
				v := viper.New()
				v.Set("folders", true)
				return v
			},
			args: []string{"folder 1"},
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
			exp.client.dashboardClient.searcher = fakeSearcher{
				hitList: models.HitList{
					{Title: "db 1", FolderTitle: "folder 1", Type: "dash-db", UID: "1"},
					{Title: "db 2", FolderTitle: "folder 2", Type: "dash-db", UID: "2"},
				},
			}
			exp.client.dashboardClient.dashboardFetcher = fakeDashboardFetcher{
				dashboards: map[string]any{
					"1": map[string]string{"foo": "bar"},
					"2": map[string]string{"foo": "bar"},
				},
			}

			var buf bytes.Buffer
			require.NoError(t, exp.exportDashboards(&buf, tt.args...))

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
	exp.client.dataSourcesClient.dataSourceFetcher = fakeDataSourceFetcher{
		dataSources: models.DataSourceList{
			{ID: 0, Name: "prometheus", Type: "prometheus", URL: "http://prometheus"},
		},
	}
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

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ searcher = fakeSearcher{}

type fakeSearcher struct {
	hitList models.HitList
	err     error
}

func (f fakeSearcher) Search(_ *search.SearchParams, _ ...search.ClientOption) (*search.SearchOK, error) {
	var result *search.SearchOK
	if f.err == nil {
		result = search.NewSearchOK()
		result.Payload = f.hitList
	}
	return result, f.err
}

var _ dashboardFetcher = fakeDashboardFetcher{}

type fakeDashboardFetcher struct {
	err        error
	dashboards map[string]any
}

func (f fakeDashboardFetcher) GetDashboardByUID(dashboardUID string, _ ...dashboards.ClientOption) (*dashboards.GetDashboardByUIDOK, error) {
	if f.err != nil {
		return nil, f.err
	}
	db, ok := f.dashboards[dashboardUID]
	if !ok {
		return nil, errors.New("dashboard not found")
	}
	result := dashboards.NewGetDashboardByUIDOK()
	result.Payload = &models.DashboardFullWithMeta{
		Dashboard: db,
	}
	return result, nil
}

var _ dataSourceFetcher = &fakeDataSourceFetcher{}

type fakeDataSourceFetcher struct {
	err         error
	dataSources models.DataSourceList
}

func (f fakeDataSourceFetcher) GetDataSources(_ ...datasources.ClientOption) (*datasources.GetDataSourcesOK, error) {
	if f.err != nil {
		return nil, f.err
	}
	ok := datasources.NewGetDataSourcesOK()
	ok.Payload = f.dataSources
	return ok, nil
}
