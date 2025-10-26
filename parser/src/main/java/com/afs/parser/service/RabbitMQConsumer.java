package com.afs.parser.service;

import com.afs.parser.model.EStatementRecord;
import com.afs.parser.model.FileMessage;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.amqp.rabbit.annotation.RabbitListener;
import org.springframework.stereotype.Service;

import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Base64;

@Service
public class RabbitMQConsumer {

    private static final Logger log = LoggerFactory.getLogger(RabbitMQConsumer.class);

    private final RabbitMQProducer producer;

    public RabbitMQConsumer(RabbitMQProducer producer) {
        this.producer = producer;
    }

    @RabbitListener(queues = "${rabbitmq.queue.parse-ready}")
    public void consumeFromParseReady(FileMessage message) {
        try {
            log.info("\n[→] Received message from parse_ready");
            log.info("  Job ID: {}", message.getJobId());
            log.info("  File Hash: {}", message.getFileHash());
            log.info("  Filename: {}", message.getFilename());

            // Decode base64 file content
            byte[] fileBytes = Base64.getDecoder().decode(message.getFileContent());
            log.info("  Decoded: {} bytes", fileBytes.length);

            // Write to temporary file
            Path tempFile = Files.createTempFile("statement-", ".txt");
            Files.write(tempFile, fileBytes);

            // Parse the file
            log.info("  Parsing file...");
            EStatementRecord record = StatementParser.StatementParser(tempFile.toString());

            // Clean up temp file
            Files.deleteIfExists(tempFile);

            // Pass through job_id and file_hash (important for storage service!)
            record.setJobId(message.getJobId());
            record.setFileHash(message.getFileHash());

            log.info("✓ Parsed: orgId={}, name={}, transactions={}",
                record.getOrgId(), record.getName(), record.getTransactions().size());

            // Send to pdf_ready queue
            producer.sendToPdfReady(record);

            log.info("[✓] Processing complete\n");

        } catch (Exception e) {
            log.error("[✗] Error processing message: {}", e.getMessage(), e);
            // In production, consider using dead-letter queue
        }
    }
}
