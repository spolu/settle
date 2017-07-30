# Settle

*Decentralized trust graph for value exchange without a blockchain.*

Settle's goal is to explore a new financial trust primitive on the Internet,
and doing so, construct a decentralized trust graph enabling (totally) free
exchange of value without relying on a blockchain.

[https://settle.network](https://settle.network)

## Installing the `settle` client

Install `settle` locally (under `~/.settle`):
```
curl -L https://settle.network/install | sh && export PATH=$PATH:~/.settle
```

Or from the source, assuming you have [Go](https://golang.org/) installed:
```
go get -u github.com/spolu/settle/...
```

## Building and running tests

To speed up build and test execution, run `go install` from the following
vendored packages to avoid recompiling them at each build or test run:

```
./vendor/github.com/mattn/go-sqlite3
```

To run tests you may need to increase the number of open files permitted on
your account with:

```
ulimit -n 4096
```
