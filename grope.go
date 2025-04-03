package main

import (
	"codeberg.org/clambin/go-common/charmer"
	"github.com/clambin/grope/internal/grope"
	"os"
)

func main() {
	if err := grope.RootCmd.Execute(); err != nil {
		charmer.GetLogger(&grope.RootCmd).Error("failed to run", "err", err)
		os.Exit(1)
	}
}
