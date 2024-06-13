package internal_test

import (
	"fmt"
	"github.com/clambin/grope/internal"
	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"testing"
)

func TestFetchDashboards(t *testing.T) {
	tests := []struct {
		name    string
		folders []string
		want    internal.Dashboards
	}{
		{
			name:    "all folders",
			folders: nil,
			want: internal.Dashboards{
				{Folder: "foo", Title: "board 1", Model: map[string]any{"foo": "bar"}},
				{Folder: "bar", Title: "board 2", Model: map[string]any{"bar": "foo"}},
			},
		},
		{
			name:    "filtered",
			folders: []string{"foo"},
			want: internal.Dashboards{
				{Folder: "foo", Title: "board 1", Model: map[string]any{"foo": "bar"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := internal.FetchDashboards(&fakeDashboardFetcher{}, slog.Default(), tt.folders...)
			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

var _ internal.DashboardClient = &fakeDashboardFetcher{}

type fakeDashboardFetcher struct {
}

func (f fakeDashboardFetcher) Dashboards() ([]gapi.FolderDashboardSearchResponse, error) {
	return []gapi.FolderDashboardSearchResponse{
		{UID: "1", Title: "board 1", Type: "dash-db", FolderTitle: "foo"},
		{UID: "2", Title: "board 2", Type: "dash-db", FolderTitle: "bar"},
		{UID: "3", Title: "foo", Type: "folder", FolderTitle: ""},
		{UID: "4", Title: "bar", Type: "folder", FolderTitle: ""},
	}, nil
}

func (f fakeDashboardFetcher) DashboardByUID(uid string) (*gapi.Dashboard, error) {
	dashboards := map[string]*gapi.Dashboard{
		"1": {Model: map[string]any{"foo": "bar"}},
		"2": {Model: map[string]any{"bar": "foo"}},
	}

	dashboard, ok := dashboards[uid]
	if !ok {
		return nil, fmt.Errorf("invalid dashboard uid: %s", uid)
	}
	return dashboard, nil
}
