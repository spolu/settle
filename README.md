# Settle

While cryptocurrencies are maintained by distributed ledgers with no central
authority, their trust model and graph is still fully centralized: everyone has
to trust the currency.

Settle's goal is to enable a new "verb" on the Internet: trust; doing so,
constructing a decentralized trust graph that can be used to achieve fluid and
free exchange of value between humans and machines.

The Settle network is composed of [mint](mint/README.md) servers. Anyone can
run a mint and no particular mint is required for the network to operate (a
network of mints can also properly run in a disconnected split of the
Internet). Mints must be online and rely on https for authentication.

Basic operations supported by the Settle network:
- *Issue asset*: users can issue assets (basically IOUs), such as
  **stan@foobar.com:USD.2 320**, and transfer these IOUs to other users.
- *Create offer*: users can create offers to exchange assets, such as
  **stan@foobar.com:USD.2/info@sightglasscofee.com:AU-LAIT.0** at price
  **320/1** for quantity **2**.
- **Create transaction**: exchange assets by crossing a path of offers.

A trust relationship between user A and user B (A trusting B) is simply express
by an outstanding offer by user A to exchange some of their assets again some
of B's assets.

## Example use cases

### Local communities

*TODO*

### Machine to machine

*TODO*

### Global remittance

*TODO*

