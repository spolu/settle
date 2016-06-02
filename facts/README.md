# Fact smart contracts

A Fact smart contract can be created to assert of a fact and let other accounts
verify the fact. The Fact smart contract source code is available
[here](https://github.com/spolu/settle/tree/master/facts/contracts/facts-V0001.sol).

When an account creates a Fact, it must specify the type of the Fact (`bytes8`)
by complying to the convention proposed in the contract source code (as of
`V0001`):

```
    // 0x010* is reserved for legal information.
    bytes8 constant T_FULL_NAME             = 0x0101
    bytes8 constant T_ENTITY_TYPE           = 0x0102
    bytes8 constant T_DATE_OF_BIRTH         = 0x0103
    bytes8 constant T_DATE_OF_CREATION      = 0x0104
    bytes8 constant T_DATE_OF_INCORPORATION = 0x0105
    // 0x020* is reserved for contact information.
    bytes8 constant T_EMAIL                 = 0x0201
    bytes8 constant T_PHONE                 = 0x0202
    bytes8 constant T_URL                   = 0x0203
    // 0x030* is reserved for address information.
    bytes8 constant T_ADDRESS_LINE1         = 0x0301
    bytes8 constant T_ADDRESS_LINE2         = 0x0302
    bytes8 constant T_ADDRESS_CITY          = 0x0303
    bytes8 constant T_ADDRESS_POSTAL_CODE   = 0x0304
    bytes8 constant T_STATE                 = 0x0305
    bytes8 constant T_COUNTRY               = 0x0306
    // 0x040* is reserved for official document IDs.
    bytes8 constant T_TAX_ID                = 0x0401
    bytes8 constant T_PASSPORT_NUMBER       = 0x0402
    bytes8 constant T_DRIVER_LICENSE_ID     = 0x0403
    // 0x050* is reserved for financial information.
    bytes8 constant T_BANK_ACCOUNT_NUMBER   = 0x0501
    bytes8 constant T_CARD_NUMBER           = 0x0502
    bytes8 constant T_BITCOIN_ADDRESS       = 0x0502
    // 0x060* is reserved for social profiles.
    bytes8 constant T_FACEBOOK_ID           = 0x0601
    bytes8 constant T_TWITTER_HANDLE        = 0x0602
    bytes8 constant T_REDDIT_HANDLE         = 0x0603
    bytes8 constant T_GITHUB_HANDLE         = 0x0604

    // 0x0*** is reserved for future uses.
    // 0x9*** can be used for custom application specific fact types.
```

The value behind a Fact is not made available to the contract, as it would make
it public. Instead the fact owner must also specify `hash = sha3(owner, value)`
when creating the Fact, where `value` is the value underlying th Fact.

When another accounts verifies a Fact (meaning that they have access to the
underlying value), they do so by calling `verify` with `hash = sha3(authority, value)`
where `authority` is their own address.

The validity of the `hash` at Fact creation or verification is not (and cannot)
be enforced by the Contract. Instead anyone having access to the value and
willing to validate a Fact assertion for an authority, should compute `hash =
sha3(authority, value)` and call `check(authority, hash)`. The hash will be
checked against the hash provided by the authority at assertion time, ensuring
the same underlying value was asserted.
