package grope

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"iter"

	"github.com/gosimple/slug"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/grafana-operator/v5/api/v1beta1"
	"gopkg.in/yaml.v3"
)

type formatter struct {
	namespace         string
	grafanaLabelName  string
	grafanaLabelValue string
}

// grafanaOperatorCustomResource mimics a grafana-operator GrafanaDashboard, but leaves out the Status section
type grafanaOperatorCustomResource struct {
	APIVersion string                            `yaml:"apiVersion"`
	Kind       string                            `yaml:"kind"`
	Metadata   metadata                          `yaml:"metadata"`
	Spec       grafanaOperatorCustomResourceSpec `yaml:"spec"`
}

type metadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace,omitempty"`
}

type grafanaOperatorCustomResourceSpec struct {
	AllowCrossNamespaceImport bool                          `yaml:"allowCrossNamespaceImport"`
	Folder                    string                        `yaml:"folder,omitempty"`
	InstanceSelector          instanceSelector              `yaml:"instanceSelector"`
	Json                      string                        `yaml:"json,omitempty"`
	DataSource                *models.DataSourceListItemDTO `yaml:"datasource,omitempty"`
}

type instanceSelector struct {
	MatchLabels map[string]string `yaml:"matchLabels"`
}

func (f formatter) formatDashboard(w io.Writer, dashboard Dashboard) error {
	var encodedDashboard bytes.Buffer
	jEnc := json.NewEncoder(&encodedDashboard)
	jEnc.SetIndent("", "  ")
	if err := jEnc.Encode(dashboard.Model); err != nil {
		return fmt.Errorf("encode dashboard model: %w", err)
	}

	dashboardCR := grafanaOperatorCustomResource{
		APIVersion: v1beta1.GroupVersion.String(),
		Kind:       "GrafanaDashboard",
		Metadata: metadata{
			Name:      slug.Make(dashboard.Title),
			Namespace: f.namespace,
		},
		Spec: grafanaOperatorCustomResourceSpec{
			AllowCrossNamespaceImport: true,
			InstanceSelector: instanceSelector{
				MatchLabels: map[string]string{
					f.grafanaLabelName: f.grafanaLabelValue,
				},
			},
			Folder: dashboard.Folder,
			Json:   encodedDashboard.String(),
		},
	}
	_, _ = w.Write([]byte("---\n"))
	yEnc := yaml.NewEncoder(w)
	yEnc.SetIndent(2)
	return yEnc.Encode(dashboardCR)
}

func (f formatter) formatDataSources(w io.Writer, dataSources []*models.DataSourceListItemDTO) error {
	for cr := range f.grafanaOperatorCustomResources(dataSources) {
		_, _ = w.Write([]byte("---\n"))
		yEnc := yaml.NewEncoder(w)
		yEnc.SetIndent(2)
		if err := yEnc.Encode(cr); err != nil {
			return fmt.Errorf("encode data source cr: %w", err)
		}
	}
	return nil
}

func (f formatter) grafanaOperatorCustomResources(dataSources []*models.DataSourceListItemDTO) iter.Seq[grafanaOperatorCustomResource] {
	return func(yield func(grafanaOperatorCustomResource) bool) {
		for _, dataSource := range dataSources {
			cr := grafanaOperatorCustomResource{
				APIVersion: v1beta1.GroupVersion.String(),
				Kind:       "GrafanaDatasource",
				Metadata: metadata{
					Name:      "datasource-" + slug.Make(dataSource.Name),
					Namespace: f.namespace,
				},
				Spec: grafanaOperatorCustomResourceSpec{
					InstanceSelector: instanceSelector{
						MatchLabels: map[string]string{
							f.grafanaLabelName: f.grafanaLabelValue,
						},
					},
					DataSource: dataSource,
				},
			}
			if !yield(cr) {
				return
			}
		}
	}
}
