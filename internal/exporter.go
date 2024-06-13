package internal

import (
	"fmt"
	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/spf13/viper"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

type exporter struct {
	logger    *slog.Logger
	client    Fetcher
	formatter Formatter
	viper     *viper.Viper
}

type Fetcher interface {
	DashboardClient
	DataSourcesClient
}

func makeExporter(v *viper.Viper, l *slog.Logger) (exp exporter, err error) {
	exp.logger = l
	exp.viper = v

	exp.client, err = gapi.New(v.GetString("grafana.url"), gapi.Config{
		APIKey: v.GetString("grafana.token"),
		Client: http.DefaultClient,
	})
	if err != nil {
		return exp, fmt.Errorf("grafana connect: %w", err)
	}

	exp.formatter = OperatorFormatter{
		Namespace:         stringOrDefault(v.GetString("namespace"), "default"),
		GrafanaLabelName:  stringOrDefault(v.GetString("grafana.operator.label.name"), "dashboards"),
		GrafanaLabelValue: stringOrDefault(v.GetString("grafana.operator.label.value"), "grafana"),
	}
	return exp, nil
}

func stringOrDefault(s, defaultString string) string {
	if s != "" {
		return s
	}
	return defaultString
}

func (e exporter) exportDashboards(w io.Writer) error {
	//e.logger.Info("exporting dashboards")

	var folders []string
	if f := e.viper.GetString("folders"); f != "" {
		folders = strings.Split(f, ",")
	}

	dashboards, err := FetchDashboards(e.client, e.logger, folders...)
	if err != nil {
		return fmt.Errorf("fetch dashboards: %w", err)
	}

	//e.logger.Info("retrieved dashboards", "dashboards", len(dashboards))

	for _, dashboard := range dashboards {
		err = e.formatter.FormatDashboard(w, dashboard)

	}
	if err != nil {
		return fmt.Errorf("format dashboards: %w", err)
	}

	//e.logger.Info("dashboards formatted")
	return err
}

func (e exporter) exportDataSources(w io.Writer) error {
	sources, err := e.client.DataSources()
	if err != nil {
		return fmt.Errorf("grafana get datasources: %w", err)
	}

	if err = e.formatter.FormatDataSources(w, sources); err != nil {
		return fmt.Errorf("format datasources: %w", err)
	}
	return nil
}
