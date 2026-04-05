# CLAUDE.md — Kafka Event Sourcing POC

## Project Overview

A proof-of-concept exploring **event sourcing** with Kafka as the source of truth and PostgreSQL as the read database via projections (CQRS pattern). Two microservices communicate through domain events published to Kafka topics.

### Architecture

```
[HTTP Request] → [Command Handler] → [Kafka Producer] → [Kafka Topic]
                                                              ↓
[HTTP Query] ← [PostgreSQL Read DB] ← [Projector/Consumer] ←─┘
```

- **Kafka** is the event log / source of truth. Events are immutable and append-only.
- **PostgreSQL** holds read-model projections rebuilt from events. It is NOT the source of truth.
- Each service owns both its command side (publish events) and query side (consume events, update projection, serve reads).
- The projector writes the projection AND the Kafka offset in a single PostgreSQL transaction for exactly-once projection semantics.

### Services

| Service | HTTP Port | Kafka Topic | Consumer Group |
|---------|-----------|-------------|----------------|
| client-service | :8081 | `clients` | `client-projector` |
| order-service | :8082 | `orders` | `order-projector` |

The order service also consumes from the `clients` topic (group: `order-client-projector`) to denormalize client data into order projections.

---

## Tech Stack

- **Language:** Go (1.22+ required for `net/http` routing with path values)
- **Kafka client:** `github.com/confluentinc/confluent-kafka-go/v2/kafka`
- **PostgreSQL ORM:** `gorm.io/gorm` + `gorm.io/driver/postgres`
- **UUIDs:** `github.com/google/uuid`
- **Infrastructure:** Docker Compose (Confluent Kafka 7.5, PostgreSQL 16, ksqlDB, Kafka UI)

---

## Project Structure

```
kafka-event-sourcing/
├── docker-compose.yml
├── Dockerfile                       # Multi-stage build (shared by both services via SERVICE arg)
├── .dockerignore
├── Makefile
├── cmd/
│   ├── client-service/main.go       # Entrypoint: HTTP server + consumer
│   └── order-service/main.go        # Entrypoint: HTTP server + order & client consumers
├── internal/
│   ├── events/
│   │   ├── envelope.go              # Event envelope: metadata wrapper serialized to Kafka
│   │   ├── client_events.go         # ClientCreated, ClientUpdated, ClientDeleted
│   │   └── order_events.go          # OrderPlaced, OrderConfirmed, OrderCancelled
│   ├── kafka/
│   │   ├── producer.go              # Confluent Kafka producer wrapper
│   │   └── consumer.go              # Confluent Kafka consumer with manual offset commit
│   ├── projection/
│   │   └── offset.go                # Offset tracking model + upsert within GORM tx
│   ├── client/
│   │   ├── model.go                 # GORM read model: Client
│   │   ├── projector.go             # Consumes client events → writes projection
│   │   └── handler.go               # HTTP handlers (Create=command, Get/List=query)
│   └── order/
│       ├── model.go                 # GORM read model: Order, OrderItem
│       ├── projector.go             # Consumes order events → writes projection
│       ├── client_projector.go      # Consumes client events → denormalizes client_name into orders
│       └── handler.go               # HTTP handlers (Place/Confirm/Cancel=command, Get/List=query)
```

---

## Key Design Decisions

### Event Envelope

Every domain event is wrapped in an `events.Envelope` before publishing to Kafka:

```go
type Envelope struct {
    EventID     string          // UUID, unique per event
    EventType   string          // e.g. "client.created", "order.placed"
    AggregateID string          // Entity ID, used as Kafka message key
    OccurredAt  time.Time       // UTC timestamp
    Version     int             // Aggregate version for ordering
    Data        json.RawMessage // Serialized domain event payload
}
```

The `AggregateID` is used as the Kafka message key, which guarantees ordering per entity within a partition.

### Projection + Offset Atomicity

The projector writes the read model update AND saves the Kafka offset in a single GORM transaction:

```go
db.Transaction(func(tx *gorm.DB) error {
    applyProjection(tx, event)                                    // Update read model
    projection.SaveOffset(tx, consumerGroup, topic, partition, offset)  // Track progress
    return nil
})
```

The offset table includes `consumer_group` in its unique constraint, so multiple consumer groups (e.g. `client-projector` and `order-client-projector`) can independently track offsets for the same topic+partition without conflicts.

Kafka auto-commit is disabled (`enable.auto.commit: false`). The consumer commits the offset to Kafka only AFTER the PostgreSQL transaction succeeds. This means:
- If the process crashes mid-projection, the event will be reprocessed (at-least-once from Kafka's perspective).
- The PostgreSQL offset check prevents duplicate application (effectively exactly-once on the read side).

### CQRS Split

Command handlers (Create, Update, Delete) ONLY publish events to Kafka. They return `202 Accepted`.
Query handlers (Get, List) ONLY read from PostgreSQL projections.
There is an inherent eventual consistency gap — a write followed by an immediate read may not reflect the change yet.

### Soft Deletes

Client deletion uses GORM soft deletes (`deleted_at` timestamp). The `ClientDeleted` event sets this field rather than removing the row, preserving audit history in the read model.

---

## Kafka Topics

| Topic | Key | Partitions | Events |
|-------|-----|------------|--------|
| `clients` | `client_id` | 3 | `client.created`, `client.updated`, `client.deleted` |
| `orders` | `order_id` | 3 | `order.placed`, `order.confirmed`, `order.cancelled` |

Topics are created by the `kafka-init` container in docker-compose. Auto-create is disabled.

---

## Event Catalog

### Client Events

**`client.created`** — A new client was registered.
```json
{ "client_id": "uuid", "name": "string", "email": "string", "phone": "string" }
```

**`client.updated`** — Client details were modified.
```json
{ "client_id": "uuid", "name": "string", "email": "string", "phone": "string" }
```

**`client.deleted`** — Client was soft-deleted.
```json
{ "client_id": "uuid" }
```

### Order Events

**`order.placed`** — A new order was submitted.
```json
{
  "order_id": "uuid",
  "client_id": "uuid",
  "items": [{ "product_id": "string", "name": "string", "quantity": 1, "price": 9.99 }],
  "total": 9.99
}
```

**`order.confirmed`** — An order was confirmed.
```json
{ "order_id": "uuid", "confirmed_by": "string" }
```

**`order.cancelled`** — An order was cancelled.
```json
{ "order_id": "uuid", "reason": "string" }
```

---

## API Endpoints

### Client Service (:8081)

| Method | Path | Type | Description |
|--------|------|------|-------------|
| POST | `/clients` | Command | Publish `client.created` event. Returns `202` with `client_id`. |
| GET | `/clients` | Query | List all active clients from projection. |
| GET | `/clients/{id}` | Query | Get a single client by ID from projection. |

**Pending endpoints to implement:**
- `PUT /clients/{id}` — Publish `client.updated` event
- `DELETE /clients/{id}` — Publish `client.deleted` event

### Order Service (:8082)

| Method | Path | Type | Description |
|--------|------|------|-------------|
| POST | `/orders` | Command | Publish `order.placed` event. Returns `202` with `order_id`. |
| GET | `/orders` | Query | List all orders (with items) from projection. |
| GET | `/orders/{id}` | Query | Get a single order by ID (with items) from projection. |
| PUT | `/orders/{id}/confirm` | Command | Publish `order.confirmed` event. |
| PUT | `/orders/{id}/cancel` | Command | Publish `order.cancelled` event. |

---

## Running the Project

### Everything in Docker (recommended)

```bash
# Build and start all containers (infra + services)
make up

# Test
curl -X POST http://localhost:8081/clients \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "email": "alice@example.com", "phone": "+1234567890"}'

# Wait a moment for projection, then query
curl http://localhost:8081/clients
```

### Local development (services on host, infra in Docker)

```bash
# Start only infrastructure (Kafka, Zookeeper, PostgreSQL, Kafka UI, ksqlDB)
make infra

# Run services locally (in separate terminals)
make run-client
make run-order

# Or with hot reload via air
make dev-client
make dev-order
```

### Docker setup

Both services share a single multi-stage `Dockerfile`. The `SERVICE` build arg selects which entrypoint to compile:

```dockerfile
ARG SERVICE
RUN go build -tags musl -o /bin/service ./cmd/${SERVICE}
```

The `confluent-kafka-go` library requires `librdkafka` (C dependency), so the builder stage installs `gcc`, `musl-dev`, and `librdkafka-dev`, and the runtime stage includes `librdkafka`.

### Makefile targets

| Target | Description |
|--------|-------------|
| `make up` | Build and start everything in Docker |
| `make infra` | Start only infrastructure containers |
| `make infra-down` | Stop all containers |
| `make stop` | Stop containers and remove volumes |
| `make run-client` | Run client-service locally |
| `make run-order` | Run order-service locally |
| `make dev-client` | Run client-service with hot reload (air) |
| `make dev-order` | Run order-service with hot reload (air) |
| `make build` | Build both service binaries |
| `make migrate-up` | Run pending DB migrations |
| `make migrate-down` | Roll back last migration |
| `make clean` | Remove build artifacts |

---

## What's Already Built

- [x] Docker Compose with Kafka (Confluent), Zookeeper, PostgreSQL, topic init, and both services containerized
- [x] Multi-stage Dockerfile shared by both services (with librdkafka for confluent-kafka-go)
- [x] Kafka healthcheck + `service_healthy` dependency for reliable startup ordering
- [x] Event envelope with serialization/deserialization
- [x] All domain event types (client + order)
- [x] Kafka producer wrapper (keyed by aggregate ID, acks=all)
- [x] Kafka consumer wrapper (manual commit, context-aware shutdown)
- [x] Offset tracking with atomic upsert in GORM transactions
- [x] Client GORM read model with soft deletes
- [x] Client projector (handles created/updated/deleted events)
- [x] Client HTTP handler (Create command, Get/List queries)
- [x] Client service main.go with graceful shutdown
- [x] Order service (`cmd/order-service/main.go`) with graceful shutdown
- [x] Order GORM models (`internal/order/model.go`) — Order and OrderItem tables
- [x] Order projector (`internal/order/projector.go`) — handles placed/confirmed/cancelled events
- [x] Order HTTP handlers (`internal/order/handler.go`) — Place/Confirm/Cancel commands, Get/List queries
- [x] Cross-service consumption: order client projector subscribes to `clients` topic to denormalize client name into orders
- [x] SQL migration for orders and order_items tables (`000002_create_orders`)

## What Needs to Be Built

- [ ] Client update endpoint (`PUT /clients/{id}`)
- [ ] Client delete endpoint (`DELETE /clients/{id}`)
- [ ] Event replay mechanism: ability to rebuild projections from scratch by resetting consumer offsets
- [ ] Health check endpoints (`GET /health`)
- [ ] Integration tests using testcontainers-go
- [ ] Structured logging (replace `log.Printf`)

---

## Important Patterns to Follow

1. **Never write to PostgreSQL directly from a command handler.** Always publish an event to Kafka and let the projector handle the write.
2. **Every projector must wrap its work in a GORM transaction** that includes both the read model update and the offset save.
3. **Events are past tense and immutable.** `ClientCreated`, not `CreateClient`. Never modify a published event's schema — add new event types instead.
4. **Use the aggregate ID as the Kafka message key** to guarantee per-entity ordering within a partition.
5. **Return 202 Accepted from commands**, not 201 Created, because the projection hasn't happened yet.
6. **Handle unknown event types gracefully** in projectors — log and skip, don't crash. This allows evolving the event catalog without breaking existing consumers.

---

## Environment Variables

| Variable | Default (local) | Docker override | Description |
|----------|----------------|-----------------|-------------|
| `KAFKA_BROKER` | `localhost:9092` | `kafka:29092` | Kafka bootstrap server |
| `POSTGRES_DSN` | `host=localhost user=eventsrc password=eventsrc dbname=projections port=5432 sslmode=disable` | `host=postgres ...` | PostgreSQL connection string |
| `HTTP_ADDR` | `:8081` (client) / `:8082` (order) | same | HTTP listen address |

When running in Docker, the services connect via container hostnames (`kafka`, `postgres`) on internal ports. When running locally, they connect via `localhost` on the ports exposed by Docker.
