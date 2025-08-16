package main

import (
	"iter"
	"log/slog"

	"codeberg.org/clambin/go-common/set"
	"github.com/grafana/grafana-openapi-client-go/client/dashboards"
	"github.com/grafana/grafana-openapi-client-go/client/datasources"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/grafana-openapi-client-go/models"
)

type grafanaClient struct {
	Search      grafanaSearchClient
	Dashboards  grafanaDashboardClient
	Datasources grafanaDatasourcesClient
}

type grafanaSearchClient interface {
	Search(*search.SearchParams, ...search.ClientOption) (*search.SearchOK, error)
}

type grafanaDashboardClient interface {
	GetDashboardByUID(string, ...dashboards.ClientOption) (*dashboards.GetDashboardByUIDOK, error)
}

type grafanaDatasourcesClient interface {
	GetDataSourceByName(name string, opts ...datasources.ClientOption) (*datasources.GetDataSourceByNameOK, error)
}

// grafanaDashboards returns all Grafana dashboards that match args.
// If folders is false, it returns all dashboards whose title matches an element of args.
// Otherwise, it returns all dashboards in folders that matches an element of args.
func grafanaDashboards(c *grafanaClient, folders bool, args set.Set[string], logger *slog.Logger) iter.Seq2[*models.Hit, *models.DashboardFullWithMeta] {
	return func(yield func(*models.Hit, *models.DashboardFullWithMeta) bool) {
		params := search.SearchParams{Type: constP("dash-db")}
		var page int64
		for page = 1; ; page++ {
			params.Page = &page
			ok, err := c.Search.Search(&params)
			if err != nil {
				logger.Error("Error getting dashboards", "err", err)
				return
			}
			hits := ok.GetPayload()
			if len(hits) == 0 {
				return
			}
			for _, entry := range hits {
				if len(args) > 0 {
					if (!folders && !args.Contains(entry.Title)) ||
						(folders && !args.Contains(entry.FolderTitle)) {
						continue
					}
				}
				db, err := c.Dashboards.GetDashboardByUID(entry.UID)
				if err != nil {
					logger.Error("Error getting dashboard", "err", err, "uid", entry.UID, "title", entry.Title)
					return
				}
				if !yield(entry, db.GetPayload()) {
					return
				}
			}
		}
	}
}

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

func constP[T any](v T) *T {
	return &v
}
