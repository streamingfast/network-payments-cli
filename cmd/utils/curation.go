package utils

import (
	"context"
	"github.com/streamingfast/eth-go"
	ethrpc "github.com/streamingfast/eth-go/rpc"
	"strings"
)

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
