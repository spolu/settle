# Facts

Facts are typed values related to an entity along with a list of certifying
entities signatures asserting that the fact is true. Facts are created on
public FactServices using the following API. `type` and `value` are strings
whose validation is left to the discretion of the service. Once created, a fact
type and value is immutable. Facts can only be created by the entity they
relate to.

```
curl -XPOST https://settl.net/facts \
  -d type=$type \
  -d value=$value \
  -d public_key=$pk0 \
  -d signature=$sig(action=assert&entity=$pk0&type=$type&value=$value)
```

The service returns a JSON body representing the newly created assertion:

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
FactService as value:

```
{
  ...
  "fact:settl.net:email": "fact_1a1ed89jh10dj0123_PK0...",
  ...
}

```
It is invalid to publicize a fact that does not rely to the entiy publicizing
it.  Fore easy validation without querying the FactService, fact IDs include
the public key of the entity they relate to.

Facts can be signed and certified true by other Stellar accounts representing
official or unofficial entities using the following public endpoints:

```
curl -XPOST https://settl.net/facts/$id/signatures \
  -d public_key=$pk1
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
Once a revocation is made, it permanently invalidates all previous and future
signatures on this particular fact. If the fact owner revokes it, it
permanently hides the fact from the FactService API, effectively making the
fact invalid.

```
curl -XPOST https://sett.eu/facts/$id/revocations \
  -d public_key=$pk0 \
  -d signature=$sig(action=revoke&entity=$pk0&type=$type&value=$value)
```

# Settl Fact Types

- `name`: Full name of a individual, company or organization.
- `email`: Fully qualified email address.
- `url`: Fully qualified URL.
- `phone`: Fully qualified phone number.
- `twitter`: Twitter handle.
- `hashed_bank_account`: The hash of a bank account number using the Stellar
   account public key as nonce.
- `hashed_tax_id`: The hash of the Tax ID (SSN, EIN, ...) using the Stellar
   account public key as nonce.

