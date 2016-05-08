# Fact Protocol

Facts are typed values related to a Stellar account along with a list of
assertions from arbitrary accounts asserting that the fact is true. Facts are
created and retrievable on public APIs using the Fact Protocol described in
this document. `type` and `value` are strings whose validation is left to the
discretion of the service running the Fact Protocol. Once created, a fact
(type, value, account) is immutable. Facts can only be created by the account
they relate to, also called the fact owner:

```
curl -XPOST https://FACT_SERVER/accounts/$pk0/facts \
  -d type=$type \
  -d value=$value \
  -d account=$pk0 \
  -d signature=$sig(action=assert&account=$pk0&type=$type&value=$value)
```

The API returns a JSON body representing the newly created fact:

```
{
  "fact": {
    "id": "fact_1a1ed89jh10dj0123",
    "account": "$pk0",
    "type": "email",
    "value": "polu.stanislas@gmail.com",
    "operation_xdr": "..."
    "assertions": [{
      "id": "assertion_d9ceqw09dfwife0wef",
      "account": "$pk0",
      "signature": "sca239afsd0..."
    }],
    "revocations": []
  }
}
```
Creating a fact implicitely assert that the fact is true, hence automatically
creating an assertion from the fact owner.

Facts are then publicized as Stellar accounts Data fields using the following
format as key `fact:$domain:$type` and the unique `$id` returned by the
Fact API as value:

```
{
  ...
  "fact:facts.settl.net:email": "fact_1a1ed89jh10dj0123",
  ...
}
```

`FACT_SERVER`, the base URL to interact with the service maintaining the
advertised fact, can be retrieved from the `$domain`'s `stellar.toml` file.

It is invalid to publicize a fact that is not owned by the entiy publicizing
it. For easy in-place validation (without querying the associated Fact API),
fact IDs include the account (public key) of their owner.

Facts can be asserted by other Stellar accounts representing arbitrary entities
using the following public endpoints:

```
curl -XPOST https://FACT_SERVER/accounts/$pk0/facts/$id/assertions \
  -d account=$pk1 \
  -d signature=$sig(action=assert&account=$pk0&type=$type&value=$value)
```

Returning a JSON body representing the newly created assertion:
```
{
  "assertion": {
    "id": "assertion_d9ceqw09dfwife0wef",
    "account": "$pk1",
    "signature": "sca239afsd0..."
  }
}
```

Facts, assertions and revocations can be retrieved publicly using the following
endpoints:

```
curl -XGET https://FACT_SERVER/accounts/$pk0/facts/$id
```

Facts assertions (and indirectly facts) can be revoked using the following API.
Once a revocation is made, it hides the assertion from the list of assertions
for the fact. If the owner of a fact revokes its assertion, it permanently
hides the fact from the API.

```
curl -XPOST https://FACT_SERVER/accounts/$pk0/facts/$id/revocations \
  -d account=$pk1 \
  -d signature=$sig(action=revoke&account=$pk0&type=$type&value=$value)
```

Facts cannot be revoked without being previously asserted.

# Settl Fact Types

- `name`: Full name of a individual, company or organization.
- `type`: Entity type (individual, for-profit, non-profit, state).
- `date_of_birth`: Date of birth.
- `date_of_creation`: Date of creation of an organization.
- `email`: Fully qualified email address.
- `url`: Fully qualified URL.
- `phone`: Fully qualified phone number.
- `twitter`: Twitter handle without preceeding `@`.
- `github`: Github handle.
- `bank_account_hash`: The hash of a bank account number using the Stellar
   account public key as nonce.
- `hashed_tax_id`: The hash of the Tax ID (SSN, EIN, ...) using the Stellar
   account public key as nonce.

