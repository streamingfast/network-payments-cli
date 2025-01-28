# Network Payments CLI

The **Network Payments CLI** provides a streamlined command-line interface for interacting with The Graph Network's Payments API. Its primary goal is to simplify sending and receiving payments on the network.

---

## Installation

To install the CLI tools:

```bash
git clone git@github.com:streamingfast/network-payments-cli.git
cd network-payments-cli
go install ./cmd/...
```

This will install two commands:
- `sendpayment`
- `receivepayment`

---

## Usage

### Overview

The typical flow involves three main steps:
1. **Open an allocation**: The payment receiver opens an allocation on The Graph network using `receivepayment open-allocation` and shares the allocation ID with the sender.
2. **Send payment**: The payment sender transfers funds to the allocation using `sendpayment`.
3. **Close the allocation**: The payment receiver closes the allocation using `receivepayment close-allocation`.

---

### Key Notes
1. **Stake Ratio**:  
   As per [GIP-0051 on Exponential Rebates](https://forum.thegraph.com/t/gip-0051-exponential-query-fee-rebates-for-indexers/4162), indexers should maintain a stake ratio of at least **10:1** (stake to query fees). Ensure this ratio is adhered to as it may change based on network parameters.

2. **Query Fee Cut**:  
   The indexer's Query Fee Cut percentage affects the total amount received. For example, a 70% Query Fee Cut means the indexer keeps 70% of the payment. Ensure this percentage is correctly set.

3. **Network Tax**:  
   A **1% network tax** applies to all payments, reducing the allocation amount by 1% before it reaches the indexer.

---

### Example Usage

#### **Step 1**: Publish an IPFS Manifest

Create a manifest file (`manifest.yaml`) with the following content:

```yaml
specVersion: 0.0.5
description: "thegraph.market Payment Gateway usage"
usage:
  serviceName: substreams
  namespace: sf.substreams.rpc.v2.Stream
  network: mainnet
  uid: 001
```

Note: the contents of the file can vary based on the use case.

*IMPORTANT NOTE*: For security purposes, you should create a new unique manifest for each payment allocation.

Publish it to IPFS:

```bash
curl -X POST -F file=@"manifest.yaml" https://api.thegraph.com/ipfs/api/v0/add
```

The output will include a **Hash** for the uploaded file, such as `Qm1234...`. Use this hash as the Deployment ID for subsequent steps.

---

#### **Step 2**: Open an Allocation

The payment receiver opens an allocation for 100 GRT:

```bash
receivepayment open-allocation \
  --deployment-id Qm1234 \
  --allocation-amount 100 \
  --indexer-address 0xabcdef1234 \
  --private-key-file {path-to-private-key-file} \
  --rpc-url http://{receiver-arbitrum-rpc-node}
```

**Notes**:
- `--indexer-address`: The address of the indexer receiving the payment.
- `--private-key-file`: Path to the private key file for the indexer operator. Alternatively, set the `NETWORK_PAYMENT_PRIVATE_KEY` environment variable.

---

#### **Step 3**: Send a Payment

The payment sender sends 10 GRT to the allocation:

```bash
sendpayment \
  --allocation-id 0x1234 \
  --amount 10 \
  --deployment-id Qm1234 \
  --private-key-file {path-to-private-key-file} \
  --rpc-url http://{sender-arbitrum-rpc-node}
```

**Notes**:
- `--allocation-id`: The ID of the allocation receiving the payment.
- `--deployment-id`: The Deployment ID from the manifest.
- `--private-key-file`: Path to the private key file of the sender. Alternatively, set the `NETWORK_PAYMENT_PRIVATE_KEY` environment variable.

---

#### **Step 4**: Close the Allocation

The payment receiver closes the allocation:

```bash
receivepayment close-allocation \
  --allocation-id 0x1234 \
  --private-key-file {path-to-private-key-file} \
  --rpc-url http://{receiver-arbitrum-rpc-node}
```

**Notes**:
- Ensure the private key corresponds to the indexer receiving the payment.

---

### Environment Variables
- **`ARBITRUM_RPC_URL`**: RPC URL for interacting with the Arbitrum network. This can replace the `--rpc-url` flag.
- **`NETWORK_PAYMENT_PRIVATE_KEY`**: Hex-encoded private key for signing transactions. This can replace the `--private-key-file` flag.

---

### Troubleshooting

1. **Common Errors**:
    - Incorrect allocation ID: Verify the allocation ID provided by the receiver.
    - RPC URL issues: Ensure the correct Arbitrum RPC URL is configured.
    - Private key errors: Confirm the private key file path or environment variable is set correctly.

2. **Debugging**:
   Use verbose logging by appending `--log-level debug` to commands.
