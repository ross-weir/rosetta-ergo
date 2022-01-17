<p align="center">
  <a href="https://www.rosetta-api.org">
    <img width="90%" alt="Rosetta" src="https://www.rosetta-api.org/img/rosetta_header.png">
  </a>
</p>
<h3 align="center">
   Rosetta Ergo
</h3>
<p align="center">
  <a href="https://github.com/ross-weir/rosetta-ergo/actions/workflows/ci.yml"><img src="https://github.com/ross-weir/rosetta-ergo/actions/workflows/ci.yml/badge.svg" /></a>
  <a href="https://goreportcard.com/report/github.com/ross-weir/rosetta-ergo"><img src="https://goreportcard.com/badge/github.com/ross-weir/rosetta-ergo" /></a>
</p>

## Overview
`rosetta-ergo` provides a reference implementation of the Rosetta API for Ergo in Golang. If you haven't heard of the Rosetta API, you can find more information [here](https://rosetta-api.org).

## Terminology

Rosetta uses abstract terminology to represent entities in a blockchain.

Here's how they relate to ergo:

| Rosetta   | Ergo               | Notes                 |
|-----------|--------------------|-----------------------|
| `coin`    | `utxo`, `ergo box` |                       |
| `account` | `address`          | Any ergo address type |

## Implementation status

### Data

- [x] ~~`Network`~~
  - [x] ~~`/network/list`~~
  - [x] ~~`/network/options`~~
  - [x] ~~`/network/status`~~
- [x] ~~`Account`~~
  - [x] ~~`/account/balance`~~
  - [x] ~~`/account/coins`~~
- [x] ~~`Block`~~
  - [x] ~~`/block`~~
  - [x] ~~`/block/transaction`~~
- [x] ~~`Mempool`~~
  - [x] ~~`/mempool`~~
  - [x] ~~`/mempool/transaction`~~

### Construction

- [ ] `Construction`
  - [ ] `/construction/combine`
  - [ ] `/construction/derive`
  - [ ] `/construction/hash`
  - [ ] `/construction/metadata`
  - [ ] `/construction/parse`
  - [ ] `/construction/payloads`
  - [ ] `/construction/preprocess`
  - [ ] `/construction/submit`
