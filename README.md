# MLOpsTrack

A lightweight experiment tracking and MLOps platform built with Go and Python.

## Current Features

- Go HTTP API
- PostgreSQL integration
- Experiment creation and listing
- Run creation and listing
- Run parameter tracking
- Training metric tracking
- Run lifecycle tracking (`running`, `completed`, `failed`)
- Python SDK for experiment and run tracking
- PyTorch training integration with automatic metric logging
- Learning-rate comparison across tracked training runs
- Automated API request validation tests
- Automated Go testing with GitHub Actions
- Model artifact upload, listing, and download

## Tech Stack

- Go
- PostgreSQL
- Python


## Tests

Run the Go test suite:

    go test ./...

Run tests with detailed output:

    go test -v ./internal/api

Run Go  analysis:

    go vet ./...

the API validation tests check invalid request bodies,
missing required fields, unsupported HTTP methods,
and invalid parameters without requiring a database.



## Tests

Run all Go tests:

```bash
go test ./...
```

Run API tests:

```bash
go test -v ./internal/api
```

Check code:

```bash
go vet ./...
```

### Database Integration Tests

Database integration tests use a separate PostgreSQL
database named `mlopstrack_test`.

Apply the SQL migrations from the `migrations/`
directory before running the tests.

Set `TEST_DATABASE_URL` to the test database connection
string, then run:

    go test -v ./internal/api -run TestDatabaseIntegration -count=1

The tests verify experiment and run creation, database
reads, parameter and metric upserts, and run completion.

Test records are removed automatically after execution.

If `TEST_DATABASE_URL` is not set, the integration test
is skipped.

## Experiment Comparison

Compare training runs with different learning rates (0.01, 0.1, 0.2).

```powershell
.\.venv\Scripts\python.exe -m training.compare
```

Runs are compared by final validation accuracy.


## Status

Under active development.