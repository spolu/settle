+++
type = "index"
title = "Settle"
description = ""
date = "2016-11-04T17:28:19-07:00"
+++
# Settle

*Decentralized trust graph for value exchange without a blockchain.*

While cryptocurrencies are maintained by distributed ledgers with no central
authority, their trust model and graph is still fully centralized: everyone has
to trust the currency.

Settle's goal is to explore a new financial trust primitive on the Internet,
and doing so, construct a decentralized trust graph enabling (totally) free
exchange of value without relying on a blockchain.

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

Mint your first asset (`USD` in cents):
```
settle mint USD.2
```

Trust your friend for $10:
```
settle trust friend@m.settle.network USD.2 1000
```

Pay someone $1 (assuming a trust path exists):
```
settle pay someone@m.settle.network USD.2 100
```
