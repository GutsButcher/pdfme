package com.afs.parser.service;

import com.afs.parser.model.EStatementRecord;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

@Service
public class RabbitMQProducer {

    private static final Logger log = LoggerFactory.getLogger(RabbitMQProducer.class);

    private final RabbitTemplate rabbitTemplate;

    @Value("${rabbitmq.queue.pdf-ready}")
    private String pdfReadyQueue;

    public RabbitMQProducer(RabbitTemplate rabbitTemplate) {
        this.rabbitTemplate = rabbitTemplate;
    }

    public void sendToPdfReady(EStatementRecord record) {
        try {
            rabbitTemplate.convertAndSend(pdfReadyQueue, record);
            log.info("✓ Sent to '{}' queue: orgId={}, transactions={}",
                pdfReadyQueue, record.getOrgId(), record.getTransactions().size());
        } catch (Exception e) {
            log.error("✗ Error sending to queue: {}", e.getMessage());
            throw e;
        }
    }
}
