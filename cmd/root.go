package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/airfocusio/kube-network-monitor/internal"
	"github.com/spf13/cobra"
)

var (
	verbose             bool
	rootCmdSelfNodeName string
	rootCmdInterval     time.Duration
	rootCmd             = &cobra.Command{
		Use: "kube-network-monitor",
		RunE: func(cmd *cobra.Command, args []string) error {
			if rootCmdSelfNodeName == "" {
				return fmt.Errorf("self-node-name is required")
			}
			service, err := internal.NewService(internal.ServiceOpts{
				SelfNodeName: rootCmdSelfNodeName,
				Interval:     rootCmdInterval,
			})
			if err != nil {
				return err
			}

			term := make(chan os.Signal, 1)
			signal.Notify(term, syscall.SIGTERM)
			signal.Notify(term, syscall.SIGINT)
			return service.Run(term)
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if !verbose {
				internal.Debug = log.New(ioutil.Discard, "", log.LstdFlags)
			}
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "")
	rootCmd.Flags().StringVar(&rootCmdSelfNodeName, "self-node-name", "", "")
	rootCmd.Flags().DurationVar(&rootCmdInterval, "interval", 5*time.Second, "")
	rootCmd.AddCommand(versionCmd)
}
