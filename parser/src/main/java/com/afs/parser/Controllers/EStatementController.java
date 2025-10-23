package com.afs.parser.Controllers;
import com.afs.parser.model.EStatementRecord;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;

import static com.afs.parser.service.StatementParser.StatementParser;

@RestController
@RequestMapping("/api/statement")
public class EStatementController {


    @PostMapping("/upload")
    public ResponseEntity<EStatementRecord> uploadFile(@RequestParam("file") MultipartFile file) {
        try {

            // Save the uploaded file to a temporary location
            Path tempFile = Files.createTempFile("statement-", ".txt");
            Files.write(tempFile, file.getBytes());

            // Call your parser with the actual file path
            EStatementRecord record = StatementParser(tempFile.toString());

            return ResponseEntity.ok(record);

        } catch (IOException e) {
            return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).build();
        }
    }
}
