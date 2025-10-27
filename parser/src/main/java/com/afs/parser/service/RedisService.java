package com.afs.parser.service;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.concurrent.TimeUnit;

@Service
public class RedisService {
    private static final Logger logger = LoggerFactory.getLogger(RedisService.class);

    @Autowired
    private RedisTemplate<String, byte[]> redisTemplate;

    /**
     * Get file content from Redis blob storage
     * @param fileHash File hash (used as part of Redis key)
     * @return File content as byte array
     */
    public byte[] getFileBlob(String fileHash) {
        String redisKey = "blob:" + fileHash;
        logger.info("Fetching file from Redis: {}", redisKey);

        byte[] content = redisTemplate.opsForValue().get(redisKey);

        if (content == null) {
            logger.error("File not found in Redis: {} (may have expired)", redisKey);
            throw new RuntimeException("File not found in Redis: " + redisKey);
        }

        logger.info("Retrieved {} bytes from Redis", content.length);
        return content;
    }

    /**
     * Delete file content from Redis after processing
     * @param fileHash File hash
     */
    public void deleteFileBlob(String fileHash) {
        String redisKey = "blob:" + fileHash;
        logger.info("Deleting file from Redis: {}", redisKey);

        Boolean deleted = redisTemplate.delete(redisKey);

        if (Boolean.TRUE.equals(deleted)) {
            logger.info("Successfully deleted blob from Redis: {}", redisKey);
        } else {
            logger.warn("Failed to delete blob from Redis (may not exist): {}", redisKey);
        }
    }

    /**
     * Check if file blob exists in Redis
     * @param fileHash File hash
     * @return true if exists, false otherwise
     */
    public boolean blobExists(String fileHash) {
        String redisKey = "blob:" + fileHash;
        Boolean exists = redisTemplate.hasKey(redisKey);
        return Boolean.TRUE.equals(exists);
    }
}
