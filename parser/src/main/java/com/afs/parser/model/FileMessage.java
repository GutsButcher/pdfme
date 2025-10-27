package com.afs.parser.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class FileMessage {
    @JsonProperty("job_id")
    private String jobId;        // Pass through to next stage

    @JsonProperty("file_hash")
    private String fileHash;     // Pass through to next stage

    private String filename;

    @JsonProperty("redis_key")
    private String redisKey;     // Redis key where file content is stored

    @JsonProperty("file_size")
    private Long fileSize;       // File size in bytes

    public FileMessage() {
    }

    public String getJobId() {
        return jobId;
    }

    public void setJobId(String jobId) {
        this.jobId = jobId;
    }

    public String getFileHash() {
        return fileHash;
    }

    public void setFileHash(String fileHash) {
        this.fileHash = fileHash;
    }

    public String getFilename() {
        return filename;
    }

    public void setFilename(String filename) {
        this.filename = filename;
    }

    public String getRedisKey() {
        return redisKey;
    }

    public void setRedisKey(String redisKey) {
        this.redisKey = redisKey;
    }

    public Long getFileSize() {
        return fileSize;
    }

    public void setFileSize(Long fileSize) {
        this.fileSize = fileSize;
    }
}
