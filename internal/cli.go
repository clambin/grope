package internal

import (
	"github.com/clambin/go-common/charmer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log/slog"
	"os"
)

var (
	configFilename string
	RootCmd        = cobra.Command{
		Use:   "grafana-exporter",
		Short: "exports Grafana dashboards & datasources",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			charmer.SetTextLogger(cmd, viper.GetBool("debug"))
		},
	}
	dashboardsCmd = &cobra.Command{
		Use:   "dashboards",
		Short: "export Grafana dashboards",
		RunE:  exportDashboards,
	}
	dataSourcesCmd = &cobra.Command{
		Use:   "datasources",
		Short: "export Grafana data sources provisioning",
		RunE:  ExportDataSources,
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	initArgs()
}

var args = charmer.Arguments{
	"debug":                        {Default: false, Help: "Log debug messages"},
	"namespace":                    {Default: "default", Help: "Namespace for k8s config maps"},
	"grafana.url":                  {Default: "http://localhost:3000", Help: "Grafana URL"},
	"grafana.token":                {Default: "", Help: "Grafana API token (must have admin rights)"},
	"grafana.operator.label.name":  {Default: "dashboards", Help: "label used to select the grafana instance (grafana-operator only)"},
	"grafana.operator.label.value": {Default: "grafana", Help: "label value used to select the grafana instance (grafana-operator only)"},
}

func initArgs() {
	//RootCmd.Version = version.BuildVersion

	RootCmd.PersistentFlags().StringVarP(&configFilename, "config", "c", "", "Configuration file")
	if err := charmer.SetPersistentFlags(&RootCmd, viper.GetViper(), args); err != nil {
		panic("failed to set flags: " + err.Error())
	}

	dashboardsCmd.Flags().StringP("folders", "f", "", "Dashboard folders to export")
	_ = viper.BindPFlag("folders", dashboardsCmd.Flags().Lookup("folders"))

	RootCmd.AddCommand(dashboardsCmd)
	RootCmd.AddCommand(dataSourcesCmd)
}

func initConfig() {
	if configFilename != "" {
		viper.SetConfigFile(configFilename)
	} else {
		viper.AddConfigPath("/etc/grope/")
		viper.AddConfigPath("$HOME/.grope")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("GRAFANA_EXPORTER")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		slog.Warn("failed to read config file", "err", err)
	}
}

func exportDashboards(cmd *cobra.Command, _ []string) error {
	exp, err := makeExporter(viper.GetViper(), charmer.GetLogger(cmd))
	if err != nil {
		return err
	}

	return exp.exportDashboards(os.Stdout)
}

func ExportDataSources(cmd *cobra.Command, _ []string) error {
	exp, err := makeExporter(viper.GetViper(), charmer.GetLogger(cmd))
	if err != nil {
		return err
	}
	return exp.exportDataSources(os.Stdout)
}
