/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	metricsendpoint "github.com/bluedigital/prometheus-cloudwatch-exporter/internal/pkg/metrics-endpoint"
	cwexporter "github.com/bluedigital/prometheus-cloudwatch-exporter/pkg/cw-exporter"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	homedir "github.com/mitchellh/go-homedir"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var signalCh chan os.Signal = make(chan os.Signal, 1)
var cfgFile string
var rootCmd = &cobra.Command{
	Use:   "prometheus-cloudwatch-exporter",
	Short: "Prometheus CloudWatch exporter",
	Run: func(cmd *cobra.Command, args []string) {
		e := cwexporter.NewExporter(
			cwexporter.WithLogger(&log.Logger),
			cwexporter.WithViper(viper.GetViper()),
		)
		m := metricsendpoint.NewMetricsEndpoint(e)

		err := e.Load()
		if err != nil {
			log.Fatal().Err(err).Send()
		}
		m.Start()
		log.Info().Msg("prometheus cloudwatch exporter is up")

		signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		switch <-signalCh {
		case syscall.SIGHUP:
			e.Load()
		case syscall.SIGTERM:
			m.Stop()
			fallthrough
		case syscall.SIGINT:
			os.Exit(0)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootFlags := rootCmd.PersistentFlags()

	rootFlags.StringVar(&cfgFile, "config", "", "config file (default is $HOME/.prometheus-cloudwatch-exporter.yaml)")

	rootFlags.String("metrics-address", "0.0.0.0:9016", "Address to provide metrics endpoint")
	rootFlags.Duration("metrics-read-timeout", 5*time.Second, "Read body maximum duration")
	rootFlags.Duration("metrics-read-header-timeout", 1*time.Second, "Read header maximum duration")
	rootFlags.Duration("metrics-idle-timeout", 1*time.Second, "Idle connection maximum duration")
	rootFlags.Duration("metrics-shutdown-timeout", 1*time.Second, "Shutdown server maximum duration")
	rootFlags.String("metrics-path", "/metrics", "Metrics HTTP path")

	rootFlags.String("aws-region", "us-east-1", "AWS default region")
	rootFlags.String("aws-access-key-id", "", "AWS access key ID")
	rootFlags.String("aws-secret-access-key", "", "AWS secret access key")
	rootFlags.String("aws-session-token", "", "AWS session token")
	rootFlags.Duration("aws-walk-scrape", 0, "Walk a period until endTime >= now - walkDuration")
	rootFlags.Int("aws-max-retries", -1, "Maximum number to retry AWS operations, -1 will set to services specific configuration")

	rootFlags.VisitAll(func(flag *pflag.Flag) {
		if flag.Name == "config" {
			return
		}
		if err := viper.BindPFlag(strings.Replace(flag.Name, "-", "_", -1), flag); err != nil {
			panic(err)
		}
	})
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		viper.GetViper()
	} else {
		home, err := homedir.Dir()
		if err != nil {
			log.Fatal().Err(err).Send()
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".prometheus-cloudwatch-exporter")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Info().Str("config", viper.ConfigFileUsed()).Msgf("config file")
	}
}
