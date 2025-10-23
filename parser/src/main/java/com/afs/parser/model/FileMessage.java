package com.afs.parser.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class FileMessage {
    private String filename;

    @JsonProperty("file_content")
    private String fileContent; // Base64 encoded

    @JsonProperty("org_id")
    private String orgId;

    public FileMessage() {
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

    public String getOrgId() {
        return orgId;
    }

    public void setOrgId(String orgId) {
        this.orgId = orgId;
    }
}
