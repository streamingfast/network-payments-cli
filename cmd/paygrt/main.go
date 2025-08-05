package main

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/logging"
	"github.com/streamingfast/network-payments-cli/cmd/utils"
)

// Injected at build time
var version = ""

var zlog, tracer = logging.RootLogger("paygrt", "github.com/streamingfast/network-payments-cli")

func main() {
	logging.InstantiateLoggers()

	Run(
		"paygrt <alloc-amount> <pay-amount> <indexer>",
		"Write a SAFE multi-transaction JSON snippet for GRT payment on the network",

		Execute(run),

		ExactArgs(3),
		Description(`
			Write a SAFE multi-transaction JSON snippet for GRT payment on the network.
		`),
		Example(`
			paygrt 20 2 0x35917C0eB91d2E21BEF40940D028940484230c06
		`),

		ConfigureVersion(version),
		ConfigureViper("PAYGRT"),
		OnCommandErrorLogAndExit(zlog),
	)
}

func run(cmd *cobra.Command, args []string) error {
	allocAmountGRT, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid alloc-amount %q: %w", args[0], err)
	}

	payAmountGRT, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid pay-amount %q: %w", args[1], err)
	}

	indexerAddress := args[2]

	allocationIDBytes, proofBytes, err := utils.GenerateAllocationIDAndProof(indexerAddress)
	if err != nil {
		return fmt.Errorf("failed to generate allocation ID and proof: %w", err)
	}
	allocationID := "0x" + hex.EncodeToString(allocationIDBytes)
	proof := "0x" + hex.EncodeToString(proofBytes)

	deploymentQM, err := utils.GenerateDeployment()
	if err != nil {
		return fmt.Errorf("generating deployment ID: %w", err)
	}

	deploymentBytes, err := utils.ConvertIPFSHashToByteString(deploymentQM)
	if err != nil {
		return fmt.Errorf("failed to convert deployment ID to byte string: %w", err)
	}

	deploymentID := "0x" + hex.EncodeToString(deploymentBytes)

	json := generateJSON(
		indexerAddress,
		deploymentID,
		allocationID,
		proof,
		allocAmountGRT,
		payAmountGRT,
	)

	fmt.Println(json)

	return nil
}

func generateJSON(
	indexerAddress string,
	deploymentID string,
	allocationID string,
	proof string,
	allocAmountGRT uint64,
	payAmountGRT uint64,
) string {
	allocAmount := utils.ConvertToWei(allocAmountGRT).String()
	payAmount := utils.ConvertToWei(payAmountGRT).String()
	return fmt.Sprintf(jsonTemplate,
		indexerAddress,
		deploymentID,
		allocAmount,
		allocationID,
		proof,
		utils.ConvertToWei(allocAmountGRT+payAmountGRT).String(),
		payAmount,
		allocationID,
		allocationID,
	)
}

var jsonTemplate = `{"version":"1.0","chainId":"42161","createdAt":1754409348371,"meta":{"name":"Transactions Batch","description":"","txBuilderVersion":"1.18.0","createdFromSafeAddress":"","createdFromOwnerAddress":"","checksum":"0x6079306dfaa34e44e083b9218d570236d5d94bed1a6601ab765398db8fe87b8a"},"transactions":[{"to":"0x00669A4CF01450B64E8A2A20E9b1FCB71E61eF03","value":"0","data":null,"contractMethod":{"inputs":[{"internalType":"address","name":"_indexer","type":"address"},{"internalType":"bytes32","name":"_subgraphDeploymentID","type":"bytes32"},{"internalType":"uint256","name":"_tokens","type":"uint256"},{"internalType":"address","name":"_allocationID","type":"address"},{"internalType":"bytes32","name":"_metadata","type":"bytes32"},{"internalType":"bytes","name":"_proof","type":"bytes"}],"name":"allocateFrom","payable":false},"contractInputsValues":{"_indexer":"%s","_subgraphDeploymentID":"%s","_tokens":"%s","_allocationID":"%s","_metadata":"0x0000000000000000000000000000000000000000000000000000000000000000","_proof":"%s"}},{"to":"0x9623063377AD1B27544C965cCd7342f7EA7e88C7","value":"0","data":null,"contractMethod":{"inputs":[{"internalType":"address","name":"spender","type":"address"},{"internalType":"uint256","name":"amount","type":"uint256"}],"name":"approve","payable":false},"contractInputsValues":{"spender":"0x00669A4CF01450B64E8A2A20E9b1FCB71E61eF03","amount":"%s"}},{"to":"0x00669A4CF01450B64E8A2A20E9b1FCB71E61eF03","value":"0","data":null,"contractMethod":{"inputs":[{"internalType":"uint256","name":"_tokens","type":"uint256"},{"internalType":"address","name":"_allocationID","type":"address"}],"name":"collect","payable":false},"contractInputsValues":{"_tokens":"%s","_allocationID":"%s"}},{"to":"0x00669A4CF01450B64E8A2A20E9b1FCB71E61eF03","value":"0","data":null,"contractMethod":{"inputs":[{"internalType":"address","name":"_allocationID","type":"address"},{"internalType":"bytes32","name":"_poi","type":"bytes32"}],"name":"closeAllocation","payable":false},"contractInputsValues":{"_allocationID":"%s","_poi":"0x0"}}]}`
