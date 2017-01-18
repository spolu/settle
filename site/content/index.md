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
the currency that powers the blockchain (**BTC**, **ETH**, **XLM**...).

Settle's goal is to explore a new financial trust primitive, and doing so,
construct a decentralized trust graph enabling (totally) free exchange of value
without relying on a blockchain, making it particularly adapted to
machine-to-machine transactions.

The Settle transaction protocol allows users to safely transfer value along
pre-expressed trust paths (see [Mint documentation](/documentation)). Similarly
to email, users register on a "mint": a server of their choice (possibly their
own) that manage the assets they issue as well as the trust they express
towards assets in the network.

Settle distributes a command line utility, `settle`, to interact with mints. It
lets you register on a publicly available mint (or login on a mint of your
choice), create assets, express trust between your assets and others', and
transact using these trust paths.

Settle also maintains a publicly available mint **m.settle.network** on which
you can register freely.

Additional resources:

- [Settle introductory blog post](posts/intro)
- [Mint documentation](documentation)

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

Mint (activate) your first asset (**USD** with a scale of 2, i.e expressed in cents):
```
settle mint USD.2
```
(Assets are expressed as `{CODE}.{SCALE}` where the code is an alphanumeric
string of your choice and the scale, or decimal length, is an integer
representing the factor by which the asset native amount is scaled down when
represented as a signed integer).

Trust someone for **$10**:
```
settle trust kurt@princetown.edu USD.2 1000
```

Trust at a specific price:
```
settle trust supercharger@tesla.com KWH.2 10000 with USD.2 at 20/100
```
(Prices are expressed as `base_asset/quote_asset` where the base asset is your
own asset (here **you@yourdomain.com[USD.2]**) and the quote asset is the asset
you trust (here **supercharger@tesla.com[KWH.2]**).


Pay someone **1€** (assuming a trust path exists):
```
settle pay alan@npl.co EUR.2 100
```
