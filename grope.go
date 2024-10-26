package main

import (
	"github.com/clambin/go-common/charmer"
	"github.com/clambin/grope/internal"
	"os"
)

func main() {
	if err := internal.RootCmd.Execute(); err != nil {
		charmer.GetLogger(&internal.RootCmd).Error("failed to run", "err", err)
		os.Exit(1)
	}
}
