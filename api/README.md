# API

The Settl API provides the following core functionalities:
- Key management: storing encrypted seeds based on scrypt for secure and
  user-friendly access from multiple devices.
- Name resolution: the API takes care of translating Settl usernames or
  federated addresses into native addresses transparently.
- Augmented functionalities: such as easing the issuance and exchange of
  currencies between Settl users, providing tools to accept payments through
  automatic exchange of these currencies.

# Account creation and key management

Settl only stores encrypted version of the seed associated with a user
account. User creation involves the following steps:

```
- (client-side) generate a keypair (p_A, k_A).
- (client-side) encrypt the private key using scrypt with password pw_A:
  e_A = k_A XOR scrypt(pw_A, p_A).
- (client-side) use onboarding mechanism to create underlying Stellar account
  and verify email.
- (client-side) submit username u_A, p_A, e_A to server and authenticate the
  request by resolving an authentication challenge (see below).
- (server) create user and store e_A, p_A and u_A
```

# Authentication

Each request must specify a `Challenge` header and use Basic Authentication
with the user address as username and the signature of the challenge as
password. Challenges can only be used once. Challenges values can be retrieved
using the following endpoint without authentication:
```
curl -XGET https://api.settl.net/challenges?count=2
{
  "challenges": [
    {
      "value": "P7m4Iy13sludc0Hu:JcRPRdbKtmSxF0MsKZVgZ/pdrRPxikf59C+q9DG6RQDbXq4zHSOUQriBF187d1VPSPCLIYqJVtqgRM1xXhukAw==",
      "created": 1463788608772
    },
    {
      "value": "FkDOYmXaTgtlmi26:/xsKdGgSAizTxV3nSGymas3NNhROaIc1dPc4wm5oBBHGn9pMJMp/464wCf9dxMEEHW4aY7ycYHli4uZKldqWAQ==",
      "created": 1463788608773
    }
  ]
}
```

```
curl -XPOST https://api.settl.net/stellar/operations
  -H "Livemode: true" \
  -H "Authentication-Challenge: P7m4Iy13sludc0Hu:JcRPRdbKtmSxF0MsKZVgZ/pdrRPxikf59C+q9DG6RQDbXq4zHSOUQriBF187d1VPSPCLIYqJVtqgRM1xXhukAw==" \
  -u $address:$signature \
  -d type={payment, path_payment, manager_offer} \
  -d destination=alistair*settl.net \
  -d asset=usd \
  -d amount=5
```

# API reference

## Account creation

Account creation is the final step of a user onboarding on Settl. It creates a
user for a given username, email and existing Stellar account. The email must
be asserted by a trusted verifier (see Facts Protocol). Actual email
verification and Stellar account creation happens outside of the API.

```
curl -XPOST https://api.settl.net/users
  -H "Livemode: true" \
  -H "Authentication-Challenge: P7m4Iy13sludc0Hu:JcRPRdbKtmSxF0MsKZVgZ/pdrRPxikf59C+q9DG6RQDbXq4zHSOUQriBF187d1VPSPCLIYqJVtqgRM1xXhukAw==" \
  -u $address:$signature \
  -d username=stan \
  -d email=polu.stanislas@gmail.com \
  -d verifier=GABL2... \
  -d encrypted_seed=WgH...
```

## Native Operations

Native operations lets Settl users perform native Stellar operations from any
device using their username and password.

```
curl -XPOST settl.net/native/operations
  -H "Livemode: true" \
  -H "Authentication-Challenge: P7m4Iy13sludc0Hu:JcRPRdbKtmSxF0MsKZVgZ/pdrRPxikf59C+q9DG6RQDbXq4zHSOUQriBF187d1VPSPCLIYqJVtqgRM1xXhukAw==" \
  -u $address:$signature \
  -d type={payment|path_payment|manager_offer} \
  -d destination=alistair \
  -d asset=usd \
  -d amount=5 \

{
  operation: {
    id: "operation_k23hj2o2mlkno",
    type: "payment",
    status: "pending",
    destination: "alistair*settl.net",
    amount: 5,
    asset: "usd",
    transaction: "ASDVASD/ASdsa..."
  }
}

```

```
curl -XPOST https://settl.net/native/operations/operation_k23hj2o2mlkno/submit
  -H "Livemode: true" \
  -H "Authentication-Challenge: P7m4Iy13sludc0Hu:JcRPRdbKtmSxF0MsKZVgZ/pdrRPxikf59C+q9DG6RQDbXq4zHSOUQriBF187d1VPSPCLIYqJVtqgRM1xXhukAw==" \
  -u $address:$signature \
  -d envelope="ADWQ/khj21e..."
```
