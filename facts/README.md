# Facts

Facts are typed values related to an entity along with a list of certifying
entities signatures asserting that the fact is true. Facts are created and
retrievable on a public Fact API using the endpoints described in this
document. `type` and `value` are strings whose validation is left to the
discretion of the service running the Fact API. Once created, a fact (type,
value, entity) is immutable. Facts can only be created by the entity they
relate to, also called the fact owner.

```
curl -XPOST https://settl.net/facts \
  -d type=$type \
  -d value=$value \
  -d public_key=$pk0 \
  -d signature=$sig(action=assert&entity=$pk0&type=$type&value=$value)
```

The Fact API returns a JSON body representing the newly created assertion:

```
{
  "id": "fact_1a1ed89jh10dj0123_PK0...",
  "entity": "PK0...",
  "type": "email",
  "value": "polu.stanislas@gmail.com",
  "operation_xdr": "..."
  "signatures": [{
    "id": "signature_d9ceqw09dfwife0wef",
    "public_key": "PK0...",
    "signature": "sca239afsd0..."
  }],
  "revocations": []
}
```
Creating a fact implicitely assert that the fact is true.

Facts are then publicized as Stellar accounts Data fields using the following
format as key `fact:$domain:$type` and the unique `$id` returned by the
Fact API as value:

```
{
  ...
  "fact:settl.net:email": "fact_$pk0_1a1ed89jh10dj0123",
  ...
}

```
It is invalid to publicize a fact that is not owned by the entiy publicizing
it. For easy in-place validation (without querying the associated Fact API),
fact IDs include the public key of their owner.

Facts can be signed and certified by other Stellar accounts representing
official or unofficial entities using the following public endpoints:

```
curl -XPOST https://settl.net/facts/$id/signatures \
  -d public_key=$pk1 \
  -d signature=$sig(action=assert&entity=$pk0&type=$type&value=$value)
```

Returning a JSON body representing the newly created signature:
```
{
  "id": "signature_d9ceqw09dfwife0wef",
  "public_key": "ASD...",
  "signature": "sca239afsd0..."
}
```

Facts and Signatures can be retrieved publicly using the following endpoints:

```
curl -XGET https://settl.net/facts/$id
```

Facts signatures (and indirectly facts) can be revoked using the following API.
Once a revocation is made, it hides the signature from the list of signatures
for the fact. If the owner of a fact revokes its signature, it permanently
hides the fact from the API.

```
curl -XPOST https://sett.eu/facts/$id/revocations \
  -d public_key=$pk1 \
  -d signature=$sig(action=revoke&entity=$pk0&type=$type&value=$value)
```

Facts cannot be revoked without being previously signed.

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

