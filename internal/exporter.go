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
	client    grafanaClient
	formatter Formatter
	folders   bool
}

type grafanaClient struct {
	dashboardClient
	dataSourcesClient
}

func makeExporter(v *viper.Viper, l *slog.Logger) (exporter, error) {
	target, err := url.Parse(v.GetString("grafana.url"))
	if err != nil {
		return exporter{}, fmt.Errorf("grafana.url invalid: %w", err)
	}
	cfg := goapi.TransportConfig{
		Host:     target.Host,
		BasePath: "/api",
		Schemes:  []string{target.Scheme},
		APIKey:   v.GetString("grafana.token"),
	}
	c := goapi.NewHTTPClientWithConfig(strfmt.Default, &cfg)
	return exporter{
		logger: l,
		client: grafanaClient{
			dashboardClient: dashboardClient{
				searcher:         c.Search,
				dashboardFetcher: c.Dashboards,
			},
			dataSourcesClient: dataSourcesClient{
				dataSourceFetcher: c.Datasources,
			},
		},
		formatter: Formatter{
			Namespace:         stringOrDefault(v.GetString("namespace"), "default"),
			GrafanaLabelName:  stringOrDefault(v.GetString("grafana.operator.label.name"), "dashboards"),
			GrafanaLabelValue: stringOrDefault(v.GetString("grafana.operator.label.value"), "grafana"),
		},
		folders: v.GetBool("folders"),
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
		if err = e.formatter.FormatDashboard(w, dashboard); err != nil {
			return fmt.Errorf("error formating dashboard %q: %w", dashboard.Title, err)
		}
	}
	return nil
}

func (e exporter) exportDataSources(w io.Writer) error {
	sources, err := getDataSources(e.client.dataSourcesClient)
	if err == nil {
		err = e.formatter.FormatDataSources(w, sources)
	}
	return err
}
