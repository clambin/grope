package grope

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"

	"github.com/go-openapi/strfmt"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/spf13/viper"
)

type exporter struct {
	logger    *slog.Logger
	client    *grafanaClient
	formatter formatter
	tags      string
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
		tags:   v.GetString("tags"),
		formatter: formatter{
			namespace:         v.GetString("namespace"),
			grafanaLabelName:  cmp.Or(v.GetString("grafana.operator.label.name"), "dashboards"),
			grafanaLabelValue: cmp.Or(v.GetString("grafana.operator.label.value"), "grafana"),
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

func (e exporter) exportDashboards(w io.Writer, args ...string) error {
	for dashboard, err := range yieldDashboards(e.client.dashboardClient, e.folders, args...) {
		if err != nil {
			return fmt.Errorf("error fetching dashboard: %w", err)
		}
		for _, tag := range strings.Split(e.tags, ",") {
			if tag = strings.TrimSpace(tag); tag != "" {
				if err = tagDashboard(dashboard, tag); err != nil {
					return fmt.Errorf("error tagging dashboard: %w", err)
				}
			}
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

func tagDashboard(db Dashboard, tag string) error {
	jsonModel, ok := db.Model.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected model type: %T; expected map[string]any", db.Model)
	}
	tagsAny, ok := jsonModel["tags"]
	if !ok {
		return errors.New("dashboard does not contain tags")
	}
	tags, ok := tagsAny.([]any)
	if !ok {
		return fmt.Errorf("unexpected tags type: %T; expected []any", tagsAny)
	}
	for _, t := range tags {
		if t.(string) == tag {
			return nil
		}
	}
	tags = append(tags, tag)
	jsonModel["tags"] = tags
	return nil
}
