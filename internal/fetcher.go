package internal

import (
	"fmt"
	gapi "github.com/grafana/grafana-api-golang-client"
	"iter"
	"log/slog"
)

type DashboardClient interface {
	Dashboards() ([]gapi.FolderDashboardSearchResponse, error)
	DashboardByUID(uid string) (*gapi.Dashboard, error)
}

type DataSourcesClient interface {
	DataSources() ([]*gapi.DataSource, error)
}

type Dashboards []Dashboard

type Dashboard struct {
	Folder string
	Title  string
	Model  map[string]any
}

func FetchDashboards(c DashboardClient, logger *slog.Logger, shouldExport func(gapi.FolderDashboardSearchResponse) bool) (Dashboards, error) {
	foundBoards, err := c.Dashboards()
	if err != nil {
		return nil, fmt.Errorf("grafana search: %w", err)
	}

	dashboards := make(Dashboards, 0, len(foundBoards))
	for board := range dashboardsToExport(foundBoards, shouldExport) {
		logger.Debug("dashboard found", "data", folderDashboard(board))
		rawBoard, err := c.DashboardByUID(board.UID)
		if err != nil {
			return nil, fmt.Errorf("grafana get board: %w", err)
		}
		dashboards = append(dashboards, Dashboard{
			Folder: board.FolderTitle,
			Title:  board.Title,
			Model:  rawBoard.Model,
		})
	}

	return dashboards, nil
}

func dashboardsToExport(dashboards []gapi.FolderDashboardSearchResponse, shouldExport func(gapi.FolderDashboardSearchResponse) bool) iter.Seq[gapi.FolderDashboardSearchResponse] {
	return func(yield func(gapi.FolderDashboardSearchResponse) bool) {
		for _, board := range dashboards {
			// Only process dashboards, not folders
			// Only export if the dashboard meets the criteria
			if board.Type == "dash-db" && shouldExport(board) {
				if !yield(board) {
					return
				}
			}
		}
	}
}

var _ slog.LogValuer = folderDashboard{}

type folderDashboard gapi.FolderDashboardSearchResponse

func (d folderDashboard) LogValue() slog.Value {
	return slog.GroupValue(slog.String("title", d.Title),
		slog.String("type", d.Type),
		slog.String("folder", d.FolderTitle),
	)
}
