package internal

import (
	"fmt"
	"github.com/clambin/go-common/set"
	"github.com/grafana/grafana-openapi-client-go/client/dashboards"
	"github.com/grafana/grafana-openapi-client-go/client/datasources"
	"github.com/grafana/grafana-openapi-client-go/client/search"
	"github.com/grafana/grafana-openapi-client-go/models"
	"iter"
)

type dashboardClient struct {
	searcher
	dashboardFetcher
}

type searcher interface {
	Search(*search.SearchParams, ...search.ClientOption) (*search.SearchOK, error)
}

type dashboardFetcher interface {
	GetDashboardByUID(string, ...dashboards.ClientOption) (*dashboards.GetDashboardByUIDOK, error)
}

type dataSourcesClient struct {
	dataSourceFetcher
}

type dataSourceFetcher interface {
	GetDataSources(opts ...datasources.ClientOption) (*datasources.GetDataSourcesOK, error)
}

type Dashboards []Dashboard

type Dashboard struct {
	Folder string
	Title  string
	Model  models.JSON
}

func yieldDashboards(c dashboardClient, folders bool, args ...string) iter.Seq2[Dashboard, error] {
	f := shouldExport(folders, args...)
	dashboardType := "dash-db"
	params := search.SearchParams{Type: &dashboardType}
	return func(yield func(Dashboard, error) bool) {
		var page int64
		for page = 0; ; page++ {
			params.Page = &page
			ok, err := c.Search(&params)
			if err != nil {
				yield(Dashboard{}, err)
				return
			}
			hits := ok.GetPayload()
			if len(hits) == 0 {
				return
			}
			for _, entry := range hits {
				if !f(entry) {
					continue
				}
				db, err := c.GetDashboardByUID(entry.UID)
				if err != nil {
					yield(Dashboard{}, fmt.Errorf("dash-db lookup for %q: %w", entry.Title, err))
					return
				}
				if !yield(Dashboard{Title: entry.Title, Folder: entry.FolderTitle, Model: db.GetPayload().Dashboard}, nil) {
					return
				}
			}
		}
	}
}

func shouldExport(folders bool, args ...string) func(*models.Hit) bool {
	validNames := set.New(args...)
	return func(hit *models.Hit) bool {
		if len(args) == 0 {
			return true
		}
		if folders {
			return validNames.Contains(hit.FolderTitle)
		}
		return validNames.Contains(hit.Title)
	}
}

func getDataSources(c dataSourcesClient) (models.DataSourceList, error) {
	ok, err := c.GetDataSources()
	if err != nil {
		return nil, fmt.Errorf("getDatasources: %w", err)
	}
	return ok.GetPayload(), nil
}
