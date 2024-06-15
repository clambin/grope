package internal

import (
	"bytes"
	"github.com/clambin/go-common/testutils"
	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"testing"
)

func Test_folderDashboard_LogValue(t *testing.T) {
	dashboard := gapi.FolderDashboardSearchResponse{FolderTitle: "folder", Title: "dashboard", Type: "db"}

	var output bytes.Buffer
	l := testutils.NewTextLogger(&output, slog.LevelInfo)
	l.Info("dashboard found", "dashboard", folderDashboard(dashboard))
	assert.Equal(t, `level=INFO msg="dashboard found" dashboard.title=dashboard dashboard.type=db dashboard.folder=folder
`, output.String())
}
