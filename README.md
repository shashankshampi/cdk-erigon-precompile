<a name="readme-top"></a>

<!-- PROJECT LOGO -->
<br />
<div align="center">
  <a href="https://github.com/shashankshampi/cdk-erigon-precompile">
    <img src="files/logo-full.svg" alt="Logo" width="140" height="100">
  </a>

<h3 align="center">CDK-Erigon Precompile Testing</h3>

  <p align="center">
    EVM Precompile Testing using Kurtosis, Go, and Solidity
    <br />
    <a href="https://github.com/shashankshampi/cdk-erigon-precompile"><strong>Explore the Docs »</strong></a>
  </p>
</div>

---

## Table of Contents

- [Overview](#overview)
    - [Features](#features)
    - [Built With](#built-with)
- [Getting Started](#getting-started)
    - [Fresh Setup](#fresh-setup)
    - [Resetting an Existing Setup](#resetting-an-existing-setup)
    - [Check Setup Status](#check-setup-status)
- [Configuration](#configuration)
- [Usage](#usage)
    - [Step 1: Precompile Raw Call](#step-1-precompile-raw-call)
    - [Step 2: Deploy Solidity Wrapper](#step-2-deploy-solidity-wrapper)
    - [Step 3: Invoke Solidity Wrapper](#step-3-invoke-solidity-wrapper)
- [Validation](#validation)
- [Contact](#contact)

---

## Overview

This repository demonstrates how to test native EVM precompiles in a local CDK-Erigon devnet using Kurtosis. It supports both direct (raw) precompile invocation and via a Solidity wrapper contract. The project is written in Go and leverages the Solidity compiler (`solc`) for wrapper generation.

### Features

- 🧪 Native precompile call (raw `eth_call`)
- ⚙️ Solidity wrapper contract deployment
- 🔁 Wrapper-based precompile invocation and validation
- 📝 Results saved in structured JSON format

### Built With

* [![Go][go.dev]][go-url]
* [![Solidity][solidity-badge]][solidity-url]
* [![Docker][docker.com]][docker-url]
* [![Kurtosis][kurtosis-badge]][kurtosis-url]

---

## Getting Started

### Fresh Setup

```bash
git clone https://github.com/0xPolygon/kurtosis-cdk
cd kurtosis-cdk
kurtosis run . --enclave cdk-erigon
```

Kurtosis Docs: [https://docs.kurtosis.com](https://docs.kurtosis.com)

### Resetting an Existing Setup

```bash
kurtosis enclave ls
kurtosis enclave stop <ENCLAVE_IDENTIFIER>
kurtosis enclave rm <ENCLAVE_IDENTIFIER>
```

### Check Setup Status

```bash
kurtosis engine status
```

Check block production:

```bash
curl --location 'http://127.0.0.1:<port>' --header 'Content-Type: application/json' --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

---

## Configuration

Create a `.env` file in the project root:

```env
DEPLOYER_PRIVATE_KEY=abc
RPC_HOST=127.0.0.1
RPC_PORT=55180
```

---

## Usage

### Step 1: Precompile Raw Call

Install Go module dependencies
```bash
go mod tidy
```

Compile and run:

```bash
go run scripts/stage1_raw_call.go
```

Expected output:

```
=== Precompile Call Results ===
RPC Endpoint: http://127.0.0.1:55180
Precompile Address: 0x02
Input: "hello world"
Expected SHA256: b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
Returned SHA256: b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
✅ Result matches expected hash
📝 Results saved to results_stage1.json
```

---

### Step 2: Deploy Solidity Wrapper

Compile the contract:

```bash
solc contracts/Sha256Wrapper.sol --bin --abi -o artifacts
```

For redeploys:

```bash
solc contracts/Sha256Wrapper.sol --bin --abi -o artifacts --overwrite
```

Deploy:

```bash
go run scripts/stage2_deploy_wrapper.go
```

Expected output:

```
✅ Connected to Ethereum node at http://127.0.0.1:55180
🔐 Using deployer address: 0x...
📦 Bytecode loaded
🔢 Nonce: 9
📨 Sending deployment transaction...
⏳ Waiting for transaction to be mined...
✅ Transaction mined in block 574
✅ Contract verification passed - Code size: 639 bytes
🚀 Deployment successful!
📝 Results saved to results_stage2.json
📌 Contract Address: 0x1f7ad7caA53e35b4f0D138dC5CBF91aC108a2674
```

---

### Step 3: Invoke Solidity Wrapper

```bash
go run scripts/stage3_invoke_wrapper.go
```

Expected output:

```
✅ Connected to Ethereum node at http://127.0.0.1:55180
📌 Using contract at: 0x1f7ad7ca...
✅ Contract verified (code size: 639 bytes)

🧪 Test results:
✅ Input: 'hello world' => OK
✅ Input: '' => OK
✅ Input: 'The quick brown fox...' => OK
✅ Input: 'cdk-erigon' => OK

📝 Results saved to results_stage3.json
```

---

## Validation

All results are saved in the root of the project:

- `results_stage1.json`
- `results_stage2.json`
- `results_stage3.json`

Each file contains structured output logs for corresponding stages of the test suite.

---

## Contact

Created by:

- [@Shashank Sanket](mailto:shashank.sanket1995@gmail.com)

---

<!-- MARKDOWN LINKS & IMAGES -->
[go.dev]: https://img.shields.io/badge/Go-1.23-blue?logo=go
[go-url]: https://go.dev/
[docker.com]: https://img.shields.io/badge/Docker-Container%20Platform-2496ED?logo=docker
[docker-url]: https://www.docker.com/
[solidity-badge]: https://img.shields.io/badge/Solidity-0.8+-363636?logo=solidity
[solidity-url]: https://soliditylang.org/
[kurtosis-badge]: https://img.shields.io/badge/Kurtosis-Test%20Orchestration-orange?logo=docker
[kurtosis-url]: https://docs.kurtosis.com/

<p align="right">(<a href="#readme-top">back to top</a>)</p>