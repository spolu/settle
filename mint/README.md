# Mints

The Settle network connects mint servers exposing the API described in this
document. By default, mints should publicly expose this API over HTTPS on port
`2406`. Users register on mint servers, getting Settle addresses for the domain
under which these servers are running.

The Settle network address `stan@foo.bar` points to user `stan` on the mint
running at `foo.bar` on port `2406`.

Usernames must comply to the folollowing regular expression:
`[A-Za-z0-9\-_.]{1,256}`. The `+` is admissible in Settle network addresses and
will be ignored similarly to email addresses.

## Assets

Assets are issued by users. They are represented by the following pattern:
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
  Sightglass Coffee. "Au laits" are not fungible.

## API

### Livemode

All requests should specify a `Livemode` header. If `true` the request is
considered live, otherwise it is a test request. The test environment behaves
similarly to the live one with the only added convention that assets are not
due in testmode, and consequently have no value whatsoever.

### User API

#### Onboarding and authentication

User onboarding for the User API is left to the discretion of the mint
implementor or administrator.

Authentication relies on the HTTP Authorization header, using "Basic"
authentication. The proposed implementation in this repository rely on the SQL
storage to lookup users and current password (onboarding happening outside of
the mint and resulting in records getting created/updated in the SQL database
used by the mint).

#### Create an offer

An offer on a given asset pair, can be either a bid or an ask:
- a bid on pair **bob@corewars.org[USD.2]/alice@rocket.science[USD.2]**
  represents an offer to buy **bob@corewars.org[USD.2]** for a certain amount of
  **alice@rocket.science[USD.2]** at a given price.
- an ask on pair **bob@corewars.org[USD.2]/stan@foo.bar[USD.2]** represents an
  offer to sell **bob@corewars.org[USD.2]** for a certain amount of
  **stan@foo.bar[USD.2]** at a given price.

The asset on the left of the pair is called the base asset while the asset of
the right end of the pair is called the quote asset.

Since there is no "central" or "shared" currency, it's important to note that a
bid on **bob@corewars.org[USD.2]/stan@foo.bar[USD.2]** is strictly equivalent to
an ask on **stan@foo.bar[USD.2]/bob@corewars.org[USD.2]**.

Because the decimal length may differ from one asset to another, offers price
are expressed as a quotient of 128-bit signed integers (always positive). The
price of a bid on **stan@foobar.com[USD.2]/info@sightglasscofee.com:AU-LAIT.0**
can be expressed as **320/1**. Which is equivalent to an ask on
**info@sightglasscofee.com:AU-LAIT.0/stan@foobar.com[USD.2]** at price **1/320**.

Offers amount are always expressed in the unit of the quote asset (the asset on
the right).

Each mint maintains a list of bids and asks for the assets their users issue.
The bids on **stan@foo.bar[USD.2]** are all the bids on asset pairs of the form
**stan@foo.bar[USD.2]/...**. The asks on **stan@foo.bar[USD.2]** are all the asks
on asset pairs of the form **stan@foo.bar[USD.2]/...**.

Offers to exchange assets can be created and publicized on the network. Users
create offers on asset pairs from their own mint and it is the responsibility
of each mint to correctly propagate these offers to the relevant mints (one
other mint if one of the asset in the pair is issued by one of the mint's user,
two other mints otherwise).

```
curl -XPOST https://foo.bar:2406/offers \
  -H livemode: true \
  -u username:password \
  -d type=bid \
  -d pair=stan@foo.bar[USD.2]/info@sightglasscofee.com:AU-LAIT.0 \
  -d price=320/1 \
  -d amount=3

{
  id: "stan@foo.bar[offer_7t3sk24sdvz0a]",
  livemode: true,
  pair: "stan@foo.bar:USD.2/info@sightglasscoffee.com:AU-LAIT.0",
  type: "bid",
  price: "320/1",
  amount: 3,
  remaining_amount: 3,
  status: "active",
  transactions: []
}
```

This offer will also be visible on mint **sightglasscofee.com** as an ask:

```
curl -XGET https://sightglasscofee.com:2406/offers/stan@foo.bar[offer_7t3sk24sdvz0a]

{
  id: "stan@foo.bar[offer_7t3sk24sdvz0a]",
  livemode: true,
  pair: "info@sightglasscoffee.com:AU-LAIT.0/stan@foo.bar[USD.2]",
  type: "ask",
  price: "1/320",
  amount: 960,
  remaining_amount: 960,
  status: "active",
  transactions: []
}
```

#### Create a transaction

Creating a simple transaction let you credit a user account with a given asset.

```
curl -XPOST https://foo.bar:2406/transactions \
  -H livemode: true \
  -u username:password \
  -d pair=stan@foo.bar[USD.2]/stan@foo.bar[USD.2] \
  -d amount=500 \
  -d destination="bob@corewars.com"

{
  id: "stan@foo.bar[transaction_9iop2182cm73s]",
  livemode: true,
  pair: "stan@foo.bar[USD.2]/stan@foo.bar[USD.2]",
  operations: [{
    offer: null,
    asset: "stan@foo.bar[USD.2]",
    source: "stan@foo.bar",
    destination: "bob@corewars.com",
    amount: 500,
    status: "settled"
  }]
}
```

A transaction can define a path to cross an offer. If the destination
(`destination` parameter) is ommitted, the initiator of the transaction is
assumed:

Only asks can be crossed by transactions (but given a bid, it is easy to find
the equivalent ask on the network).

Assuming we have the following active offer on the network:
- **bob@corewars.org[offer_s8ka7812djnmk]**: ask
  **bob@corewars.org[USD.2]/stan@foo.bar[USD.2]** at **100/100** for **500** for
  **stan@foo.bar[USD.2]**
(this offer can be found as a an ask on **corewars.org** and as a bid on
**foo.bar**).

```
curl -XPOST https://foo.bar:2406/transactions \
  -H livemode: true \
  -u username:password \
  -d pair=stan@foo.bar[USD.2]/bob@corewars.org[USD.2] \
  -d price=100/100 \
  -d amount=500 \
  -d path[]=bob@corewars.org[offer_s8ka7812djnmk]

{
  id: "stan@foo.bar[transaction_8yhs2op9sckD2]",
  livemode: true,
  lock: "ae7b2a3ffd9c43a...",
  pair: "stan@foo.bar[USD.2]/bob@corewars.org[USD.2]",
  operations: [{
    offer: null,
    asset: "stan@foo.bar[USD.2]",
    source: "stan@foo.bar",
    destination: "bob@corewars.org",
    amount: 500,
    status: "reserved"
  }, {
    offer: "bob@corewars.org[offer_s8ka7812djnmk]",
    asset: "bob@corewars.org[USD.2]",
    source: "bob@corewars.org",
    destination: "stan@foo.bar",
    amount: 500,
    status: "reserved"
  }]
}
```

Crossing an offer involves two operations, one to credit the base asset of the
offer to the seller and one to credit the quote asset to the initiator of the
transaction.

A transaction can specify a longer path to send money by crossing a path of
offers.

Let's assume we have the following two active offers on the network:
- **bob@corewars.org[offer_s8ka7812djnmk]**: ask
  **bob@corewars.org[USD.2]/stan@foo.bar[USD.2]** at **100/100** for **500**.
- **alice@rocket.science[offer_9iop2182cm73s]**: ask
  **alice@rocket.science[USD.2]/bob@corewars.org[USD.2]** at **100/100** for
  **3500**.

Creating the following transaction will cross these two offers for **500**,
effectively crediting **500 alice@rocket.science[USD.2]** to stan in exchange
for **500 stan@foo.bar[USD.2]**.

```
curl -XPOST https://foo.bar:2406/transactions \
  -H livemode: true \
  -u username:password \
  -d pair=stan@foo.bar[USD.2]/alice@rocket.science[USD.2]
  -d price=100/100 \
  -d amount=500 \
  -d path[]=bob@corewars.org[offer_s8ka7812djnmk] \
  -d path[]=alice@rocket.science[offer_9iop2182cm73s]

{
  id: "stan@foo.bar[transaction_9iop2182cm73s]",
  livemode: true,
  lock: "ae7b2a3ffd9c43a...",
  pair: "stan@foo.bar[USD.2]/alice@rocket.science[USD.2]",
  operations: [{
    offer: null,
    asset: "stan@foo.bar[USD.2]",
    source: "stan@foo.bar",
    destination: "bob@corewars.org",
    amount: 500,
    status: "reserved"
  }, {
    offer: "bob@corewars.org[offer_s8ka7812djnmk]",
    asset: "bob@corewars.org[USD.2]",
    source: "bob@corewars.org",
    destination: "alice@rocket.science",
    amount: 500,
    status: "reserved"
  }, {
    offer: "alice@rocket.science[offer_9iop2182cm73s]",
    asset: "alice@rocket.science[USD.2]",
    source: "alice@rocket.science",
    destination: "stan@foo.bar",
    amount: 500,
    status: "reserved"
  }]
}
```

### Public API

#### Retrieve order books

Offers can be retrieved publicly from the following endpoints. Retrieving both
legs lets your reconstruct the full order book for an asset pair:

```
curl -XGET https://foo.bar:2406/offers?
  pair=stan@foo.bar[USD.2]/bob@corewars.org[USD.2]
  type=ask \
  -H livemode: true

[{
  id: "stan@foo.bar[offer_7t3sk24sdvz0a]",
  livemode: true,
  pair: "stan@foo.bar[USD.2]/bob@corewars.org[USD.2]",
  type: "ask",
  price: "100/100",
  amount: 5000,
  remaining_amount: 5000,
  status: "active",
  transactions: []
}]

curl -XGET https://foo.bar:2406/offers?
  pair=stan@foo.bar[USD.2]/bob@corewars.org[USD.2]
  type=bid \
  -H livemode: true

[{
  ...
 }, {
  ...
}]
```

#### Propagate transactions

To ensure funds are reserved along the path of a transaction, mints communicate
with one another to create the underlying operations.

Taking the example above, as transaction **transaction_9iop2182cm73s** is being
created on **foo.bar**, the mint checks the validity of the path and creates
the list of operations required to fulfill the transaction, marking the first
operation (not attached to any offer) as reserved. Funds associated with a
reserved operations are reserved for that operation until it is either
**settled** or **canceled**.

The created transaction object is made available on **foo.bar** with the
following state:

```
curl -XGET https://foo.bar:2406/transactions[transaction_9iop2182cm73s] \
  -H livemode: true

{
  id: "stan@foo.bar[transaction_9iop2182cm73s]",
  livemode: true,
  lock: "ae7b2a3ffd9c43a...",
  pair: "stan@foo.bar[USD.2]/alice@rocket.science[USD.2]",
  operations: [{
    offer: null,
    asset: "stan@foo.bar[USD.2]",
    source: "stan@foo.bar",
    destination: "bob@corewars.org",
    amount: 500,
    status: "reserved"
  }, {
    offer: "bob@corewars.org[offer_s8ka7812djnmk]",
    asset: "bob@corewars.org[USD.2]",
    source: "bob@corewars.org",
    destination: "alice@rocket.science",
    amount: 500,
    status: "pending"
  }, {
    offer: "alice@rocket.science[offer_9iop2182cm73s]",
    asset: "alice@rocket.science[USD.2]",
    source: "alice@rocket.science",
    destination: "stan@foo.bar",
    amount: 500,
    status: "pending"
  }]
}
```

Despite making the transaction available on its API, it does not reply to the
transaction creation call yet and instead propagates synchronously the
transaction to the next mint in the transaction path by specifying the `id` of
the newly generated transaction:

```
curl -XPOST https://corewars.org:2406/transactions \
  -H livemode: true \
  -d id=stan@foo.bar[transaction_9iop2182cm73s]
```

By receiving this request **corewars.org** retrieves the transaction from
**foo.bar** effectively checking that **foo.bar** has reserved the funds and
that the operations are valid (offers are still active with sufficient
remaining amounts). It then reserves the funds for the second operation and
copy the transaction object and makes it available at the following URL:

```
curl -XGET https://corewars.org:2406/transactions[transaction_9iop2182cm73s] \
  -H livemode: true

{
  id: "stan@foo.bar[transaction_9iop2182cm73s]",
  livemode: true,
  lock: "ae7b2a3ffd9c43a...",
  pair: "stan@foo.bar[USD.2]/alice@rocket.science[USD.2]",
  operations: [{
    offer: null,
    asset: "stan@foo.bar[USD.2]",
    source: "stan@foo.bar",
    destination: "bob@corewars.org",
    amount: 500,
    status: "reserved"
  }, {
    offer: "bob@corewars.org[offer_s8ka7812djnmk]",
    asset: "bob@corewars.org[USD.2]",
    source: "bob@corewars.org",
    destination: "alice@rocket.science",
    amount: 500,
    status: "reserved"
  }, {
    offer: "alice@rocket.science[offer_9iop2182cm73s]",
    asset: "alice@rocket.science[USD.2]",
    source: "alice@rocket.science",
    destination: "stan@foo.bar",
    amount: 500,
    status: "pending"
  }]
}
```

Similarly to **foo.bar**, it does not reply to the original requests but
instead fowards it to **rocket.science**.

By receiving that request, **rocket.science** undergoes the same process,
checking both **foo.bar** and **corewars.org** transaction objects. It reserves
the funds for the tranaction and replies with the final state of the
transaction (since it is the last mint on the path of the transaction):

```
curl -XPOST https://rocket.science:2406/transactions \
  -H livemode: true \
  -d id=stan@foo.bar[transaction_9iop2182cm73s]

{
  id: "stan@foo.bar[transaction_9iop2182cm73s]",
  livemode: true,
  lock: "ae7b2a3ffd9c43a...",
  pair: "stan@foo.bar[USD.2]/alice@rocket.science[USD.2]",
  operations: [{
    offer: null,
    asset: "stan@foo.bar[USD.2]",
    source: "stan@foo.bar",
    destination: "bob@corewars.org",
    amount: 500,
    status: "reserved"
  }, {
    offer: "bob@corewars.org[offer_s8ka7812djnmk]",
    asset: "bob@corewars.org[USD.2]",
    source: "bob@corewars.org",
    destination: "alice@rocket.science",
    amount: 500,
    status: "reserved"
  }, {
    offer: "alice@rocket.science[offer_9iop2182cm73s]",
    asset: "alice@rocket.science[USD.2]",
    source: "alice@rocket.science",
    destination: "stan@foo.bar",
    amount: 500,
    status: "reserved"
  }]
}
```

In turn **corewars.org** updates its state with the latest state of the
transaction and replies to **foo.bar**. Finally **foo.bar** updates the state
of the transaction (eventually checking with **rocket.science** directly) and
replies to the initial creation request. At this stage, the transaction is
fully reserved and considered valid.

#### Settle an operation

Settling a transaction is a similar process with calls going the opposite
direction. To settle an operation, a mint must have the secret that permitted
the generation of the hash lock under the `lock` attribute of the
`transaction`; with `secret` and `lock` such that:

```
lock = hex(scrypt(secret, transaction_id, 8, 1, 64))
```

Relying on an hashlock (generated by the initiator of a transaction) prevents
**rocket.science** from settling with **corewars.org** without giving a chance
to **foo.bar** to cancel the transaction if an error happenned along its
creation.

To settle the transction above, **foo.bar** comunicates the secret to
**rocket.science** and recursively, **rocket.science** calls **corewars.org**
and finally **foo.bar** itself.

```
curl -XPOST https://rocket.science:2406/transactions/stan@foo.bar[transaction_9iop2182cm73s]/settle \
  -H livemode: true \
  -d secret=a2bd3ef2249add...

{
  id: "stan@foo.bar[transaction_9iop2182cm73s]",
  livemode: true,
  lock: "ae7b2a3ffd9c43a...",
  pair: "stan@foo.bar[USD.2]/alice@rocket.science[USD.2]",
  operations: [{
    offer: null,
    asset: "stan@foo.bar[USD.2]",
    source: "stan@foo.bar",
    destination: "bob@corewars.org",
    amount: 500,
    status: "settled"
  }, {
    offer: "bob@corewars.org[offer_s8ka7812djnmk]",
    asset: "bob@corewars.org[USD.2]",
    source: "bob@corewars.org",
    destination: "alice@rocket.science",
    amount: 500,
    status: "settled"
  }, {
    offer: "alice@rocket.science[offer_9iop2182cm73s]",
    asset: "alice@rocket.science[USD.2]",
    source: "alice@rocket.science",
    destination: "stan@foo.bar",
    amount: 500,
    status: "settled"
  }]
}
```

