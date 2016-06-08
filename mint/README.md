# Mints

The Settle network connects mint servers exposing the API described in this
document. By default, mints should publicly expose this API over HTTPS on port
`2406`. Users register on mint servers, getting Settle addresses for the domain
under which these servers are running.

The Settle network address `stan@foo.bar` points to user `stan` on the mint
running at `foo.bar` on port `2406`.

## Assets

Assets are issued by users. They are represented by the following patter:
`{ISSUER}:{NAME}.{DECIMAL_LENGTH}`. `{ISSUER}` should be a valid Settle
address, `{NAME}` is an alphanumeric string matching the following regular
expression `[A-Z0-9\-]{1,64}`. If the asset represents a IOU for an existing
currency, the ISO-4217 3-letter code should be used. Decimal length is an
integer between `0` and `24` representing the factor by wich the asset native
amount is scaled down when represented as an 128-bit signed integer.

A few examples of valid assets are:
- **stan@foobar.com:USD.2 320** represents **$3.20** issued by Stan.
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

Authentication should rely on HTTP Basic authentication whether this is through
the use of API keys or actual `user:password` credentials.

#### Create an offer

An offer represents a user's willingness to *sell* a certain amount of an asset
for another asset she owns at a given ask price.

Offers to *sell* an asset lives on the mint for the issuer of that asset. As an
example an offer to sell **stan@foo.bar:USD.2** for **bob@corewards.org:USD.2**
lives on **foo.bar**'s mint. Conversely, an oppositve offer to sell
**bob@corewards.org:USD.2** for **stan@foo.bar:USD.2** lives on
**corewars.org**'s mint.

An offer to sell an asset can be created by specifying a `buy` asset, an `ask`
price (price of the sold asset for a native unit of the bought asset) and an
`amount` the total amount of the sold asset the user is willing to sell.

An offer to buy **5 au laits** for **$3.20** each would be created as follows:

```
curl -XPOST https://foo.bar:2406/assets/stan@foo.bar:USD.2/offers \
  -H livemode: true \
  -u username:password \
  -d buy=info@sightglasscoffee.com:AU-LAIT.0 \
  -d ask=320& \
  -d amount=1600

{
  id: "stan@foo.bar:USD.2/offer_7t3sk24sdvz0a",
  sell: "stan@foo.bar:USD.2",
  buy: "info@sightglasscoffee.com:AU-LAIT.0",
  ask: 320,
  amount: 1600,
  remaining_amount: 1600,
  status: "active",
  transactions: []
}
```

Given the above definition, offers for an asset pair, or an asset path
generally live on multiple mints. Mints are not in charge of crossing offers,
transactions should be created for that purpose.

#### Create a transaction

Creating a simple transaction let you credit a user account with a given asset.

```
curl -XPOST https://foo.bar:2406/assets/stan@foo.bar:USD.2/transactions \
  -H livemode: true \
  -u username:password \
  -d amount=500 \
  -d to="bob@corewars.com"

{
  id: "stan@foo.bar:USD.2/transaction_9iop2182cm73s",
  operations: [{
    offer: null,
    asset: "stan@foo.bar:USD.2",
    from: "stan@foo.bar",
    to: "bob@corewars.com",
    amount: 500,
    status: "settled"
  }]
}
```

A transaction can define a path to cross an offer. If the destination (`to`
parameter) is ommitted, the initiator of the transaction is assumed:

Assuming we have one active offer on the network:
- **bob@corewars.org:USD.2/offer_s8ka7812djnmk**: selling **500** at **100**
  for **stan@foo.bar:USD.2**

```
curl -XPOST https://foo.bar:2406/assets/stan@foo.bar:USD.2/transactions \
  -H livemode: true \
  -u username:password \
  -d buy=bob@corewards.org:USD.2 \
  -d ask=100 \
  -d amount=500 \
  -d path[]=bob@corewars.org:USD.2/offer_s8ka7812djnmk

{
  id: "stan@foo.bar:USD.2/transaction_9iop2182cm73s",
  lock: "ae7b2a3ffd9c43a...",
  operations: [{
    offer: null,
    asset: "stan@foo.bar:USD.2",
    from: "stan@foo.bar",
    to: "bob@corewars.org",
    amount: 500,
    status: "reserved"
  }, {
    offer: "bob@corewars.org:USD.2/offer_s8ka7812djnmk",
    asset: "bob@corewars.org:USD.2",
    from: "bob@corewars.org",
    to: "stan@foo.bar",
    amount: 500,
    status: "reserved"
  }]
}
```

Crossing an offer involves two operations, one to move the bidded asset to the
seller and one to move the offered asset to the initiator of the transaction.

A transaction can specify a destination as well as a longer path to send money
by crossing a path of offers.

Let's assume we have two active offers on the network:
- **bob@corewars.org:USD.2/offer_s8ka7812djnmk**: selling **500** at **100**
  for **stan@foo.bar:USD.2**
- **alice@rocket.science:USD.2/offer_9iop2182cm73s**: selling **3500** at **100**
  for **bob@corewars.org:USD.2**

Creating the following transaction will cross these two offers for **500**,
effectively sending **500 alice@rocket.science:USD.2** to Alice by spending
**500 stan@foo.bar:USD.2**.

```
curl -XPOST https://foo.bar:2406/assets/stan@foo.bar:USD.2/transactions \
  -H livemode: true \
  -u username:password \
  -d buy=alice@rocket.science:USD.2 \
  -d to=alice@rocket.science \
  -d ask=100 \
  -d amount=500 \
  -d path[]=bob@corewars.org:USD.2/offer_s8ka7812djnmk \
  -d path[]=alice@rocket.science:USD.2/offer_9iop2182cm73s

{
  id: "stan@foo.bar:USD.2/transaction_9iop2182cm73s",
  lock: "ae7b2a3ffd9c43a...",
  operations: [{
    offer: null,
    asset: "stan@foo.bar:USD.2",
    from: "stan@foo.bar",
    to: "bob@corewars.org",
    amount: 500,
    status: "reserved"
  }, {
    offer: "bob@corewars.org:USD.2/offer_s8ka7812djnmk",
    asset: "bob@corewars.org:USD.2",
    from: "bob@corewars.org",
    to: "alice@rocket.science",
    amount: 500,
    status: "reserved"
  }, {
    offer: "alice@rocket.science:USD.2/offer_9iop2182cm73s",
    asset: "alice@rocket.science:USD.2",
    from: "alice@rocket.science",
    to: "stan@foo.bar",
    amount: 500,
    status: "reserved"
  }]
}
```

### Public API

#### Retrieve order books

Offers can be retrieved publicly from the following endpoints.  Retrieving both
legs lets your reconstruct the full order book for an asset pair:

```
curl -XGET https://foo.bar:2406/assets/stan@foo.bar:USD.2/offers?
  buy=bob@corewars.org \
  -H livemode: true

[{
  id: "stan@foo.bar:USD.2/offer_7t3sk24sdvz0a",
  sell: "stan@foo.bar:USD.2",
  buy: "bob@corewars.org:USD.2",
  ask: 100,
  amount: 5000,
  status: "active"
}]

curl -XGET https://corewars.org:2406/assets/bob@corewars.org:USD.2/offers?
  buy=stan@foo.bar \
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
reserved operations are reserved for that operation until it is either settled
or canceled.

The created transaction object is made available on **foo.bar** with the
following state:

```
curl -XGET https://foo.bar:2406/assets/stan@foo.bar:USD.2/transactions/transaction_9iop2182cm73s \
  -H livemode: true

{
  id: "stan@foo.bar:USD.2/transaction_9iop2182cm73s",
  lock: "ae7b2a3ffd9c43a...",
  operations: [{
    offer: null,
    asset: "stan@foo.bar:USD.2",
    from: "stan@foo.bar",
    to: "bob@corewars.org",
    amount: 500,
    status: "reserved"
  }, {
    offer: "bob@corewars.org:USD.2/offer_s8ka7812djnmk",
    asset: "bob@corewars.org:USD.2",
    from: "bob@corewars.org",
    to: "alice@rocket.science",
    amount: 500,
    status: "pending"
  }, {
    offer: "alice@rocket.science:USD.2/offer_9iop2182cm73s",
    asset: "alice@rocket.science:USD.2",
    from: "alice@rocket.science",
    to: "stan@foo.bar",
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
curl -XPOST https://corewars.org:2406/assets/bob@corewars.org:USD.2/transactions \
  -H livemode: true \
  -d id=stan@foo.bar:USD.2/transaction_9iop2182cm73s
```

By receiving this request **corewars.org** retrieves the transaction from
**foo.bar** effectively checking that **foo.bar** has reserved the funds and
that the operations are valid (offers are still active with sufficient
remaining amounts). It then reserves the funds for the second operation and
copy the transaction object and makes it available at the following URL:

```
curl -XGET https://corewars.org:2406/assets/bob@corewars.org:USD.2/transactions/transaction_9iop2182cm73s \
  -H livemode: true

{
  id: "stan@foo.bar:USD.2/transaction_9iop2182cm73s",
  initiator: "stan@foo.bar:USD.2",
  lock: "ae7b2a3ffd9c43a...",
  operations: [{
    offer: null,
    asset: "stan@foo.bar:USD.2",
    from: "stan@foo.bar",
    to: "bob@corewars.org",
    amount: 500,
    status: "reserved"
  }, {
    offer: "bob@corewars.org:USD.2/offer_s8ka7812djnmk",
    asset: "bob@corewars.org:USD.2",
    from: "bob@corewars.org",
    to: "alice@rocket.science",
    amount: 500,
    status: "reserved"
  }, {
    offer: "alice@rocket.science:USD.2/offer_9iop2182cm73s",
    asset: "alice@rocket.science:USD.2",
    from: "alice@rocket.science",
    to: "stan@foo.bar",
    amount: 500,
    status: "pending"
  }]
}
```

Similarly to **foo.bar**, it does not reply to the original requests but
instead fowards it to **rocket.science**.

By receiving that request, **rocket.science** undergoes the same process,
checking both **foo.bar** and then **corewards.org** transaction objects. It
reserves the funds for the tranaction and replies with the final state of the
transaction:

```
curl -XPOST https://rocket.science:2406/assets/alice@rocket.science:USD.2/transactions \
  -H livemode: true \
  -d id=stan@foo.bar:USD.2/transaction_9iop2182cm73s

{
  id: "stan@foo.bar:USD.2/transaction_9iop2182cm73s",
  initiator: "stan@foo.bar:USD.2",
  lock: "ae7b2a3ffd9c43a...",
  operations: [{
    offer: null,
    asset: "stan@foo.bar:USD.2",
    from: "stan@foo.bar",
    to: "bob@corewars.org",
    amount: 500,
    status: "reserved"
  }, {
    offer: "bob@corewars.org:USD.2/offer_s8ka7812djnmk",
    asset: "bob@corewars.org:USD.2",
    from: "bob@corewars.org",
    to: "alice@rocket.science",
    amount: 500,
    status: "reserved"
  }, {
    offer: "alice@rocket.science:USD.2/offer_9iop2182cm73s",
    asset: "alice@rocket.science:USD.2",
    from: "alice@rocket.science",
    to: "stan@foo.bar",
    amount: 500,
    status: "reserved"
  }]
}
```

In turn **corewars.org** updates its state with the latest state of the
transaction and replies to **foo.bar**, and finally **foo.bar** updates the
state of the transaction (eventually checking with **corewars.org**) and
replies to the initial creation request. At this stage, the transaction is
fully reserved and considered valid.

#### Settle an operation

It is the responsibility of each mints to settle operations that benefit their
users (their user is in the `to` field). To settle an operation, a mint must
have the secret that permitted the generation of the hash lock under the `lock`
attribute of the `transaction`; with `secret` and `lock` such that:

```
lock = hex(scrypt(secret, transaction_id, 8, 1, 64))
```

Relying on an hashlock (generated by the initiator of a transaction) prevents
**rocket.science** from settling with **corewars.org** without giving a chance
to **foo.bar** to cancel the transaction if an error happenned along its
creation.

It is up to the mint to recursively settle as they are given the secret
depending on their trust relationship with the mint before them in the
transaction path.

```
curl -XPOST https://rocket.science2406/assets/alice@rocket.science:USD.2/transactions/transaction_9iop2182cm73s/settle \
  -H livemode: true \
  -d secret=a2bd3ef2249add...

{
  id: "stan@foo.bar:USD.2/transaction_9iop2182cm73s",
  lock: "ae7b2a3ffd9c43a...",
  created: 123123123,
  operations: [{
    offer: "stan@foo.bar:USD.2/offer_7t3sk24sdvz0a",
    asset: "stan@foo.bar:USD.2",
    from: "stan@foo.bar",
    to: "bob@corewars.org",
    amount: 500,
    status: "reserved",
    lifetime: 120
  }, {
    offer: "bob@corewars.org:USD.2/offer_s8ka7812djnmk",
    asset: "bob@corewars.org:USD.2",
    from: "bob@corewars.org",
    to: "alice@rocket.science",
    amount: 500,
    status: "reserved",
    lifetime: 120
  }, {
    offer: "alice@rocket.science:USD.2/offer_9iop2182cm73s",
    asset: "alice@rocket.science:USD.2",
    from: "alice@rocket.science",
    to: "stan@foo.bar",
    amount: 500,
    status: "settled",
    lifetime: 3600
  }]
}
```

