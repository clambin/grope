package internal

import (
	"fmt"
	"github.com/go-openapi/strfmt"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/spf13/viper"
	"io"
	"log/slog"
	"net/url"
)

type exporter struct {
	logger    *slog.Logger
	client    *grafanaClient
	formatter formatter
	folders   bool
}

type grafanaClient struct {
	dashboardClient
	dataSourcesClient
}

func makeExporter(v *viper.Viper, l *slog.Logger) (*exporter, error) {
	client, err := newGrafanaClient(v.GetString("grafana.url"), v.GetString("grafana.token"))
	if err != nil {
		return nil, err
	}
	return &exporter{
		logger: l,
		client: client,
		formatter: formatter{
			namespace:         stringOrDefault(v.GetString("namespace"), "default"),
			grafanaLabelName:  stringOrDefault(v.GetString("grafana.operator.label.name"), "dashboards"),
			grafanaLabelValue: stringOrDefault(v.GetString("grafana.operator.label.value"), "grafana"),
		},
		folders: v.GetBool("folders"),
	}, nil
}

func newGrafanaClient(grafanaURL, apiKey string) (*grafanaClient, error) {
	target, err := url.Parse(grafanaURL)
	if err != nil {
		return nil, fmt.Errorf("grafana.url invalid: %w", err)
	}
	if target.Scheme == "" {
		return nil, fmt.Errorf("grafana.url scheme invalid: %s", grafanaURL)
	}
	cfg := goapi.TransportConfig{
		Host:     target.Host,
		BasePath: "/api",
		Schemes:  []string{target.Scheme},
		APIKey:   apiKey,
	}
	c := goapi.NewHTTPClientWithConfig(strfmt.Default, &cfg)
	return &grafanaClient{
		dashboardClient: dashboardClient{
			searcher:         c.Search,
			dashboardFetcher: c.Dashboards,
		},
		dataSourcesClient: dataSourcesClient{
			dataSourceFetcher: c.Datasources,
		},
	}, nil
}

func stringOrDefault(s, defaultString string) string {
	if s != "" {
		return s
	}
	return defaultString
}

func (e exporter) exportDashboards(w io.Writer, args ...string) error {
	for dashboard, err := range yieldDashboards(e.client.dashboardClient, e.folders, args...) {
		if err != nil {
			return fmt.Errorf("error fetching dashboard: %w", err)
		}
		if err = e.formatter.formatDashboard(w, dashboard); err != nil {
			return fmt.Errorf("error formating dashboard %q: %w", dashboard.Title, err)
		}
	}
	return nil
}

func (e exporter) exportDataSources(w io.Writer) error {
	sources, err := getDataSources(e.client.dataSourcesClient)
	if err == nil {
		err = e.formatter.formatDataSources(w, sources)
	}
	return err
}
