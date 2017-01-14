+++
type = "post"
title = "Decentralized trust graph for online value exchange without a blockchain"
description = "Introductory post about Settle, how it enables value exchange online without a blockchainm, and why it may be useful."
date = "2017-01-14T10:31:00-07:00"
+++

# Settle

I'm happy to release Settle, a personal research project I've been working on
over the past 8 months!

Settle's goal is to explore a new financial trust primitive on the Internet,
and doing so, construct a decentralized trust graph enabling (totally) free
exchange of value without relying on a blockchain.

If you really can't wait to play with it, you can install it locally (under
`~/.settle`) with the following command:
```
curl -L https://settle.network/install | sh && export PATH=$PATH:~/.settle
```

... and read the command-line help with:
```
settle help
```

All of the source code is also availabe at: https://github.com/spolu/settle

*Value-exchange for humans and machines alike*

Settle was born out of the idea of experimenting with what would be the
simplest way to let machines and humans issue IOUs ("I owe you") and exchange
them safely. Early on, I tinkered with tooling around Bitcoin (colored
coins), Stellar, and Ethereum (simple asset contracts), but run into the exact
same issues on each of these systems:

- Even if you exchange IOUs, you have to pay transaction fees to maintain the
blockchain backing each of these systems. So you really can't openly onboard
users on whichever tool you create as you have to credit their account with
some amount of BTC, XLM or ETH to get them started. This is a pretty tough
barrier to entry, and as a direct consequence, you can't programmatically
onboard machines at scale without a substantial costs.
- micro-payments particularly relevant to machine to machine transactions are
alos most of the time impractical because of these fees[0]

This led me to the obvious realization that:

*Currencies operate on a centralized trust graph.*

This sentence is almost tautological but let's consider both fiat and crypto
currencies:

Fiat currencies value is by definition backed by the government that issued it.
Hence, by construction, the trust graph powering exchanges based on fiat
currencies is centralized (it's no surprise that we generally call the issuer
of a fiat currency a central bank). Two individuals transacting in USD must
both trust the US government backing of dollars.

Conversely, cryptocurrencies such as Bitcoin or Ethereum have decentralized the
process of issuing and managing a currency. But while the operations of such
currencies, based on blockchains, have been fully decentralized, the trust
graph of these cryptocurrencies have remained entirely centralized. Everyone
need to trust Bitcoin to transact in Bitcoin, and everyone needs to trust
Ethereum to transact in Ethereum or assets issued on the Ethereum blockchain.
The centralized nature of the trust involved in these cryptocurrencies being
actually at the core of how these currencies operate, as the only viable way so
far to properly incentivize a proof-of-work system.

From that realization, I embarked on the task of exploring whether it would be
possible to instead create a decentralized trust graph between user-issued
assets and leverage it to safely exchange value without a central authority or
shared blockchain to maintain.

After 6 months of work on that idea. I believe it's possible. 

*The Settle network is operated by mints.*

Since you don't have a shared blockchain, the nodes of your network have to be
online. So, very much like email, users register on "mints", a server of their
choice (possibly their own), that manage the assets they issue as well as the
trust they express between assets in the network.

You can freely register with the `settle` command-line on the mint I opened
publicly at `m.settle.network`:
```
settle register
```

And you can also directly log in the mint of your choice with:
```
settle login
```

More precisely, mints are in charge of maintaining the following for their
users:

- the list of assets issued by its users
- balances associated with these assets
- an authoritative list of offers put on the network by its users
- an indicative list of offers involving the assets it is authoritative for
  (order book)

One nice property of this, is that users don't have to manage a private key,
they just have credentials with their mint that can be rolled or retrieved
easily. But they do have to trust their mint.

*Value is exchanged by crossing offers across mints*

Users of Settle issue assets (or IOUs) of the form
**kurt@princetown.edu[USD.2]**; an IOU from Kurt on a mint operated by
Princetown, for **USD** dollars expressed as cents. Each users can freely issue
their own assets. A user's mint is authoritative for the assets they issue, and
it maintains and manages balances (of other users) for these assets. Hence,
issuing or transferring IOUs to others simply involves posting instruction to
one's own mint. As an example **kurt@princetown.edu** can simply issue a
certain amount of **kurt@princetown.edu[USD.2]** to **alan@npl.co.uk** by
posting an instruction to the mint at **princetown.edu**. Since the mint is
authoritative for that asset, no synchronization is required with the mint at
**npl.co.uk**.

Let's assume that both **alan@npl.co.uk[USD.2]** and
**albert@princetown.edu[USD.2]** were activated by their respective users on
their respective mints.
```
settle mint USD.2    # activates USD.2
```

At the mint level, users express trust in the network by posting offers on
their mints to issue and exchange assets they control against assets controlled
by other users they trust. Maybe Alan trusts Kurt for up to $200. Alan will
represent that trust by posting on his mint an offer on pair
**alan@npl.co.uk[USD.2]/kurt@princetown.edu[USD.2]** for **20000** at price
**1/1**.

```
settle trust kurt@princetown.edu USD.2 20000    # alan runs this
```

Similarly, let's assume that Kurt trusts Albert up to $100 and has an
outstanding offer on pair
**kurt@princetown.edu[USD.2]/albert@princetown.edu[USD.2]** for **10000** at
price **1/1**. These two offers will allow Albert to transact with Alan without
requiring Alan to trust Albert directly. If Alan sells a machine part for $10,
by crossing the two offers, Albert can buy that part from Alan. He first issues
and exchanges **albert@princetown.edu[USD.2] 1000** against
**kurt@princetown.edu[USD.2] 1000** using the offer from Kurt, and then exchange
the acquired **kurt@princetown.edu[USD.2] 1000** against **alan@npl.co.uk[USD.2]
1000** using Alan's offer that he can finally credit back to Alan in exchange for
the part.

```
settle pay alan@npl.co.uk USD.2 1000    # albert runs this
```

After this operation is complete, Alan will have a balance of
**kurt@princetown.edu[USD.2] 1000** on the mint at **princetown.edu**, Alan's
offer will be valid for a remaining **19000** and Kurt's offer for **9000**.

The mint protocol provides a mechanism called transactions (backing the `pay`
command above) to cross a chain of offers atomically and safely[1][2]. A
transaction involves reserving funds along that chain and then committing the
balance operations on each mints that participate once the offers are secured.
Transactions, if successful, are instantaneously confirmed and fee-less.

Of course users can also specify trust with a discount (because of possible
risk) or an exchange rate, as an example:

```
setlle trust alan@npl.co.uk GBP.2 120 with USD.2 at 122/100
```
/The settle.network mint/

With this post, I'm releasing an open-source implementation for mints, as well
as the command-line tool `settle`. That you can install locally with:
```
curl -L https://settle.network/install | sh && export PATH=$PATH:~/.settle
```

I'm also opening a mint at `m.settle.network` to let people issue and exchange
assets as well as a guide to setup your own mint:

- [Source code](https://github.com/spolu/settle)
- [Documentation](/documentation)
- [Guide: Setting up a mint](/posts/guide-setting-up-a-mint/)
- [Community](/community)

/Mint your assets!/

I'd like to write a few more posts exploring how Settle could be used in a
couple of interesting situations related to self-driving car networks[3] and
e-commerce[4], but the first strong use-case for Settle is very likely to come
from somewhere else, you.

So I definitely invite you to play with the `settle` command line. When you
mint assets, feel free to ping me on IRC (spolu in #settle on FreeNode), I'd be
more than happy to trust you for a few cents and transact with you to test the
system.

Setting up your own mint is also definitely a great way to learn more and help
the community grow. If you setup a mint, let's add it to the `settle register`
command!

And if you really want to get your hands dirty, building an app or a
gateway[5], are definitely things that I would love to collaborate on in the
future.

-stan

[0] Lightning networks elegantly circumvent the problem of per transaction fees
but don't help with the "onboarding" problem.

[1] see the [Mint documentation](/documentation). In particular, the protocol
is safe in the sense that there is no double-spend but users can loose money if
they trust malicious users.

[2] skipping a few steps and notions, but for reference, the exact commitment
(ensuring safety) that a mint is doing when accepting to participate in a
transaction is the following:
```
Mint at hop h (position along the offer path), commits to:
  - settle (irrevocably issue/credit the funds) if:
    - it has not yet canceled the transaction.
    - it is presented with the lock secret.
    - node at h-1 has made the same commitment.
  - cancel the transaction if:
    - it has not yet settled the transaction.
    - node at h+1 has canceled.
```

[3] The main idea here would be to configure Teslas to trust the Tesla Network
solely **modelx_AX7GD@telsa.com[USD.2]/network@tesla.com[USD.2] 1/1** and have
the Tesla Network trust various entities with a discount
**network@telsa.com[USD.2]/settle@sfgov.org[USD.2] 100/110** such that Tesla
gets a cut of each transactions without necessarily enforcing the use of a
particular currency to pay your trip on a Tesla. Conversely car-owners could
trust **network@telsa.com[USD.2]** to use these credits at supercharger
stations, enabling Telsas on the Tesla Network to pay for their recharge at
home owner's garages.

[4] E-commerce website could use Settle to issue store credit that is easily
usable at other place. This could potentially solve the problem of unlinked
refunds in e-commerce while ensuring merchants that these credits would, by
construction, be used at their store.

[5] Lets users deposit fiat currency in exchange for trust on the network, or
get back fiat currency by paying the gateway on the network.


