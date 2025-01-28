package utils

import (
	"context"
	"github.com/streamingfast/eth-go"
	ethrpc "github.com/streamingfast/eth-go/rpc"
	"strings"
)

const GRTTokenContractAddress = "0x9623063377ad1b27544c965ccd7342f7ea7e88c7"
const StakingContractAddress = "0x00669a4cf01450b64e8a2a20e9b1fcb71e61ef03"
const L2CurationContractAddress = "0x22d78fb4bc72e191C765807f8891B5e1785C8014"

func IsCuratedCall(ctx context.Context, cli *ethrpc.Client, deploymentId string) (bool, error) {
	methodDef, err := eth.NewMethodDef("isCurated(bytes32)")
	if err != nil {
		return false, err
	}

	deployment, err := ConvertIPFSHashToByteString(deploymentId)
	if err != nil {
		return false, err
	}

	methodCall := methodDef.NewCall()
	methodCall.AppendArg(deployment)

	data, err := methodCall.Encode()
	if err != nil {
		return false, err
	}

	resp, err := cli.Call(ctx, ethrpc.CallParams{
		To:   eth.MustNewAddress(L2CurationContractAddress),
		Data: data,
	})
	if err != nil {
		return false, err
	}

	respToBool := func(resp string) bool {
		if strings.HasSuffix(resp, "1") {
			return true
		}
		return false
	}

	return respToBool(resp), nil
}
