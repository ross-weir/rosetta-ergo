<p align="center">
  <a href="https://www.rosetta-api.org">
    <img width="90%" alt="Rosetta" src="https://www.rosetta-api.org/img/rosetta_header.png">
  </a>
</p>
<h3 align="center">
   Rosetta Ergo
</h3>

## Overview
`rosetta-ergo` provides a reference implementation of the Rosetta API for Ergo in Golang. If you haven't heard of the Rosetta API, you can find more information [here](https://rosetta-api.org).

## Implementation status

### Data

- [x] `Network`
  - [x] `/network/list`
  - [x] `/network/options`
  - [x] `/network/status`
- [ ] `Account`
  - [ ] `/account/balance`
  - [ ] `/account/coins`
- [ ] `Block`
  - [ ] `/block`
  - [ ] `/block/transaction`
- [ ] `Mempool`
  - [ ] `/mempool`
  - [ ] `/mempool/transaction`

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
