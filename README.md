# omnistash

omnistash is a registry for cataloging and distributing artifacts. This registry is [ORAS](https://oras.land).

## Set up the database

Run migrations before starting the server for the first time, or after upgrading.

Download the latest `omnistash-migrate` binary from the [releases page](../../releases) and run:

```bash
omnistash-migrate -dsn omnistash.db
```

| Flag | Default | Description |
|---|---|---|
| `-dsn` | | SQLite database path (required) |

## Run the server

```bash
go run ./cmd/omnistash -addr :10080 -metastore-dsn omnistash.db
```

| Flag | Default | Description |
|---|---|---|
| `-addr` | `:10080` | Listen address |
| `-metastore-dsn` | `omnistash.db` | SQLite database path |

The server logs to stderr as JSON and shuts down gracefully on `SIGINT` or `SIGTERM`.
