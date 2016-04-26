# Settle

An experimental project aiming to extend the Stellar Netowrk with a strong
notion of identity in order to ease online payment and trust.

# Facts

Facts are created on public FactServices using the following API, `type` and
`value` are strings whose validation is left to the discretion of the fact
service:

```
curl -XPOST https://$domain/facts \
  -H Public-Key: ASD1... \
  -H Signature: $signature(payload) \
  -d type=$type \
  -d value=$value
```

The FactService returns a JSON body representing the newly created fact:

```
{
  "id": "fact_1a1ed89jh10dj0123",
  "type": "email",
  "value": "polu.stanislas@gmail.com",
  "operation_xdr": "..."
}
```

Facts are stored as Stellar accounts Data fields using the following format as
key `fact:$domain:$type` and the unique `$id` returned by the FactService as
value:

```
{
  ...
  "fact:settl.eu:email": "fact_1a1ed89jh10dj0123",
  ...
}

```

Facts can be verified by other Stellar accounts representing official or
unofficial entities using the following public endpoints:

```
curl -XPOST https://settl.eu/facts/$id/signatures \
  -H Public-Key: ASD2... \
  -H Signature: $signature($id)
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
curl -XGET https://settl.eu/facts/$id
curl -XGET https://settl.eu/facts/$id/signatures?cursor=signature_d9ceqw09dfwife0wef&limit=42
```

Facts and signatures can be deleted by their creator using the `DELETE` verb:

```
curl -XDELETE https://sett.eu/facts/$id \
  -H Public-Key: ASD1... \
  -H Signature: $signature($id)

curl -XDELETE https://sett.eu/facts/$fact_id/singatures/$signature_id \
  -H Public-Key: ASD2... \
  -H Signature: $signature($id)
```

# Settle Fact Types

- `name`: Full name of a individual, company or organization.
- `email`: Fully qualified email address.
- `url`: Fully qualified URL.
- `phone`: Fully qualified phone number.
- `twitter`: Twitter handle.
- `hashed_bank_account`: The hash of a bank account number using the Stellar
   account public key as nonce.
- `hashed_tax_id`: The hash of the Tax ID (SSN, EIN, ...) using the Stellar
   account public key as nonce.

