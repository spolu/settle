# API

The Settl API provides the following core functionalities:
- Key management: storing encrypted seeds based on scrypt for secure and
  user-friendly access from multiple devices.
- Name resolution: the API takes care of translating Settl usernames or
  federated addresses into native addresses transparently.
- 

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
curl -XGET settl.net/challenges?count=2
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
curl -XPOST settl.net/stellar/operations
  -H "Challenge: P7m4Iy13sludc0Hu:JcRPRdbKtmSxF0MsKZVgZ/pdrRPxikf59C+q9DG6RQDbXq4zHSOUQriBF187d1VPSPCLIYqJVtqgRM1xXhukAw=="
  -u $address:$signature
  -d type={payment, path_payment, manager_offer}
  -d destination=alistair*settl.net
  -d asset=usd
  -d amount=5

{
  operation: {
    id: "operation_k23hj2o2mlkno",
    type: "payment",
    status: "pending",
    token: "1463699742482_x92krP3Nl0uaVOQL:VpguS4+k+uZP6DuxORFic+...",
    destination: "alistair*settl.net",
    amount: 5,
    asset: "usd",
    transaction: "ASDVASD/ASdsa..."
  }
}
```

# API reference

## Account creation

```
curl -XPOST settl.net/users
  -H "Challenge: P7m4Iy13sludc0Hu:JcRPRdbKtmSxF0MsKZVgZ/pdrRPxikf59C+q9DG6RQDbXq4zHSOUQriBF187d1VPSPCLIYqJVtqgRM1xXhukAw=="
  -H $address:$signature
  -d username=stan
  -d address=ABASD...
  -d encrypted_seed=ASDA...
```

## Stellar Operations

```
curl -XPOST settl.net/stellar/operations
  -H "Challenge: P7m4Iy13sludc0Hu:JcRPRdbKtmSxF0MsKZVgZ/pdrRPxikf59C+q9DG6RQDbXq4zHSOUQriBF187d1VPSPCLIYqJVtqgRM1xXhukAw=="
  -H $address:$signature
  -d type={payment, path_payment, manager_offer}
  -d destination=alistair*settl.net
  -d asset=usd
  -d amount=5

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
curl -XPOST settl.net/stellar/operations/operation_k23hj2o2mlkno/submit
  -H $address:$signature
  -d token=1oeiwqlkdj_1231230123_k2jt329jpfldfas
  -d envelope="ADWQ/khj21e..."
```

```
curl -XGET settl.net/stellar/operations
```
