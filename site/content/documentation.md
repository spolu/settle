+++
type = "page"
description = ""
title = "Documentation"
date = "2016-11-04T09:04:39-07:00"
hidden = "true"
+++

# Mint documentation

The Settle network connects mint servers exposing the API described in this
document. By default, mints should publicly expose this API over HTTPS on port
`2406` (HTTP on port `2407` for the QA network). Users register on mint
servers, getting Settle addresses for the domain name under which these servers
are running.

The Settle network address `stan@foo.bar` points to user `stan` on the mint
running at `foo.bar` on port `2406`.

Usernames must comply to the folollowing regular expression:
`[A-Za-z0-9\-_.]{1,256}`.

## Assets

Assets are issued by users. They are represented by the following expression:
`{ISSUER}[{CODE}.{SCALE}]`. `{ISSUER}` should be a valid Settle
address, `{CODE}` is an alphanumeric string matching the following regular
expression `[A-Z0-9\-]{1,64}`. If the asset represents a IOU for an existing
currency, the ISO-4217 3-letter code should be used. The scale, or decimal
length, is an integer between `0` and `24` representing the factor by wich the
asset native amount is scaled down when represented as a signed integer.

A few examples of valid assets are:

- **stan@foobar.com[USD.2] 320** represents **$3.20** issued by Stan.
- **info@sightglasscofee.com:AU-LAIT.0 2**: represents **2 au laits** issued by
  Sightglass Coffee. "Au laits" are not fungible and therefore have scale of 0.

## Authentication and registration

User onboarding for the Mint API is left to the discretion of the mint
implementor or administrator.

Authentication relies on the HTTP Authorization header, using "Basic"
authentication. The proposed implementation in this repository rely on the SQL
data store to lookup users and current password, registration happening outside
of the mint and resulting in records getting created/updated in the SQL
database used by the mint (see
[register](https://github.com/spolu/settle/blob/master/register) for the
registration service used by the mint maintained by the Settle developers).

## Offers

Trust in the network is expressed by offers. Offers are always represented as
asks:
- an ask on pair **bob@corewars.org[USD.2]/stan@foo.bar[USD.2]** represents an
  offer to issue and sell **bob@corewars.org[USD.2]** for a certain amount of
  **stan@foo.bar[USD.2]** at a given price.

The asset on the left of the pair is called the base asset while the asset of
the right end of the pair is called the quote asset.

Because the decimal length may differ from one asset to another, offers price
are expressed as a quotient of signed integers (always positive). The price of
ask **stan@foobar.com[USD.2]/info@sightglasscofee.com:AU-LAIT.0** can be
expressed as **320/1**.

Offer amounts are always expressed in the unit of the quote asset (the asset on
the right).

Offers to exchange assets can be created and publicized on the network. Users
create offers from their own mint and it is the responsibility of each mint to
propagate (best-effort) these offers to the relevant mints (one mint if the
quote asset is controled by another mint, none otherwise).

```
curl -XPOST https://foo.bar:2406/offers \
  -u username:password \
  -d pair=stan@foo.bar[USD.2]/info@sightglasscofee.com:AU-LAIT.0 \
  -d price=320/1 \
  -d amount=3

{
  id: "stan@foo.bar[offer_7t3sk24sdvz0a]",
  pair: "stan@foo.bar:USD.2/info@sightglasscoffee.com:AU-LAIT.0",
  price: "320/1",
  amount: 3,
  remainder: 3,
  status: "active"
}
```

This offer will potentially also be visible on mint **sightglasscofee.com** if
propagation is successful:

```
curl -XGET https://sightglasscofee.com:2406/offers/stan@foo.bar[offer_7t3sk24sdvz0a]

{
  id: "stan@foo.bar[offer_7t3sk24sdvz0a]",
  pair: "stan@foo.bar:USD.2/info@sightglasscoffee.com:AU-LAIT.0",
  price: "320/1",
  amount: 3,
  remainder: 3,
  status: "active"
}
```

### Semantics of offers

Since offers are offers to **issue** and exchange an asset against another, the
base asset of an offer must always be controlled by the owner of the offer,
ensuring that they can issue that asset.

As such, an ask **bob@corewars.org[USD.2]/stan@foo.bar[USD.2]** created by
**bob@corewars.org** represents a offer from him to issue
**bob@corewars.org[USD.2]** in exchange for **stan@foo.bar[USD.2]** at the
exchange price specified by the offer (as an example, **1/1**). As such it
expresses a trust (or credit) line from **bob@corewars.org** to
**stan@foo.bar**.

Chain of offers can therefore be leveraged to transact between users of the
settle network. The network provides a primitive to do so safely and
atomically: transactions.

## Transactions

Creating a simple transaction can let you credit a user account with a given
asset you issue or own:

```
curl -XPOST https://foo.bar:2406/transactions \
  -u username:password \
  -d pair=stan@foo.bar[USD.2]/stan@foo.bar[USD.2] \
  -d amount=500 \
  -d destination="bob@corewars.com"

{
  id: "stan@foo.bar[transaction_9iop2182cm73s]",
  pair: "stan@foo.bar[USD.2]/stan@foo.bar[USD.2]",
  operations: [{
    offer: null,
    asset: "stan@foo.bar[USD.2]",
    source: "stan@foo.bar",
    destination: "bob@corewars.com",
    amount: 500,
    status: "reserved"
  }]
}
```

To leverage chains of offers, a transaction can define a path of offers to
cross.

Assuming we have the following active offer on the network:
- **bob@corewars.org[offer_s8ka7812djnmk]**: ask
  **bob@corewars.org[USD.2]/stan@foo.bar[USD.2]** at **1/1** for
  **stan@foo.bar[USD.2] 500**

```
curl -XPOST https://foo.bar:2406/transactions \
  -u username:password \
  -d pair=stan@foo.bar[USD.2]/bob@corewars.org[USD.2] \
  -d price=1/1 \
  -d amount=200 \
  -d path[]=bob@corewars.org[offer_s8ka7812djnmk]

{
  id: "stan@foo.bar[transaction_8yhs2op9sckD2]",
  lock: "ae7b2a3ffd9c43a...",
  pair: "stan@foo.bar[USD.2]/bob@corewars.org[USD.2]",
  price: "1/1",
  amount: "200",
  operations: [{
    offer: null,
    asset: "stan@foo.bar[USD.2]",
    source: "stan@foo.bar",
    destination: "bob@corewars.org",
    amount: 200,
    status: "reserved"
  }]
}
```

Commiting this transaction to the network requires the mint **corewars.org**
and **foo.bar** to synchronize their state. They do so using the algorithm
described in the next subsections.

### Creation and reservation

When creating a transaction, a mint will retrieve the offers involved in the
transaction path, compute a transaction plan to verify its validity and commit
the transcation in **pending** `state`. It will also generate a
`lock=scrypt(secret)`, and will keep the secret private.

It then propagates the transaction to the last mint of the path.

When receiving a propagated transaction, a mint computes the transaction plan
and verify its validity, commits it as **pending** and recursively propagates
the transaction to the mint before it in the transaction plan, all the way to
the first one.

The first mint on the path, commits the transaction as **reserved**. When
committing a transaction as **reserved**, a mint makes the following
commitment on behalf of its user:

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
Note that this commitment is only possible since base asset of offers are
controlled by their owner.


The first mint is able to make that commitment without depending on any mint
(the first mint is the mint of the user creating the transaction). As it
replies with a **reserved** status, the next mint on the path is able to make
the same commitment and commit the transaction as **reserved** as well,
recursively to the last mint of the path.

Once this chain of commitments is made, the transaction is created and
**reserved** on all mints.

Implementation of the reservation algorithm can be found at:
https://github.com/spolu/settle/blob/master/mint/endpoint/create_transaction.go

### Settlement

Each of these mints are now willing to release the funds to make the
transaction happen in exchange for the secret (without talking to other mints)
as they know they can retrieve the funds from the mint before them with that
same secret (unless the mint before them is malicious and refuses to release
the funds, effectively breaking the trust expressed by the offer at this mint,
but not preventing the transaction from happening).

To settle a transaction, the originating mint release the secret to the last
mint of the path which accepts the transaction as **settled** without any
communication (at this point the transaction is effectively settled). The last
mint then asynchronously settles the transaction at the mint before it by
sending the secret there. Mints recursively and asynchronously settle the
transaction along the path.

Implementation of the settlement algorithm can be found at:
https://github.com/spolu/settle/blob/master/mint/endpoint/settle_transaction.go

### Transaction cancellation

Settle also provide a mechanism to safely cancel a transaction, in case one of
the node has a technical issue. Cancellation can be blocked indefinitely by
malicious mints.

Cancellation at a mint is possible only if the mint after it has cancelled the
transaction. Only the last mint of the path can initially trigger the
cancellation of a transaction.

If a mint does not know about a transaction (technical error), it is considered
cancelled.

Implementation of the cancellation algorithm can be found at:
https://github.com/spolu/settle/blob/master/mint/endpoint/cancel_transaction.go
