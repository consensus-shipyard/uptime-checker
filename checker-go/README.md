# Uptime-Checker-Golang
The checker counter-part of `Uptime-Checker`.

## Setup
Install rust with `curl https://sh.rustup.rs -sSf | sh`.
 the source code in the root folder using: `make build/.filecoin-install && make uptime-checker`.

## Usage
`uptime-checker-golang`will look in `~/.lotus` to connect to a running daemon and resume checking of both nodes and fellow checkers.

For other usage see `./uptime-checker --help`

Before starting the checker, define the following env variable:
```
FULL_NODE=$(./lotus auth api-info --perm admin)
export ${FULL_NODE}
```
Then start the app using `./uptime-checker run ...`.