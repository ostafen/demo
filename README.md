# Modules structure

- **model** contains domain types definitions which are used by the other modules;
- **store** is responsible for persisting data on disk;
- **api** contains code which exposes the REST apis.

# Building an executable

The service is compatible with Go versions starting from **1.18**. To build an executable of the service, run the following commands:
```bash
git clone https://github.com/ostafen/demo.git
cd demo
go mod download
go build ./cmd/service
```

An executable file named `service` will be created in the demo folder. To run the service with default parameters, simple type:

```bash
./service
```

The service also allows to configure the path where peristent data will be stored (default is "."), and the listen address of the server (default is localhost:8080).

```bash
./service -h

Usage of ./service:
  -host string
    	bind address of the server (default "localhost:8080")
  -storage string
    	root directory where persistent data will be stored (default ".")
```

# Tests

To run module tests and inspect the code coverage, run the following sequence of commands:

```bash
go test ./... -v -coverprofile cover.out
go tool cover -html=cover.out
```

The **store** module contains tests which ensure that data is persistent correctly.

The **api** tests that operations on the event store are correctly exposed by the REST api (each test makes http requests to a background server, which is only valid for the duration of the test).

# REST APIs

- **PUT** /answers: creates a new answer.
- **GET** /answers/{key}: reads an answer.
- **POST** /answers: updates an answer.
- **DELETE** /answers/{key}: deletes an answer.
- **GET** /answers/{key}/events: retrieves the list of events associated to an answer.

Both **PUT** and **POST** requests require a JSON request body containing the answer in the format:

```json
{
  "key: "myKey",
  "value": "myValue"
}
```