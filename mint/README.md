# Mints

The Settle network connects mint servers exposing the API described in this
document. By default, mints should publicly expose this API over HTTPS on port
`2406`. Users register on mint servers, getting Settle addresses for the domain
under which these servers are running.

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
asset native amount is scaled down when represented as a 128-bit signed
integer.

A few examples of valid assets are:
- **stan@foobar.com[USD.2] 320** represents **$3.20** issued by Stan.
- **info@sightglasscofee.com:AU-LAIT.0 2**: represents **2 au laits** issued by
  Sightglass Coffee. "Au laits" are not fungible and therefore have scale of 0.

## Mint API

### Authentication and registration

User onboarding for the Mint API is left to the discretion of the mint
implementor or administrator.

Authentication relies on the HTTP Authorization header, using "Basic"
authentication. The proposed implementation in this repository rely on the SQL
data store to lookup users and current password, registration happening outside
of the mint and resulting in records getting created/updated in the SQL
database used by the mint (see [register](/register) for the registration
service used by the mint maintained by the Settle developers).

### Create an offer

Trust in the network is expressed by offers. Offers are always represented as
asks:
- an ask on pair **bob@corewars.org[USD.2]/stan@foo.bar[USD.2]** represents an
  offer to sell **bob@corewars.org[USD.2]** for a certain amount of
  **stan@foo.bar[USD.2]** at a given price.

The asset on the left of the pair is called the base asset while the asset of
the right end of the pair is called the quote asset.

Because the decimal length may differ from one asset to another, offers price
are expressed as a quotient of 128-bit signed integers (always positive). The
price of ask **stan@foobar.com[USD.2]/info@sightglasscofee.com:AU-LAIT.0**
can be expressed as **320/1**.

Offers amount are always expressed in the unit of the quote asset (the asset on
the right).

Offers to exchange assets can be created and publicized on the network. Users
create offers from their own mint and it is the responsibility of each mint to
correctly propagate these offers to the relevant mints (one other mint if one
of the asset in the pair is issued by one of the mint's user, two other mints
otherwise).

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
  remaining_amount: 3,
  status: "active"
}
```

This offer will also be visible on mint **sightglasscofee.com** as an ask:

```
curl -XGET https://sightglasscofee.com:2406/offers/stan@foo.bar[offer_7t3sk24sdvz0a]

{
  id: "stan@foo.bar[offer_7t3sk24sdvz0a]",
  pair: "stan@foo.bar:USD.2/info@sightglasscoffee.com:AU-LAIT.0",
  price: "320/1",
  amount: 3,
  remaining_amount: 3,
  status: "active"
}
```

#### Semantics of offers

Users can only create asks for which they own the base asset and mints
automatically issue base assets to satisfy an offer. 

As such an ask **bob@corewars.org[USD.2]/stan@foo.bar[USD.2]** created by
**bob@corewars.org** represents a commitment from him to issue
**bob@corewars.org[USD.2]** in exchange for **stan@foo.bar[USD.2]** at the
exchange price specified by the offer (as an example, **1/1**). As such it
expresses a trust (or credit) line from **bob@corewars.org** to
**stan@foo.bar**.

Chain of offers can therefore be leveraged to transact between users of the
settle network. The network provides a primitive to do so safely and
atomically: transactions.

### Create a transaction

Creating a simple transaction can let you credit a user account with a given
asset you own:

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

Commiting this transaction to the network requires te mint **corewars.org** and
**foo.bar** to syncrhonize their state. They do so using the algorithm
described in the next subsections.

#### Transaction creation and reservation

```
endpoint: POST /transactions

CREATE_TRANSACTION [canonical]
  compute transaction plan
  generate lock=scrypt(secret)
  create transaction tx in pending state
  RESERVE_TRANSACTION(tx, -1) [with retry]
  IF success
    RETURN success
  ELSE
    RETRUN success
```

```
endpoint: POST /transactions/:transaction/reserve

RESERVE_TRANSACTION(tx, node)
  IF transaction unknown:
    retrieve transaction from canonical node
    create transaction in pending state
  Compute transaction plan 
  IF node == 0 [canonical]
    create reserved operation [idempotent]
    RETURN success with reserved operation
  RESERVE_TRANSACTION(tx, node-1)
  IF failed
    RETURN failure
  ELSE
    check returned reserved operation
    create reserved offer crossing [idempotent]
    create reserved operation [idempotent]
    RETURN success with reseved operation and crossing
```

#### Transaction settlement


```
endpoint: POST /transactions/:transaction/settle

SETTLE_TRANSACTION(tx) [canonical]
  Retrieve secret
  SETTLE_TRANSACTION(tx, -1, secret)

SETTLE_TRANSACTION(tx, node, secret)
  IF lock=scrypt(lock)
    settle operations and crossings that are reserved for the transaction
    mark the transaction as settled
    SETTLE_TRANSACTION(tx, node-1, secret)
    RETURN success
  ELSE
    RETURN failure
```
