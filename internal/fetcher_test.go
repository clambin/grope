package internal

import (
	"bytes"
	"github.com/clambin/go-common/testutils"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
)

func Test_folderDashboard_LogValue(t *testing.T) {
	dashboard := models.Hit{
		FolderTitle: "folder",
		Type:        "dash-db",
		Title:       "dashboard",
	}
	var output bytes.Buffer
	l := testutils.NewTextLogger(&output, slog.LevelInfo)
	l.Info("dashboard found", "dashboard", folderDashboard(dashboard))
	assert.Equal(t, `level=INFO msg="dashboard found" dashboard.title=dashboard dashboard.type=dash-db dashboard.folder=folder
`, output.String())
}
