package main

import "github.com/spf13/cobra"

func newReceivePaymentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "receivepayment",
		Short: "receive a payment to an open allocation",
	}
}
