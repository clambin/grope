package main

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"os"
	"time"

	"codeberg.org/clambin/go-common/charmer"
	"github.com/gosimple/slug"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/grafana-operator/v5/api/v1beta1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var (
	dataSourcesCmd = &cobra.Command{
		Use:   "datasources <name> [ <name> ...]",
		Short: "export Grafana data sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := configurationFromViper(viper.GetViper())
			client, err := cfg.grafanaClient()
			if err != nil {
				return fmt.Errorf("grafana: %w", err)
			}
			return exportDatasources(os.Stdout, client, cfg, args, charmer.GetLogger(cmd))
		},
	}
)

func init() {
	rootCmd.AddCommand(dataSourcesCmd)
}

func exportDatasources(
	w io.Writer,
	client *grafanaClient,
	cfg configuration,
	args []string,
	logger *slog.Logger,
) error {

	for datasource := range grafanaDataSources(client, args, logger) {
		if len(datasource.SecureJSONFields) > 0 {
			logger.Warn("datasource uses secure JSON fields and requires manual changes. See https://grafana.github.io/grafana-operator/docs/datasources/", "datasource", datasource.Name)
		}
		body, err := yaml.Marshal(operatorDatasource(cfg, datasource))
		if err != nil {
			logger.Error("failed to marshal operator datasource", "err", err)
			return err
		}
		_, _ = w.Write([]byte("---\n"))
		_, _ = w.Write(body)

	}
	return nil
}

// grafanaDataSources returns all datasources that match the names in args.
func grafanaDataSources(c *grafanaClient, args []string, logger *slog.Logger) iter.Seq[*models.DataSource] {
	return func(yield func(*models.DataSource) bool) {
		for _, name := range args {
			ds, err := c.Datasources.GetDataSourceByName(name)
			if err != nil {
				logger.Error("Error getting datasources", "name", name, "err", err)
				continue
			}
			if !yield(ds.GetPayload()) {
				return
			}
		}
	}
}

// datasourceManifest is a stripped-down version of Grafana Operator Datasource custom resource.
// This allows us to marshal the datasource to YAML without including the Status section.
type datasourceManifest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              v1beta1.GrafanaDatasourceSpec `json:"spec,omitempty"`
}

func operatorDatasource(cfg configuration, datasource *models.DataSource) datasourceManifest {
	var jsonData json.RawMessage
	if datasource.JSONData != nil {
		jsonData, _ = json.Marshal(datasource.JSONData)
	}
	return datasourceManifest{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1beta1.GroupVersion.String(),
			Kind:       "GrafanaDatasource",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      slug.Make(datasource.Name),
			Namespace: cfg.Namespace,
		},
		Spec: v1beta1.GrafanaDatasourceSpec{
			GrafanaCommonSpec: v1beta1.GrafanaCommonSpec{
				ResyncPeriod:              metav1.Duration{Duration: 10 * time.Minute},
				AllowCrossNamespaceImport: true,
				InstanceSelector:          cfg.instanceSelector(),
			},
			// isn't there a way to get *GrafanaDatasourceInternal directly?
			Datasource: &v1beta1.GrafanaDatasourceInternal{
				UID:            datasource.UID,
				Name:           datasource.Name,
				Type:           datasource.Type,
				URL:            datasource.URL,
				Access:         string(datasource.Access),
				Database:       datasource.Database,
				User:           datasource.User,
				IsDefault:      &datasource.IsDefault,
				BasicAuth:      &datasource.BasicAuth,
				BasicAuthUser:  datasource.BasicAuthUser,
				OrgID:          &datasource.OrgID,
				Editable:       constP(false), // TODO: editable even if this is false.
				JSONData:       jsonData,
				SecureJSONData: nil, // unavailable from the grafana API.
			},
		},
	}
}
