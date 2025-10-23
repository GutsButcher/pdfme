#!/usr/bin/env node

/**
 * Test Producer for RabbitMQ
 * Sends test requests from test_data/*.json files to the 'pdf_ready' queue
 */

const amqp = require('amqplib');
const fs = require('fs').promises;
const path = require('path');

const RABBITMQ_URL = process.env.RABBITMQ_URL || 'amqp://admin:admin123@localhost:5672';
const QUEUE_NAME = 'pdf_ready';
const TEST_DATA_DIR = './test_data';

/**
 * Send a message to RabbitMQ queue
 */
async function sendMessage(channel, message) {
  const messageBuffer = Buffer.from(JSON.stringify(message, null, 2));

  return channel.sendToQueue(QUEUE_NAME, messageBuffer, {
    persistent: true // Message survives broker restart
  });
}

/**
 * Main function
 */
async function main() {
  let connection;

  try {
    // Get command line arguments
    const args = process.argv.slice(2);

    if (args.length === 0) {
      console.log('Usage: node test_producer.js <test_file.json> [output_filename.pdf]');
      console.log('');
      console.log('Available test files:');
      const files = await fs.readdir(TEST_DATA_DIR);
      const testFiles = files.filter(f => f.endsWith('.json'));
      testFiles.forEach(f => console.log(`  - ${f}`));
      process.exit(1);
    }

    const testFile = args[0];
    const outputFilename = args[1] || null;

    // Read test data file
    const testFilePath = path.join(TEST_DATA_DIR, testFile);
    console.log(`Reading test data from: ${testFilePath}`);

    const fileContent = await fs.readFile(testFilePath, 'utf-8');
    const testData = JSON.parse(fileContent);

    // Add output filename if provided
    if (outputFilename) {
      testData.output_filename = outputFilename;
    }

    // Connect to RabbitMQ
    console.log('Connecting to RabbitMQ...');
    connection = await amqp.connect(RABBITMQ_URL);
    const channel = await connection.createChannel();

    // Assert queue exists
    await channel.assertQueue(QUEUE_NAME, {
      durable: true
    });

    console.log(`Queue '${QUEUE_NAME}' is ready`);

    // Send message
    console.log('Sending message to queue...');
    const sent = sendMessage(channel, testData);

    if (sent) {
      console.log('✓ Message sent successfully!');
      console.log('');
      console.log('Message content:');
      console.log(JSON.stringify(testData, null, 2));
    } else {
      console.error('✗ Failed to send message');
    }

    // Close channel and connection
    await channel.close();
    await connection.close();

    console.log('');
    console.log('Done! Check the output/ directory for the generated PDF.');

  } catch (error) {
    console.error('Error:', error.message);
    if (connection) {
      try {
        await connection.close();
      } catch (e) {
        // Ignore close errors
      }
    }
    process.exit(1);
  }
}

// Run the script
main();
