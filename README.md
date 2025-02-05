# verification-dapp

A Go-based verification server that implements Polygon ID authentication. This server handles authentication requests and verifies zero-knowledge proofs for social credentials.

## Features

- Authentication request generation
- ZK proof verification
- Support for Polygon Mumbai testnet
- File serving for static content
- Callback handling for authentication responses

## Prerequisites

- Go 1.x
- Access to Polygon Mumbai RPC node
- Verification keys in the `keys` directory

## Setup

1. Ensure you have the required verification keys in the `../keys` directory
2. Update the following configuration in `go/index.go`:
   - RPC URL endpoint
   - Contract address
   - Callback URL
   - Audience DID

## Running the Server

The server runs on port 8080 and exposes the following endpoints:

- `/` - Serves static content
- `/api/sign-in` - Generates authentication requests
- `/api/callback` - Handles authentication callbacks and verifies proofs

To start the server:

```bash
go run go/index.go
```

## Dependencies

- github.com/ethereum/go-ethereum
- github.com/iden3/go-circuits
- github.com/iden3/go-iden3-auth
- github.com/iden3/iden3comm

## License

[Add your license here]