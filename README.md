# Flow Account API

This API is an intermediary for non-custodial Flow wallets. It has two responsibilities:

- Creating a new account on the behalf of a just-initialized wallet.
- Maintaining a registry of `publicKey` -> `accountAddress`.

## Running locally

This project uses Docker Compose and the Flow Emulator to simulate a blockchain environment
on a local machine.

Run this command to start the Docker Compose network:

```shell script
make run
```

### Run with emulator outside of Docker

Start the emulator using the `flow.json` file in this directory:

```shell script
flow emulator start
```

In a separate process, launch the API:

```shell script
make run-with-local-emulator
```

## API Routes

### Create Account

```shell script
curl --request POST \
  --url http://localhost:8081/accounts \
  --header 'content-type: application/json' \
  --data '{
	"publicKey": "6b1523db40836078eb6f80f8d4f934f03725a4e66574815b5d2a9f2ba5dcf9c483fc1b543392f6ada01cc13790f996d0969ee6f9c8d9190f54dc31f44be0a53b",
	"signatureAlgorithm": "ECDSA_P256",
	"hashAlgorithm": "SHA3_256"
}
'
```

Sample response:

```json
{
  "address": "01cf0e2f2f715450",
  "publicKeys": [
    {
      "publicKey": "6b1523db40836078eb6f80f8d4f934f03725a4e66574815b5d2a9f2ba5dcf9c483fc1b543392f6ada01cc13790f996d0969ee6f9c8d9190f54dc31f44be0a53b",
      "signatureAlgorithm": "ECDSA_P256",
      "hashAlgorithm": "SHA3_256"
    }
  ]
}
```

### Get Account By Public Key

```shell script
curl --request GET \
  --url 'http://localhost:8081/accounts?publicKey=6b1523db40836078eb6f80f8d4f934f03725a4e66574815b5d2a9f2ba5dcf9c483fc1b543392f6ada01cc13790f996d0969ee6f9c8d9190f54dc31f44be0a53b'
```

Sample response:

```json
{
  "address": "01cf0e2f2f715450",
  "publicKeys": [
    {
      "publicKey": "6b1523db40836078eb6f80f8d4f934f03725a4e66574815b5d2a9f2ba5dcf9c483fc1b543392f6ada01cc13790f996d0969ee6f9c8d9190f54dc31f44be0a53b",
      "signatureAlgorithm": "ECDSA_P256",
      "hashAlgorithm": "SHA3_256"
    }
  ]
}
```
