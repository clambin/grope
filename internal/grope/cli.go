package grope

import (
	"codeberg.org/clambin/go-common/charmer"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"runtime/debug"
)

var (
	configFilename string
	RootCmd        = cobra.Command{
		Use:   "grope",
		Short: "exports Grafana dashboards & datasources as grafana-operator custom resources",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			charmer.SetTextLogger(cmd, viper.GetBool("debug"))
		},
	}
	dashboardsCmd = &cobra.Command{
		Use:   "dashboards [flags] [name [...]]",
		Short: "export Grafana dashboards",
		RunE: func(cmd *cobra.Command, args []string) error {
			exp, err := makeExporter(viper.GetViper(), charmer.GetLogger(cmd))
			if err != nil {
				return err
			}
			return exp.exportDashboards(os.Stdout, args...)
		},
	}
	dataSourcesCmd = &cobra.Command{
		Use:   "datasources",
		Short: "export Grafana data sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			exp, err := makeExporter(viper.GetViper(), charmer.GetLogger(cmd))
			if err != nil {
				return err
			}
			return exp.exportDataSources(os.Stdout)
		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	initArgs()
}

var args = charmer.Arguments{
	"debug":                        {Default: false, Help: "Log debug messages"},
	"namespace":                    {Default: "default", Help: "Namespace for k8s config maps"},
	"tags":                         {Default: "", Help: "Dashboard tags (comma-separated; optional)"},
	"grafana.url":                  {Default: "http://localhost:3000", Help: "Grafana URL"},
	"grafana.token":                {Default: "", Help: "Grafana API token (must have admin rights)"},
	"grafana.operator.label.name":  {Default: "dashboards", Help: "label used to select the grafana instance (grafana-operator only)"},
	"grafana.operator.label.value": {Default: "grafana", Help: "label value used to select the grafana instance (grafana-operator only)"},
}

func initArgs() {
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		RootCmd.Version = buildInfo.Main.Version
	}

	RootCmd.PersistentFlags().StringVarP(&configFilename, "config", "c", "", "Configuration file")
	_ = charmer.SetPersistentFlags(&RootCmd, viper.GetViper(), args)
	dashboardsCmd.Flags().BoolP("folders", "f", false, "Export folder")
	_ = viper.BindPFlag("folders", dashboardsCmd.Flags().Lookup("folders"))

	RootCmd.AddCommand(dashboardsCmd, dataSourcesCmd)
}

func initConfig() {
	initViper(viper.GetViper())
}

func initViper(v *viper.Viper) {
	if configFilename != "" {
		v.SetConfigFile(configFilename)
	} else {
		v.AddConfigPath("/etc/grope/")
		v.AddConfigPath("$HOME/.grope")
		v.AddConfigPath(".")
		v.SetConfigName("config")
	}

	v.SetEnvPrefix("GROPE")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to read config file: %s\n", err.Error())
	}
}
