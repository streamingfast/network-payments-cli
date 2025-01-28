package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/streamingfast/eth-go"
	ethrpc "github.com/streamingfast/eth-go/rpc"
	"github.com/streamingfast/eth-go/signer/native"
	"github.com/streamingfast/network-payments-cli/cmd/utils"
	"go.uber.org/zap"
)

func newCloseAllocationCmd(logger *slog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close-allocation",
		Short: "close an allocation",
		RunE:  closeAllocationE(logger),
	}

	cmd.Flags().String("private-key-file", "", "the private key file (if not provided, NETWORK_PAYMENT_PRIVATE_KEY env var will be used for the private key value directly)")
	cmd.Flags().String("allocation-id", "", "the allocation ID to close")
	cmd.Flags().String("rpc-url", os.Getenv("ARBITRUM_RPC_URL"), "the rpc url. if not provided, will check the ARBITRUM_RPC_URL env var")
	cmd.Flags().Int64("gas-price", 0, "the gas price to use for the transaction. If 0, the gas price will be fetched from the network")

	return cmd
}

func closeAllocationE(logger *slog.Logger) func(cmd *cobra.Command, args []string) (err error) {
	return func(cmd *cobra.Command, args []string) (err error) {
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		ctx = utils.WithLogger(ctx, logger)

		rpcUrl, err := cmd.Flags().GetString("rpc-url")
		if err != nil {
			return err
		}

		allocationID, err := cmd.Flags().GetString("allocation-id")
		if err != nil {
			return err
		}
		if allocationID == "" {
			return fmt.Errorf("deployment ID is required")
		}

		gasPrice, err := cmd.Flags().GetInt64("gas-price")
		if err != nil {
			return err
		}

		var privateKey *eth.PrivateKey
		privateKeyFile, err := cmd.Flags().GetString("private-key-file")
		if err != nil {
			return err
		}
		if privateKeyFile == "" {
			privateKey, err = eth.NewPrivateKey(os.Getenv("NETWORK_PAYMENT_PRIVATE_KEY"))
			if err != nil {
				return fmt.Errorf("import private key: %s", err)
			}
		} else {
			pkBytes, err := os.ReadFile(privateKeyFile)
			if err != nil {
				return fmt.Errorf("read private key file: %s", err)
			}
			pkHex := strings.TrimSpace(string(pkBytes))
			privateKey, err = eth.NewPrivateKey(pkHex)
			if err != nil {
				return fmt.Errorf("import private key: %s", err)
			}
		}
		if privateKey == nil || privateKey.String() == "" {
			return fmt.Errorf("private key is required, either through the NETWORK_PAYMENT_PRIVATE_KEY environment variable or --private-key-file flag")
		}

		ctx = utils.WithPrivateKey(ctx, privateKey)

		rpcClient := ethrpc.NewClient(rpcUrl)

		closeTrx, err := closeAllocationCall(ctx, utils.StakingContractAddress, privateKey.PublicKey().Address().String(), rpcClient, allocationID, gasPrice)
		if err != nil {
			return err
		}

		fmt.Println("Allocation closed successfully")
		fmt.Printf("See transaction on arbiscan: https://arbiscan.io/tx/%s\n", closeTrx)

		return nil
	}
}

func closeAllocationCall(ctx context.Context, to string, from string, cli *ethrpc.Client, allocationID string, gasPrice int64) (string, error) {
	methodDef, err := eth.NewMethodDef("closeAllocation(address,bytes32)")
	if err != nil {
		return "", fmt.Errorf("creating method definition: %w", err)
	}

	allocationIDAddress := eth.MustNewAddress(allocationID)

	methodCall := methodDef.NewCall()
	methodCall.AppendArg(allocationIDAddress)
	methodCall.AppendArg(make([]byte, 32)) // empty poi

	data, err := methodCall.Encode()
	if err != nil {
		return "", fmt.Errorf("encoding method call: %w", err)
	}

	signer, err := native.NewPrivateKeySigner(zap.NewNop(), utils.MustGetChainID(ctx, cli), utils.MustGetPrivateKey(ctx))
	if err != nil {
		return "", fmt.Errorf("unable to create signer: %w", err)
	}

	var gasPriceBigInt *big.Int
	if gasPrice == 0 {
		gasPriceBigInt = utils.MustGetGasPrice(ctx, cli)
	} else {
		gasPriceBigInt = big.NewInt(gasPrice)
	}

	signedTx, err := signer.SignTransaction(
		utils.MustGetAccountNonce(ctx, eth.MustNewAddress(from), cli),
		eth.MustNewAddress(to),
		big.NewInt(0),
		big.NewInt(7_000_000).Uint64(),
		gasPriceBigInt,
		data,
	)
	if err != nil {
		return "", err
	}

	resp, err := cli.SendRaw(ctx, signedTx)
	if err != nil {
		return "", err
	}

	receipt, err := utils.FetchReceiptWithProgress(ctx, cli, eth.MustNewHash(resp))
	if err != nil {
		return "", err
	}

	if receipt == nil {
		return "", fmt.Errorf("failed to close allocation. no receipt found for transaction %s", resp)
	}

	if len(receipt.Logs) == 0 {
		return "", fmt.Errorf("failed to close allocation. no logs found for transaction %s", resp)
	}

	return resp, nil
}
