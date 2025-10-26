const amqp = require('amqplib');
const { generatePDF } = require('./pdfGenerator');
const { transformParserData } = require('./parserDataTransformer');

const RABBITMQ_URL = process.env.RABBITMQ_URL || 'amqp://admin:admin123@rabbitmq:5672';
const CONSUME_QUEUE = 'pdf_ready';
const PRODUCE_QUEUE = 'storage_ready';
const DEFAULT_BUCKET = process.env.DEFAULT_BUCKET || 'pdfs';

let globalChannel = null;

/**
 * Generate random 6-digit suffix
 */
function generateRandomSuffix() {
  return Math.floor(100000 + Math.random() * 900000).toString();
}

/**
 * Connect to RabbitMQ with retry logic
 */
async function connectWithRetry(maxRetries = 10, delay = 5000) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      console.log(`Attempting to connect to RabbitMQ (attempt ${i + 1}/${maxRetries})...`);
      const connection = await amqp.connect(RABBITMQ_URL);
      console.log('✓ Connected to RabbitMQ');
      return connection;
    } catch (error) {
      console.error(`Failed to connect to RabbitMQ: ${error.message}`);
      if (i < maxRetries - 1) {
        console.log(`Retrying in ${delay / 1000} seconds...`);
        await new Promise(resolve => setTimeout(resolve, delay));
      }
    }
  }
  throw new Error('Failed to connect to RabbitMQ after max retries');
}

/**
 * Send message to storage_ready queue
 */
async function sendToStorageQueue(storageMessage) {
  if (!globalChannel) {
    throw new Error('RabbitMQ channel not initialized');
  }

  // Assert storage_ready queue exists
  await globalChannel.assertQueue(PRODUCE_QUEUE, {
    durable: true
  });

  // Send message
  const messageBuffer = Buffer.from(JSON.stringify(storageMessage));
  globalChannel.sendToQueue(PRODUCE_QUEUE, messageBuffer, {
    persistent: true
  });

  console.log(`✓ Sent to '${PRODUCE_QUEUE}' queue: ${storageMessage.filename}`);
}

/**
 * Process a PDF generation request from queue
 */
async function processPDFRequest(message) {
  try {
    const request = JSON.parse(message.content.toString());

    // Extract pass-through fields (critical for storage service!)
    const jobId = request.jobId || request.job_id;
    const fileHash = request.fileHash || request.file_hash;

    if (!jobId || !fileHash) {
      throw new Error('job_id and file_hash are required fields');
    }

    console.log(`  Job ID: ${jobId.substring(0, 8)}...`);
    console.log(`  File Hash: ${fileHash.substring(0, 12)}...`);

    let template_name, data, pagination, bucket_name, filename_prefix;

    // Check if this is parser format (has orgId field) or direct format
    if (request.orgId && request.transactions) {
      // Parser format - transform it first
      console.log(`Processing parser data: orgId=${request.orgId}, transactions=${request.transactions.length}`);

      const transformed = transformParserData(request);
      template_name = transformed.template_name;
      data = transformed.data;
      pagination = transformed.pagination;
      bucket_name = request.bucket_name || DEFAULT_BUCKET;
      filename_prefix = `statement_${request.orgId}`;

      console.log(`✓ Transformed to template: ${template_name}`);
    } else {
      // Direct format (backward compatible)
      template_name = request.template_name;
      data = request.data;
      pagination = request.pagination;
      bucket_name = request.bucket_name;
      filename_prefix = request.filename_prefix;

      console.log(`Processing PDF request: template=${template_name}`);

      // Validate request
      if (!template_name) {
        throw new Error('template_name is required');
      }
      if (!data) {
        throw new Error('data is required');
      }
    }

    // Generate PDF
    const pdfBuffer = await generatePDF(template_name, data, pagination);
    console.log(`✓ PDF generated (${pdfBuffer.length} bytes)`);

    // Encode to base64
    const base64Content = pdfBuffer.toString('base64');
    console.log(`✓ Encoded to base64 (${base64Content.length} chars)`);

    // Generate filename with timestamp
    const prefix = filename_prefix || template_name;
    const timestamp = Date.now();
    const filename = `${prefix}_${timestamp}.pdf`;

    // Prepare storage message (MUST include job_id and file_hash!)
    const storageMessage = {
      job_id: jobId,        // Pass through for DB update
      file_hash: fileHash,  // Pass through for Redis update
      bucket_name: bucket_name || DEFAULT_BUCKET,
      filename: filename,
      file_content: base64Content
    };

    // Send to storage_ready queue
    await sendToStorageQueue(storageMessage);

    console.log(`✓ PDF processing complete: ${filename}`);
    return { success: true, filename };
  } catch (error) {
    console.error(`✗ Error processing PDF request: ${error.message}`);
    throw error;
  }
}

/**
 * Start RabbitMQ consumer
 */
async function startConsumer() {
  try {
    // Connect to RabbitMQ
    const connection = await connectWithRetry();

    // Create channel
    const channel = await connection.createChannel();
    globalChannel = channel;
    console.log('✓ Channel created');

    // Assert pdf_ready queue exists
    await channel.assertQueue(CONSUME_QUEUE, {
      durable: true // Queue survives broker restart
    });
    console.log(`✓ Queue '${CONSUME_QUEUE}' is ready`);

    // Assert storage_ready queue exists
    await channel.assertQueue(PRODUCE_QUEUE, {
      durable: true
    });
    console.log(`✓ Queue '${PRODUCE_QUEUE}' is ready`);

    // Set prefetch to 1 - process one message at a time
    await channel.prefetch(1);

    console.log(`[*] Waiting for messages in queue '${CONSUME_QUEUE}'...`);
    console.log(`[*] Will produce to queue '${PRODUCE_QUEUE}'\n`);

    // Start consuming messages
    channel.consume(CONSUME_QUEUE, async (message) => {
      if (message) {
        try {
          console.log('\n[→] Received message from pdf_ready');
          await processPDFRequest(message);

          // Acknowledge message
          channel.ack(message);
          console.log('[✓] Message acknowledged\n');
        } catch (error) {
          console.error('[✗] Error processing message:', error.message);

          // Acknowledge to avoid infinite loop
          // In production, consider using dead-letter queue
          channel.ack(message);
          console.log('[!] Message acknowledged despite error\n');
        }
      }
    }, {
      noAck: false // Manual acknowledgment
    });

    // Handle connection close
    connection.on('close', () => {
      console.error('RabbitMQ connection closed. Exiting...');
      process.exit(1);
    });

    // Handle connection error
    connection.on('error', (err) => {
      console.error('RabbitMQ connection error:', err.message);
    });

  } catch (error) {
    console.error('Failed to start RabbitMQ consumer:', error.message);
    process.exit(1);
  }
}

module.exports = { startConsumer };
