package internal

import (
	"errors"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_yieldDashboards(t *testing.T) {
	tests := []struct {
		name     string
		searcher searcher
		fetcher  dashboardFetcher
		want     []string
		wantErr  assert.ErrorAssertionFunc
	}{
		{
			name: "pass",
			searcher: fakeSearcher{
				hitList: models.HitList{{Title: "dashboard", Type: "dash-db", UID: "1"}},
			},
			fetcher: fakeDashboardFetcher{
				dashboards: map[string]any{"1": "model"},
			},
			want:    []string{"dashboard"},
			wantErr: assert.NoError,
		},
		{
			name:     "search fails",
			searcher: fakeSearcher{err: errors.New("some error")},
			wantErr:  assert.Error,
		},
		{
			name: "fetch fails",
			searcher: fakeSearcher{
				hitList: models.HitList{{Title: "dashboard", Type: "dash-db", UID: "1"}},
			},
			fetcher: fakeDashboardFetcher{err: errors.New("some error")},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := dashboardClient{
				searcher:         tt.searcher,
				dashboardFetcher: tt.fetcher,
			}
			var dashboardNames []string
			var yieldErr error
			for db, err := range yieldDashboards(c, false) {
				if err != nil {
					yieldErr = err
					break
				}
				dashboardNames = append(dashboardNames, db.Title)
			}
			assert.Equal(t, tt.want, dashboardNames)
			tt.wantErr(t, yieldErr)
		})
	}
}

func Test_getDataSources(t *testing.T) {
	tests := []struct {
		name    string
		fetcher dataSourceFetcher
		want    []string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "pass",
			fetcher: fakeDataSourceFetcher{
				dataSources: models.DataSourceList{
					{Name: "foo"},
					{Name: "bar"},
				},
			},
			want:    []string{"foo", "bar"},
			wantErr: assert.NoError,
		},
		{
			name: "failure",
			fetcher: fakeDataSourceFetcher{
				err: errors.New("some error"),
			},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := dataSourcesClient{dataSourceFetcher: tt.fetcher}
			dataSources, err := getDataSources(c)
			var dataSourceNames []string
			for _, dataSource := range dataSources {
				dataSourceNames = append(dataSourceNames, dataSource.Name)
			}
			assert.Equal(t, tt.want, dataSourceNames)
			tt.wantErr(t, err)
		})
	}
}
