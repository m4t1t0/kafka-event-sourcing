# ksqlDB Streams & Tables

## Base Stream

The base stream maps to the event envelope structure. `DATA` is kept as `VARCHAR` because its schema varies by event type.

```sql
CREATE STREAM clients_stream (
  event_id VARCHAR,
  event_type VARCHAR,
  aggregate_id VARCHAR,
  occurred_at VARCHAR,
  version INT,
  data VARCHAR
) WITH (
  KAFKA_TOPIC='clients',
  VALUE_FORMAT='JSON'
);
```

## Derived Streams (per event type)

Each derived stream filters by `event_type` and extracts the relevant fields from the `DATA` JSON payload.

### client.created

```sql
CREATE STREAM client_created_stream AS
  SELECT
    aggregate_id AS client_id,
    EXTRACTJSONFIELD(data, '$.name') AS name,
    EXTRACTJSONFIELD(data, '$.email') AS email,
    EXTRACTJSONFIELD(data, '$.phone') AS phone,
    occurred_at
  FROM clients_stream
  WHERE event_type = 'client.created'
  EMIT CHANGES;
```

### client.updated

```sql
CREATE STREAM client_updated_stream AS
  SELECT
    aggregate_id AS client_id,
    EXTRACTJSONFIELD(data, '$.name') AS name,
    EXTRACTJSONFIELD(data, '$.email') AS email,
    EXTRACTJSONFIELD(data, '$.phone') AS phone,
    occurred_at
  FROM clients_stream
  WHERE event_type = 'client.updated'
  EMIT CHANGES;
```

### client.deleted

```sql
CREATE STREAM client_deleted_stream AS
  SELECT
    aggregate_id AS client_id,
    occurred_at
  FROM clients_stream
  WHERE event_type = 'client.deleted'
  EMIT CHANGES;
```

## Materialized Table (latest state per client)

Aggregates all events by `aggregate_id` to maintain the latest state. Supports pull queries for point lookups.

```sql
CREATE TABLE clients_latest AS
  SELECT
    aggregate_id AS client_id,
    LATEST_BY_OFFSET(EXTRACTJSONFIELD(data, '$.name')) AS name,
    LATEST_BY_OFFSET(EXTRACTJSONFIELD(data, '$.email')) AS email,
    LATEST_BY_OFFSET(event_type) AS last_event_type
  FROM clients_stream
  GROUP BY aggregate_id
  EMIT CHANGES;
```

## Querying

### Pull query (returns immediately, requires materialized table)

```sql
SELECT * FROM clients_latest WHERE client_id = 'some-uuid';
```

### Push query (continuous, works on streams and tables)

```sql
-- All events
SELECT * FROM clients_stream EMIT CHANGES;

-- Only created events
SELECT * FROM client_created_stream EMIT CHANGES;
```

### Reading from the beginning of a topic

By default, push queries start from the current offset. To read all existing events:

```sql
SET 'auto.offset.reset' = 'earliest';
SELECT * FROM clients_stream EMIT CHANGES;
```

## Cleanup

```sql
-- Drop derived objects first, then the base stream
DROP TABLE IF EXISTS clients_latest DELETE TOPIC;
DROP STREAM IF EXISTS client_deleted_stream DELETE TOPIC;
DROP STREAM IF EXISTS client_updated_stream DELETE TOPIC;
DROP STREAM IF EXISTS client_created_stream DELETE TOPIC;
DROP STREAM IF EXISTS clients_stream;
```

> Note: `DELETE TOPIC` removes the underlying Kafka topic created by ksqlDB for derived streams/tables. Omit it for the base stream to preserve the original `clients` topic.
