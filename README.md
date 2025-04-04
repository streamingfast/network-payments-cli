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

#### **Step 1**: Open an Allocation

The payment receiver opens an allocation for 100 GRT:

```bash
receivepayment open-allocation \
  --allocation-amount {amount of GRT to allocate} \
  --indexer-address {address of the indexer receiving the payment} \
  --private-key-file {path-to-private-key-file} \
  --rpc-url http://{arbitrum-rpc-endpoint}
```

This will return the `deployment ID` and `allocation ID`, which should be shared with the payment sender.

---

#### **Step 2**: Send a Payment

The payment sender sends 10 GRT to the allocation:

```bash
sendpayment \
  --allocation-id {allocation-id from step 1} \
  --deployment-id {deploymet-id from step 1} \
  --private-key-file {path-to-private-key-file} \
  --rpc-url http://{arbitrum-rpc-endpoint} \
  --amount {amount of GRT to send}
```
---

#### **Step 3**: Close the Allocation

The payment receiver closes the allocation:

```bash
receivepayment close-allocation \
  --allocation-id {{allocation-id from step 1}} \
  --deployment-id {deploymet-id from step 1} \
  --private-key-file {path-to-private-key-file} \
  --rpc-url http://{arbitrum-rpc-endpoint}
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
