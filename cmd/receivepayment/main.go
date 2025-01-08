package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd *cobra.Command

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	//recover any panics from "must" functions
	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic executing command", "err", r)
		}
	}()

	rootCmd = newReceivePaymentCmd()
	rootCmd.AddCommand(newOpenAllocationCmd(logger))
	rootCmd.AddCommand(newCloseAllocationCmd(logger))

	if err := rootCmd.Execute(); err != nil {
		logger.Error("error executing command", "err", err)
		os.Exit(1)
	}
}
