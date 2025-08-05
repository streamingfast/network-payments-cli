# Network Payments CLI

The **Network Payments CLI** provides a streamlined command-line interface for interacting with The Graph Network's Payments API. Its primary goal is to simplify sending and receiving payments on the network.

---

## Installation

To install the CLI tools:

```bash
git clone git@github.com:streamingfast/network-payments-cli.git
cd network-payments-cli
go install ./cmd/paygrt
```

This will install the `paygrt` command

---

## Usage

You should now use: `paygrt 20 2 0x35917C0eB91d2E21BEF40940D028940484230c06 > multitransactions.json` where:

* `20` allocation amount in GRT
* `2` is the payment amount in GRT
* `0x35917C0eB91d2E21BEF40940D028940484230c06` is the receiver's address (usually the our indexer)
* `multitransactions.json` is the file that you will give to your Gnosis SAFE to run the multiple transactions bundled in one.
