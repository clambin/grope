package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gosimple/slug"
	gapi "github.com/grafana/grafana-api-golang-client"
	grafanav1beta1 "github.com/grafana/grafana-operator/v5/api/v1beta1"
	"gopkg.in/yaml.v3"
	"io"
)

type Formatter struct {
	Namespace         string
	GrafanaLabelName  string
	GrafanaLabelValue string
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
	Namespace string `yaml:"namespace"`
}

type grafanaOperatorCustomResourceSpec struct {
	AllowCrossNamespaceImport bool             `yaml:"allowCrossNamespaceImport"`
	Folder                    string           `yaml:"folder,omitempty"`
	InstanceSelector          instanceSelector `yaml:"instanceSelector"`
	Json                      string           `yaml:"json,omitempty"`
	DataSource                *gapi.DataSource `yaml:"datasource,omitempty"`
}

type instanceSelector struct {
	MatchLabels map[string]string `yaml:"matchLabels"`
}

func (f Formatter) FormatDashboard(w io.Writer, dashboard Dashboard) error {
	var encodedDashboard bytes.Buffer
	jEnc := json.NewEncoder(&encodedDashboard)
	jEnc.SetIndent("", "  ")
	if err := jEnc.Encode(dashboard.Model); err != nil {
		return fmt.Errorf("encode dashboard model: %w", err)
	}

	dashboardCR := grafanaOperatorCustomResource{
		APIVersion: grafanav1beta1.GroupVersion.String(),
		Kind:       "GrafanaDashboard",
		Metadata: metadata{
			Name:      slug.Make(dashboard.Title),
			Namespace: f.Namespace,
		},
		Spec: grafanaOperatorCustomResourceSpec{
			AllowCrossNamespaceImport: true,
			InstanceSelector: instanceSelector{
				MatchLabels: map[string]string{
					f.GrafanaLabelName: f.GrafanaLabelValue,
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

func (f Formatter) FormatDataSources(w io.Writer, dataSources []*gapi.DataSource) error {
	for _, dataSource := range dataSources {
		cr := grafanaOperatorCustomResource{
			APIVersion: grafanav1beta1.GroupVersion.String(),
			Kind:       "GrafanaDataSource",
			Metadata: metadata{
				Name:      "datasource-" + slug.Make(dataSource.Name),
				Namespace: f.Namespace,
			},
			Spec: grafanaOperatorCustomResourceSpec{
				InstanceSelector: instanceSelector{
					MatchLabels: map[string]string{
						f.GrafanaLabelName: f.GrafanaLabelValue,
					},
				},
				DataSource: dataSource,
			},
		}
		_, _ = w.Write([]byte("---\n"))
		yEnc := yaml.NewEncoder(w)
		yEnc.SetIndent(2)
		if err := yEnc.Encode(cr); err != nil {
			return fmt.Errorf("encode data source cr: %w", err)
		}
	}
	return nil
}
