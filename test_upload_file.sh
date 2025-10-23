#!/bin/bash
#
# Test File Uploader
# Uploads a file to MinIO nonparsed_files bucket to trigger the workflow
#

FILE=$1
ORGID=${2:-"266"}

if [ -z "$FILE" ]; then
  echo "Usage: ./test_upload_file.sh <file> [orgId]"
  echo ""
  echo "Example:"
  echo "  ./test_upload_file.sh test_data/266003.txt 266"
  echo ""
  echo "This will:"
  echo "  1. Upload file to MinIO (nonparsed_files bucket)"
  echo "  2. File Watcher detects it (within 10s)"
  echo "  3. Sends to parse_ready queue"
  echo "  4. Parser processes it"
  echo "  5. Sends to pdf_ready queue"
  echo "  6. PDF Generator creates PDF"
  echo "  7. Sends to storage_ready queue"
  echo "  8. Storage uploads to MinIO (pdfs bucket)"
  exit 1
fi

if [ ! -f "$FILE" ]; then
  echo "Error: File not found: $FILE"
  exit 1
fi

BASENAME=$(basename "$FILE")
TARGET="${ORGID}_${BASENAME}"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Uploading: $FILE"
echo "      As: $TARGET"
echo "  Bucket: uploads"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Copy file into MinIO container and upload
docker cp "$FILE" pdfme-minio:/tmp/upload_file
docker exec pdfme-minio mc cp /tmp/upload_file "local/uploads/${TARGET}"

if [ $? -eq 0 ]; then
  echo "✓ File uploaded successfully!"
  echo ""
  echo "Monitor workflow:"
  echo "  docker-compose logs -f file-watcher"
  echo "  docker-compose logs -f parser-service"
  echo "  docker-compose logs -f pdf-generator"
  echo "  docker-compose logs -f storage-service"
  echo ""
  echo "Check result in MinIO Console:"
  echo "  http://localhost:9001 (minioadmin / minioadmin)"
  echo "  Navigate to: pdfs bucket"
else
  echo "✗ Upload failed!"
  exit 1
fi
