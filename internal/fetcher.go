package internal

import (
	"fmt"
	"github.com/clambin/go-common/set"
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

func FetchDashboards(c DashboardClient, logger *slog.Logger, folders ...string) (Dashboards, error) {
	folderSet := set.New[string](folders...)
	foundBoards, err := c.Dashboards()
	if err != nil {
		return nil, fmt.Errorf("grafana search: %w", err)
	}

	dashboards := make(Dashboards, 0, len(foundBoards))
	for _, board := range foundBoards {
		logger.Debug("dashboard found", "title", board.Title, "type", board.Type, "folder", board.FolderTitle)

		// Only process dashboards, not folders
		if board.Type != "dash-db" {
			logger.Debug("invalid type in dashboard. ignoring", "type", board.Type)
			continue
		}

		// Only export if the dashboard is in a specified folder
		if len(folders) > 0 && !folderSet.Contains(board.FolderTitle) {
			logger.Debug("folder not in scope. ignoring", "folderTitle", board.FolderTitle, "title", board.Title)
			continue
		}

		// Get the dashboard model
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
