package utils

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/streamingfast/eth-go"
)

func GenerateAllocationIDAndProof(address string) ([]byte, []byte, error) { //returns allocationID, proof, err
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
