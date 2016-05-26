# Facts Protocol

Accounts can assert of facts related to other accounts using the Stellar
ManageData operation and the Facts Protocol. A fact is stored on the account
asserting it, also called the "verifier" account and uses the following format
to assert something about the "destination" account the facts relates to:

```
  "fct.$destination_address.$type_code": "scrypt($value, $verifier_address, 1<<14, 8, 1, 64)"
```

- `$destination_address` is the Stellar address of the destination account the
facts relates to.
- `$verifier_address` is the Stellar address of the verifier account.
- `$type_code` is the fact type and should be one of the following (see list
  below).

- `00x` is reserved for legal information:
  - `000` (`name`): Full name of a individual, company or organization.
  - `001` (`entity_type`): Entity type (individual, for-profit, non-profit,
    state).
  - `002` (`date_of_birth`, `date_of_creation`, `date_of_incorporation`): Date
    of birth or creation or incorporation of an organization in the format
    YYYY-MM-DD.
- `01x` is reserved for contact information:
  - `010` (`email`): Fully qualified lowercased email address.
  - `011` (`phone`): Fully qualified phone number, without space or separator
    and starting with `+` and country code (example: `+14152165701`).
  - `012` (`url`): Fully qualified URL.
- `02x` is reserved for address information:
  - `020` (`address_line1`): Address line1.
  - `021` (`address_line2`): Address line2.
  - `022` (`address_city`): Address city.
  - `023` (`address_postal_code`): Address postal code.
  - `024` (`address_state`): Address state.
  - `025` (`address_country`): Address country.
- `03x` is reserved for supporting documents:
  - `030` (`passport`): A PNG of the passport.
  - `031` (`driver_license`): A PNG of the driver license.
- `04x` is reserved for financial information:
  - `040` (`bank_account`): Fully qualified bank account (the IBAN if EU, the
    routing and account number concatenated by `-` if in the US)
  - `041` (`tax_id`): The tax ID (SSN, EIN, ...)
- `05x` is reserved for social profiles:
  - `050` (`facebook`): Facebook account ID.
  - `051` (`twitter`): Twitter handle without preceeding `@`.
  - `053` (`reddit`): Reddit handle.
  - `054` (`github`): Github handle.
- `9xx` is reserved for custom application use.

As specified above, the value stored is an scrypt of the actual value using the
verifier address as salt (proving that the verifier did see the actual value).
When verifying a fact, the actual value and the verifier address should be
communicated through whatever communication channel available. Verifying a fact
does not require any form of authentication.
