package internal

import (
	"fmt"
	gapi "github.com/grafana/grafana-api-golang-client"
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
	for _, board := range foundBoards {
		logger.Debug("dashboard found", "data", folderDashboard(board))

		// Only process dashboards, not folders
		// Only export if the dashboard meets the criteria
		if board.Type == "dash-db" && shouldExport(board) {
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
	}

	return dashboards, nil
}

var _ slog.LogValuer = folderDashboard{}

type folderDashboard gapi.FolderDashboardSearchResponse

func (d folderDashboard) LogValue() slog.Value {
	return slog.GroupValue(slog.String("title", d.Title),
		slog.String("type", d.Type),
		slog.String("folder", d.FolderTitle),
	)
}
