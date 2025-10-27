package com.afs.parser.service;

import com.afs.parser.model.EStatementRecord;
import com.afs.parser.model.FileMessage;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.amqp.rabbit.annotation.RabbitListener;
import org.springframework.stereotype.Service;

import java.nio.file.Files;
import java.nio.file.Path;

@Service
public class RabbitMQConsumer {

    private static final Logger log = LoggerFactory.getLogger(RabbitMQConsumer.class);

    private final RabbitMQProducer producer;
    private final RedisService redisService;

    public RabbitMQConsumer(RabbitMQProducer producer, RedisService redisService) {
        this.producer = producer;
        this.redisService = redisService;
    }

    @RabbitListener(queues = "${rabbitmq.queue.parse-ready}")
    public void consumeFromParseReady(FileMessage message) {
        try {
            log.info("\n[→] Received message from parse_ready");
            log.info("  Job ID: {}", message.getJobId());
            log.info("  File Hash: {}", message.getFileHash());
            log.info("  Filename: {}", message.getFilename());
            log.info("  Redis Key: {}", message.getRedisKey());
            log.info("  File Size: {} bytes ({} MB)", message.getFileSize(),
                String.format("%.2f", message.getFileSize() / 1024.0 / 1024.0));

            // Download file from Redis
            log.info("  [↓] Downloading from Redis...");
            byte[] fileBytes = redisService.getFileBlob(message.getFileHash());
            log.info("  [✓] Downloaded {} bytes from Redis", fileBytes.length);

            // Write to temporary file
            Path tempFile = Files.createTempFile("statement-", ".txt");
            Files.write(tempFile, fileBytes);

            // Parse the file
            log.info("  [*] Parsing file...");
            EStatementRecord record = StatementParser.StatementParser(tempFile.toString());

            // Clean up temp file
            Files.deleteIfExists(tempFile);

            // Delete file from Redis (cleanup!)
            log.info("  [×] Deleting from Redis...");
            redisService.deleteFileBlob(message.getFileHash());
            log.info("  [✓] Redis cleanup complete");

            // Pass through job_id and file_hash (important for storage service!)
            record.setJobId(message.getJobId());
            record.setFileHash(message.getFileHash());

            log.info("  [✓] Parsed: orgId={}, name={}, transactions={}",
                record.getOrgId(), record.getName(), record.getTransactions().size());

            // Send to pdf_ready queue
            producer.sendToPdfReady(record);

            log.info("[✓] Processing complete\n");

        } catch (Exception e) {
            log.error("[✗] Error processing message: {}", e.getMessage(), e);
            // Note: If processing fails, file remains in Redis (1h TTL auto-cleanup)
            // In production, consider using dead-letter queue
        }
    }
}
