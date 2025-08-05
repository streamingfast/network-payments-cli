package utils

import (
	"context"
	"fmt"
	"math/big"

	"github.com/streamingfast/eth-go"
	ethrpc "github.com/streamingfast/eth-go/rpc"
	"github.com/streamingfast/eth-go/signer/native"
	"go.uber.org/zap"
)

func ApproveCall(ctx context.Context, to string, from string, cli *ethrpc.Client, addr string, amt uint64, gasPrice int64) (string, error) {
	methodDef, err := eth.NewMethodDef("approve(address,uint256)")
	if err != nil {
		return "", err
	}

	address := eth.MustNewAddress(addr)
	amount := ConvertToWei(amt)

	methodCall := methodDef.NewCall()
	methodCall.AppendArg(address)
	methodCall.AppendArg(amount)

	data, err := methodCall.Encode()
	if err != nil {
		return "", err
	}

	signer, err := native.NewPrivateKeySigner(zap.NewNop(), MustGetChainID(ctx, cli), MustGetPrivateKey(ctx))
	if err != nil {
		return "", fmt.Errorf("unable to create signer: %w", err)
	}

	var gasPriceBigInt *big.Int
	if gasPrice == 0 {
		gasPriceBigInt = MustGetGasPrice(ctx, cli)
	} else {
		gasPriceBigInt = big.NewInt(gasPrice)
	}

	signedTx, err := signer.SignTransaction(
		MustGetAccountNonce(ctx, eth.MustNewAddress(from), cli),
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

	receipt, err := FetchReceiptWithProgress(ctx, cli, eth.MustNewHash(resp))
	if err != nil {
		return "", err
	}

	if receipt == nil {
		return "", fmt.Errorf("failed to approve. receipt is nil")
	}

	return resp, nil
}
