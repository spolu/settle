# Facts Protocol

Accounts can assert of facts using the Data operation and the Facts Protocol.
A fact is stored on the account asserting it and uses the following format:

```
  "$address.$type": "$value"
```

`$address` is the Stellar address of the account the facts relates to. `$type`
is the fact type and should be one of the following:

- `name`: Full name of a individual, company or organization.
- `entity_type`: Entity type (individual, for-profit, non-profit, state).
- `date_of_birth`: Date of birth.
- `date_of_creation`: Date of creation of an organization.
- `email`: Fully qualified email address.
- `url`: Fully qualified URL.
- `phone`: Fully qualified phone number, without space or separator and
  starting with `+` and country code (example: `+14152165701`).
- `twitter`: Twitter handle without preceeding `@`.
- `github`: Github handle.
- `bank_account_scrypt`: the scrypt of the bank account using the address as
  salt: `scrypt(bank_account, address, 1<<14, 8, 1, len(address))`
- `tax_id_scrypt`: the scrypt of the tax ID (SSN, EIN, ...) using the
  address as salt: `scrypt(tax_id, address, 1<<14, 8, 1, len(address))`

