package utils

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/multiformats/go-multihash"

	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/eth-go/rpc"
)

func getAccountNonce(ctx context.Context, fromAddr eth.Address, cli *rpc.Client) (uint64, error) {
	nonce, err := cli.Nonce(ctx, fromAddr, rpc.LatestBlock)

	if err != nil {
		return 0, fmt.Errorf("unable to retrieve nonce for account %q: %w", fromAddr, err)
	}

	return nonce, nil
}

func MustGetAccountNonce(ctx context.Context, fromAddr eth.Address, cli *rpc.Client) uint64 {
	nonce, err := getAccountNonce(ctx, fromAddr, cli)

	if err != nil {
		panic(err)
	}

	return nonce

}

func getGasPrice(ctx context.Context, cli *rpc.Client) (*big.Int, error) {
	gasPrice, err := cli.GasPrice(ctx)

	if err != nil {
		return nil, fmt.Errorf("unable to retrieve gas price: %w", err)
	}

	return gasPrice, nil
}

func MustGetGasPrice(ctx context.Context, cli *rpc.Client) *big.Int {
	gasPrice, err := getGasPrice(ctx, cli)

	if err != nil {
		panic(err)
	}

	return gasPrice
}

func getChainID(ctx context.Context, cli *rpc.Client) (*big.Int, error) {
	chainId, err := cli.ChainID(ctx)

	if err != nil {
		return nil, fmt.Errorf("unable to retrieve chain id: %w", err)
	}

	return chainId, nil
}

func MustGetChainID(ctx context.Context, cli *rpc.Client) *big.Int {
	chainID, err := getChainID(ctx, cli)

	if err != nil {
		panic(err)
	}

	return chainID
}

func FetchReceiptWithProgress(ctx context.Context, rpcClient *rpc.Client, trxHash eth.Hash) (*rpc.TransactionReceipt, error) {
	backoff := 500 * time.Millisecond
	maxWaitTime := 5 * time.Minute

	startTime := time.Now()
	for {
		receipt, err := rpcClient.TransactionReceipt(ctx, trxHash)
		if err != nil {
			return nil, err
		}

		if receipt == nil {
			if time.Since(startTime) > maxWaitTime {
				fmt.Printf("Unable to find transaction receipt after %s, stopping here\n", maxWaitTime)
				return nil, nil
			}

			time.Sleep(backoff)
			if backoff < 12*time.Second {
				backoff = backoff * 2
			}

			continue
		}

		return receipt, nil
	}
}

// ConvertToWei converts a token amount to its smallest unit (wei).
func ConvertToWei(amount uint64) *big.Int {
	tokenAmount := big.NewInt(int64(amount))
	multiplier := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil)
	tokenAmount.Mul(tokenAmount, multiplier)
	return tokenAmount
}

func ConvertIPFSHashToByteString(hash string) ([]byte, error) {
	// Decode the base58 encoded hash
	decoded, err := multihash.FromB58String(hash)
	if err != nil {
		return nil, fmt.Errorf("decoding base58 string: %w", err)
	}

	if len(decoded) == 34 {
		// Remove the first two bytes (multicodec and multihash length)
		decoded = decoded[2:]
	}

	return decoded, nil
}
