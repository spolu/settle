+++
type = "index"
title = "Settle"
description = ""
date = "2016-11-04T17:28:19-07:00"
+++
# Settle

*Decentralized trust graph for value exchange without a blockchain.*

While cryptocurrencies are maintained by distributed ledgers with no central
authority, their trust graph is still fully centralized: everyone has to trust
the currency that powers the blockahin (**BTC**, **ETH**, **XLM**...).

Settle's goal is to explore a new financial trust primitive on the Internet,
and doing so, construct a decentralized trust graph enabling (totally) free
exchange of value without relying on a blockchain.

The Settle transaction protocol allows users to safely transfer value along
pre-expressed trust paths without a shared blockchain (see [Mint
documentation](/documentation)). Very much like email, users register on
"mints", a server of their choice (possibly their own), that manage the assets
they issue as well as the trust they express between assets in the network.

Settle distributes a command line utility, `settle`, to interact with mints. It
lets you register on a publicly available mint (or login on a mint of your
choice), create assets, express trust between your assets and others', and
transact using these trust pathes.

Settle also maintains a publicly available mint **m.settle.network** on which you
can register freely.

Additional resources:

- [Settle introductory blog post](posts/settle-introduction)
- [Mint documentation](documentation)
- [Community](community)

## Installation

Install `settle` locally (under `~/.settle`):
```
curl -L https://settle.network/install | sh && export PATH=$PATH:~/.settle
```
## Getting started

Read the command-line help:
```
settle help
```

Register on a mint:
```
settle register
```

Mint (activate) your first asset (**USD** in cents):
```
settle mint USD.2
```

Trust someone for **$10**:
```
settle trust kurt@princetown.edu USD.2 1000
```

Trust at a specific rate:
```
settle trust supercharger@tesla.com KWH.2 10000 with USD.2 at 100/20
```

Pay someone **1€** (assuming a trust path exists):
```
settle pay alan@npl.co EUR.2 100
```
