package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"github.com/streamingfast/eth-go"
	ethrpc "github.com/streamingfast/eth-go/rpc"
	"github.com/streamingfast/eth-go/signer/native"
	"github.com/streamingfast/network-payments-cli/cmd/utils"
	"go.uber.org/zap"
)

func newOpenAllocationCmd(logger *slog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open-allocation",
		Short: "open an allocation",
		RunE:  openAllocationE(logger),
	}

	cmd.Flags().String("private-key-file", "", "the private key file (if not provided, NETWORK_PAYMENT_PRIVATE_KEY env var will be used for the private key value directly)")
	cmd.Flags().String("indexer-address", "", "the indexer address (note: NOT the operator address)")
	cmd.Flags().String("deployment-id", "", "the deployment ID of the service being allocated to. If left empty, a random deployment ID will be generated")
	cmd.Flags().Uint64("allocation-amount", 0, "the allocation amount in GRT")
	cmd.Flags().String("rpc-url", os.Getenv("ARBITRUM_RPC_URL"), "the rpc url. if not provided, will check the ARBITRUM_RPC_URL env var")
	cmd.Flags().Int64("gas-price", 0, "the gas price to use for the transaction. If 0, the gas price will be fetched from the network")

	return cmd
}

func openAllocationE(logger *slog.Logger) func(cmd *cobra.Command, args []string) (err error) {
	return func(cmd *cobra.Command, args []string) (err error) {
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		ctx = utils.WithLogger(ctx, logger)

		rpcUrl, err := cmd.Flags().GetString("rpc-url")
		if err != nil {
			return err
		}

		var deploymentID string
		deploymentID, err = cmd.Flags().GetString("deployment-id")
		if err != nil {
			return err
		}
		if deploymentID == "" {
			fmt.Println("No deployment ID provided, generating a random one")
			deploymentID, err = utils.GenerateDeployment()
			if err != nil {
				return fmt.Errorf("generating deployment ID: %w", err)
			}
		}

		amount, err := cmd.Flags().GetUint64("allocation-amount")
		if err != nil {
			return err
		}
		if amount == 0 {
			return fmt.Errorf("amount must be greater than 0")
		}

		indexerAddress := cmd.Flag("indexer-address").Value.String()
		if indexerAddress == "" {
			return fmt.Errorf("indexer address is required")
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

		if deploymentID != "" {
			isCurated, err := utils.IsCuratedCall(ctx, rpcClient, deploymentID)
			if err != nil {
				return fmt.Errorf("failed to check if curated: %w", err)
			}
			if isCurated {
				return fmt.Errorf("deployment has curation and cannot be paid to. please generate a different deployment and open a new allocation")
			}
		}

		allocateTrx, allocationID, err := allocateCall(ctx, utils.StakingContractAddress, privateKey.PublicKey().Address().String(), rpcClient, indexerAddress, deploymentID, amount, gasPrice)
		if err != nil {
			return err
		}

		fmt.Println("Allocation created with ID: ", eth.MustNewAddress(allocationID).Pretty())
		fmt.Println("Deployment ID: ", deploymentID)
		fmt.Printf("See transaction on arbiscan: %s\n", fmt.Sprintf("https://arbiscan.io/tx/%s", allocateTrx))

		return nil
	}
}

func allocateCall(ctx context.Context, to string, from string, cli *ethrpc.Client, indexerAddress string, deploymentID string, amt uint64, gasPrice int64) (string, string, error) {
	isCurated, err := utils.IsCuratedCall(ctx, cli, deploymentID)
	if err != nil {
		return "", "", fmt.Errorf("failed to check if curated: %w", err)
	}
	if isCurated {
		return "", "", fmt.Errorf("deployment has curation and cannot be paid to. please use a different deployment and open a new allocation")
	}

	methodDef, err := eth.NewMethodDef("allocateFrom(address,bytes32,uint256,address,bytes32,bytes)")
	if err != nil {
		return "", "", fmt.Errorf("creating method definition: %w", err)
	}

	amount := utils.ConvertToWei(amt)
	allocationIDBytes, proofBytes, err := generateAllocationIDAndProof(indexerAddress)
	if err != nil {
		return "", "", fmt.Errorf("generating proof: %w", err)
	}

	qm, err := utils.ConvertIPFSHashToByteString(deploymentID)
	if err != nil {
		return "", "", fmt.Errorf("converting IPFS hash to byte string: %w", err)
	}

	allocationID := hex.EncodeToString(allocationIDBytes)
	allocationIDAddress := eth.MustNewAddress(allocationID)

	methodCall := methodDef.NewCall()
	methodCall.AppendArg(eth.MustNewAddress(indexerAddress))
	methodCall.AppendArg(qm)
	methodCall.AppendArg(amount)
	methodCall.AppendArg(allocationIDAddress)
	methodCall.AppendArg(make([]byte, 32))
	methodCall.AppendArg(proofBytes)

	data, err := methodCall.Encode()
	if err != nil {
		return "", "", fmt.Errorf("encoding method call: %w", err)
	}

	signer, err := native.NewPrivateKeySigner(zap.NewNop(), utils.MustGetChainID(ctx, cli), utils.MustGetPrivateKey(ctx))
	if err != nil {
		return "", "", fmt.Errorf("unable to create signer: %w", err)
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
		return "", "", err
	}

	resp, err := cli.SendRaw(ctx, signedTx)
	if err != nil {
		return "", "", err
	}

	receipt, err := utils.FetchReceiptWithProgress(ctx, cli, eth.MustNewHash(resp))
	if err != nil {
		return "", "", err
	}

	if receipt == nil {
		return "", "", fmt.Errorf("failed to allocate. no receipt found for transaction %s", resp)
	}

	if len(receipt.Logs) == 0 {
		return "", "", fmt.Errorf("failed to allocate. no logs found for transaction %s", resp)
	}

	return resp, allocationID, nil
}

func generateAllocationIDAndProof(address string) ([]byte, []byte, error) { //returns allocationID, proof, err
	indexerAddress := common.HexToAddress(address)

	// Create a new ECDSA key, to generate a new allocation ID.
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("error generating key: %w", err)
	}
	allocationIDAddress := crypto.PubkeyToAddress(key.PublicKey)

	pk, err := eth.NewPrivateKey(hex.EncodeToString(crypto.FromECDSA(key)))
	if err != nil {
		return nil, nil, fmt.Errorf("error creating private key: %w", err)
	}

	messageHash := crypto.Keccak256Hash(bytes.Join([][]byte{indexerAddress.Bytes(), allocationIDAddress.Bytes()}, nil))

	// Sign the message hash
	signature, err := pk.SignPersonal(messageHash.Bytes())
	if err != nil {
		return nil, nil, fmt.Errorf("error signing message: %w", err)
	}

	invertedSignature := signature.ToInverted()

	// Verify signature
	recoveredAddress, err := invertedSignature.RecoverPersonal(messageHash.Bytes())
	if err != nil {
		return nil, nil, fmt.Errorf("error recovering address: %w", err)
	}

	if !bytes.Equal(recoveredAddress.Bytes(), allocationIDAddress.Bytes()) {
		return nil, nil, fmt.Errorf("recovered address does not match allocation ID")
	}

	return allocationIDAddress.Bytes(), invertedSignature[:], nil
}
