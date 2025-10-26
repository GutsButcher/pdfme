package com.afs.parser.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class FileMessage {
    @JsonProperty("job_id")
    private String jobId;        // Pass through to next stage

    @JsonProperty("file_hash")
    private String fileHash;     // Pass through to next stage

    private String filename;

    @JsonProperty("file_content")
    private String fileContent;  // Base64 encoded

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

    public String getFileContent() {
        return fileContent;
    }

    public void setFileContent(String fileContent) {
        this.fileContent = fileContent;
    }
}
