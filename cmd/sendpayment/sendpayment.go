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

func newSendPaymentCmd(logger *slog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sendpayment",
		Short: "send a payment to an allocation",
		RunE:  sendPaymentE(logger),
	}

	cmd.Flags().String("private-key-file", "", "the sender private key file. (if not provided, NETWORK_PAYMENT_PRIVATE_KEY env var will be used for the private key value directly)")
	cmd.Flags().String("allocation-id", "", "the allocation ID to pay to")
	cmd.Flags().String("deployment-id", "", "the deployment ID of the service being allocated to")
	cmd.Flags().Uint64("amount", 0, "the amount to pay")
	cmd.Flags().String("rpc-url", os.Getenv("ARBITRUM_RPC_URL"), "the rpc url. if not provided, will check the ARBITRUM_RPC_URL env var")
	cmd.Flags().Int64("gas-price", 0, "the gas price to use for the transaction. If 0, the gas price will be fetched from the network")

	return cmd
}

func sendPaymentE(logger *slog.Logger) func(cmd *cobra.Command, args []string) (err error) {
	return func(cmd *cobra.Command, args []string) (err error) {
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		allocation, err := cmd.Flags().GetString("allocation-id")
		if err != nil {
			return err
		}

		deploymentID, err := cmd.Flags().GetString("deployment-id")
		if err != nil {
			return err
		}
		if deploymentID == "" {
			return fmt.Errorf("deployment ID is required")
		}

		amount, err := cmd.Flags().GetUint64("amount")
		if err != nil {
			return err
		}

		rpcUrl, err := cmd.Flags().GetString("rpc-url")
		if err != nil {
			return err
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
		ctx = utils.WithPrivateKey(ctx, privateKey)

		rpcClient := ethrpc.NewClient(rpcUrl)

		senderAddress := privateKey.PublicKey().Address().String()

		approvedTrx, err := approveCall(ctx, utils.GRTTokenContractAddress, senderAddress, rpcClient, utils.StakingContractAddress, amount, gasPrice)
		if err != nil {
			return fmt.Errorf("failed to approve: %w", err)
		}

		if approvedTrx == "" {
			return fmt.Errorf("failed to approve. trx is empty")
		}

		isCurated, err := utils.IsCuratedCall(ctx, rpcClient, deploymentID)
		if err != nil {
			return fmt.Errorf("failed to check if curated: %w", err)
		}
		if isCurated {
			return fmt.Errorf("deployment has curation and cannot be paid to. please use a different deployment and open a new allocation")
		}

		collectedTrx, err := collectCall(ctx, utils.StakingContractAddress, senderAddress, rpcClient, allocation, amount)
		if err != nil {
			return fmt.Errorf("failed to collect: %w", err)
		}

		if collectedTrx == "" {
			return fmt.Errorf("failed to collect. trx is empty")
		}

		fmt.Println("Payment sent")
		fmt.Printf("%d sent to allocation %s\n", amount, allocation)
		fmt.Printf("See transaction on arbiscan: %s\n", fmt.Sprintf("https://arbiscan.io/tx/%s", collectedTrx))

		return nil
	}
}

func approveCall(ctx context.Context, to string, from string, cli *ethrpc.Client, addr string, amt uint64, gasPrice int64) (string, error) {
	methodDef, err := eth.NewMethodDef("approve(address,uint256)")
	if err != nil {
		return "", err
	}

	address := eth.MustNewAddress(addr)
	amount := utils.ConvertToWei(amt)

	methodCall := methodDef.NewCall()
	methodCall.AppendArg(address)
	methodCall.AppendArg(amount)

	data, err := methodCall.Encode()
	if err != nil {
		return "", err
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
		big.NewInt(0), //,
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
		return "", fmt.Errorf("failed to approve. receipt is nil")
	}

	return resp, nil
}

func collectCall(ctx context.Context, to string, from string, cli *ethrpc.Client, allocation string, amt uint64) (string, error) {
	methodDef, err := eth.NewMethodDef("collect(uint256,address)")
	if err != nil {
		return "", err
	}

	allocationID := eth.MustNewAddress(allocation)
	amount := utils.ConvertToWei(amt)

	methodCall := methodDef.NewCall()
	methodCall.AppendArg(amount)
	methodCall.AppendArg(allocationID)

	data, err := methodCall.Encode()
	if err != nil {
		return "", err
	}

	signer, err := native.NewPrivateKeySigner(zap.NewNop(), utils.MustGetChainID(ctx, cli), utils.MustGetPrivateKey(ctx))
	if err != nil {
		return "", fmt.Errorf("unable to create signer: %w", err)
	}
	signedTx, err := signer.SignTransaction(
		utils.MustGetAccountNonce(ctx, eth.MustNewAddress(from), cli),
		eth.MustNewAddress(to),
		big.NewInt(0),
		big.NewInt(5000000).Uint64(),
		utils.MustGetGasPrice(ctx, cli),
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
		return "", fmt.Errorf("failed to approve. receipt is nil")
	}

	return resp, nil
}
