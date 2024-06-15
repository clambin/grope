package internal

import (
	"fmt"
	"github.com/clambin/go-common/set"
	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/spf13/viper"
	"io"
	"log/slog"
	"net/http"
)

type exporter struct {
	logger    *slog.Logger
	client    Fetcher
	formatter Formatter
	folders   bool
}

type Fetcher interface {
	DashboardClient
	DataSourcesClient
}

func makeExporter(v *viper.Viper, l *slog.Logger) (exporter, error) {
	c, err := gapi.New(v.GetString("grafana.url"), gapi.Config{
		APIKey: v.GetString("grafana.token"),
		Client: http.DefaultClient,
	})
	if err != nil {
		return exporter{}, fmt.Errorf("grafana connect: %w", err)
	}
	return exporter{
		logger: l,
		client: c,
		formatter: OperatorFormatter{
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
	dashboards, err := FetchDashboards(e.client, e.logger, e.shouldExport(args...))
	if err == nil {
		for _, dashboard := range dashboards {
			if err = e.formatter.FormatDashboard(w, dashboard); err != nil {
				return fmt.Errorf("format dashboard %q: %w", dashboard.Title, err)
			}
		}
	}
	return err
}

func (e exporter) shouldExport(args ...string) func(gapi.FolderDashboardSearchResponse) bool {
	validNames := set.New(args...)
	return func(dashboard gapi.FolderDashboardSearchResponse) bool {
		if len(args) == 0 {
			return true
		}
		if e.folders {
			return validNames.Contains(dashboard.FolderTitle)
		}
		return validNames.Contains(dashboard.Title)
	}
}

func (e exporter) exportDataSources(w io.Writer) error {
	sources, err := e.client.DataSources()
	if err == nil {
		err = e.formatter.FormatDataSources(w, sources)
	}
	return err
}
