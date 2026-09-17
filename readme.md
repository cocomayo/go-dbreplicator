# DB Replicator

DB Replicator is an event-driven, multi-tenant database replication and transformation engine built in Go. It handles fetching data from various source databases, pushing it through a message broker, transforming the payload dynamically using YAML mapping rules, and safely storing it into a centralized destination logging table.

## Architecture & How It Works

The replication pipeline is decoupled into two primary components that communicate asynchronously via RabbitMQ:

### 1. The Producer

Reads dynamic SQL queries (e.g., `data/pintar/queries/kaskel_transaction.sql`) and fetches new rows from the source database. It uses local JSON state files to track a "high-water mark" (like the last `created_date`) so it only fetches new or updated records on subsequent runs. These records are published to a dynamically named RabbitMQ queue.

### 2. The Consumer

Listens to the exact same RabbitMQ queue. When messages arrive, it passes them through a Decorator Pattern mapper. Using tenant-specific YAML configurations (e.g., `data/pintar/mapper/kaskel_transaction.yaml`), it maps 1:1 fields and seamlessly packs any arbitrary unmapped source fields into a serialized JSON block called `free_text`.

### 3. The Destination

The mapped output is inserted into the `logs` table using an `INSERT IGNORE` transaction block, preventing duplicate primary key crashes and ensuring idempotent writes.

## Technology Stack

- **Core Engine:** Go (Golang)
- **Message Broker:** RabbitMQ
- **Source Databases Supported:** MySQL, SQL Server (Extensible to others)
- **Destination Databases Supported:** MySQL, SQL Server (Extensible to others) — Targeting the `logs` table
- **Configuration & Mapping:** YAML and JSON
- **Containerization:** Docker & Docker Compose

## How to Run Locally

### 1. Start the Infrastructure

Ensure Docker is installed and running. Start RabbitMQ and your local databases (if containerized) using Docker Compose:

```bash
docker compose up -d
```

### 2. Start the Consumer

Open a terminal and start the consumer service. It will connect to RabbitMQ and begin listening to the queue defined by the `--use-cases` flag.

```bash
go run cmd/consumer/main.go --config=data/config.yaml --use-cases="pintar"
```

### 3. Start the Producer

Open a separate terminal window and start the producer. Note: The `--use-cases` flag must perfectly match the consumer's flag so they communicate on the same RabbitMQ queue.

```bash
go run cmd/producer/main.go --config=data/config.yaml --use-cases="pintar"
```

## Configuration & Maintenance

### Adding a New Tenant or Use Case

To replicate a new table or add a new tenant, you do not need to alter the core Go code. Simply define the configuration files:

1. **Source Query:** Create a new SQL file at `data/{tenant}/queries/{use_case}.sql`. Make sure to include a `WHERE` clause with a `?` if you are utilizing high-water mark delta fetches (e.g., `WHERE created_date > ?`).
2. **Mapper Rule:** Create a matching YAML file at `data/{tenant}/mapper/{use_case}.yaml` defining how the source columns map to the `logs` table columns.

### Reseting Replication State

The producer and consumer track their synchronization progress using local JSON files (e.g., `data/state_producer_pintar_kaskel_transaction.json`).

- If you need to force a full data refresh from the beginning, delete these state files before running the engines:

```bash
rm data/state_*.json
```

## Air-Gapped / Offline Production Deployment

Because all dependencies and Go binaries compile directly into the Docker image, this stack is fully ready for offline deployment.

1. Save the image locally:

```bash
docker save -o go-dbreplicator.tar my-local-image:latest
```

2. Transfer the `.tar` and your configuration folders to the production server.
3. Load it on the production server:

```bash
docker load -i go-dbreplicator.tar
```

4. Run via Docker Compose:

```bash
docker compose up -d
```
