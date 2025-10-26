# Parser Service - RabbitMQ Integration Changes

## Summary

Parser service now integrates with RabbitMQ to enable event-driven processing. HTTP API still works.

## Files Added

### 1. `src/main/resources/application.properties`
- RabbitMQ connection settings
- Queue names configuration

### 2. `src/main/java/com/afs/parser/config/RabbitMQConfig.java`
- Spring AMQP configuration
- Queue declarations
- JSON message converter

### 3. `src/main/java/com/afs/parser/model/FileMessage.java`
- DTO for incoming messages from `parse_ready` queue
- Fields: filename, fileContent (base64), orgId

### 4. `src/main/java/com/afs/parser/service/RabbitMQConsumer.java`
- Listens to `parse_ready` queue
- Decodes base64 file content
- Calls existing StatementParser
- Sends result to `pdf_ready` queue

### 5. `src/main/java/com/afs/parser/service/RabbitMQProducer.java`
- Sends parsed data to `pdf_ready` queue
- Uses Spring RabbitTemplate

### 6. `Dockerfile`
- Multi-stage build (Maven + JRE)
- Runs on port 8080

### 7. `.dockerignore`
- Excludes build artifacts

## Files Modified

### `pom.xml`
Added dependencies:
- `spring-boot-starter-amqp` - RabbitMQ integration
- `jackson-databind` - JSON serialization

## Message Flow

**Consumes From**: `parse_ready` queue
```json
{
  "filename": "266_statement.txt",
  "file_content": "base64_encoded_content",
  "org_id": "266"
}
```

**Produces To**: `pdf_ready` queue
```json
{
  "orgId": "266",
  "name": "AHMED ADEL HUSAIN ALI",
  "cardNumber": "5117244499894536",
  "statementDate": "21/09/2025",
  "availableBalance": 1026.248,
  "transactions": [
    {
      "date": "06/09/2025",
      "postDate": "06/09/2025",
      "description": "Payment Received",
      "amountInBHD": 149.427,
      "cr": true
    }
  ]
}
```

## Environment Variables

Set in docker-compose.yml:
- `RABBITMQ_HOST=rabbitmq`
- `RABBITMQ_PORT=5672`
- `RABBITMQ_USERNAME=admin`
- `RABBITMQ_PASSWORD=admin123`

## Backward Compatibility

HTTP API still works:
```bash
POST /api/statement/upload
```

Both interfaces (HTTP and RabbitMQ) work simultaneously.

## Deployment

```bash
docker-compose up -d --build
```

Parser service will:
1. Connect to RabbitMQ on startup
2. Listen to `parse_ready` queue
3. Process files automatically
4. Send results to `pdf_ready` queue

## No Breaking Changes

- Existing StatementParser logic unchanged
- Existing HTTP controller unchanged
- Existing models unchanged
- Just added RabbitMQ integration layer
