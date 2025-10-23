const express = require('express');
const path = require('path');
const fs = require('fs').promises;
const { generatePDF } = require('./services/pdfGenerator');
const { startConsumer } = require('./services/rabbitmqConsumer');
const { transformParserData } = require('./services/parserDataTransformer');

const app = express();
const PORT = 3000;
const TEMPLATES_DIR = path.join(__dirname, '../templates');

app.use(express.json());

// Get list of available templates
app.get('/api/templates', async (req, res) => {
  try {
    const files = await fs.readdir(TEMPLATES_DIR);
    const templates = files
      .filter(file => file.endsWith('.json'))
      .map(file => file.replace('.json', ''));
    res.json(templates);
  } catch (error) {
    console.error('Error listing templates:', error);
    res.status(500).json({ error: 'Failed to list templates' });
  }
});

// Get a specific template
app.get('/api/templates/:name', async (req, res) => {
  try {
    const templatePath = path.join(TEMPLATES_DIR, `${req.params.name}.json`);
    const templateData = await fs.readFile(templatePath, 'utf-8');
    res.json(JSON.parse(templateData));
  } catch (error) {
    console.error('Error loading template:', error);
    res.status(404).json({ error: 'Template not found' });
  }
});

// Save a template
app.post('/api/templates', async (req, res) => {
  try {
    const { name, template } = req.body;

    if (!name) {
      return res.status(400).json({ error: 'Template name is required' });
    }

    if (!template) {
      return res.status(400).json({ error: 'Template data is required' });
    }

    // Sanitize template name
    const safeName = name.replace(/[^a-zA-Z0-9_-]/g, '_');
    const templatePath = path.join(TEMPLATES_DIR, `${safeName}.json`);

    await fs.writeFile(templatePath, JSON.stringify(template, null, 2));
    res.json({ success: true, name: safeName });
  } catch (error) {
    console.error('Error saving template:', error);
    res.status(500).json({ error: 'Failed to save template' });
  }
});

// Generate PDF from template (backward compatible - direct template data)
app.post('/pdf', async (req, res) => {
  try {
    const { template_name, data, pagination } = req.body;

    if (!template_name) {
      return res.status(400).json({ error: 'template_name is required' });
    }

    if (!data) {
      return res.status(400).json({ error: 'data is required' });
    }

    // Generate PDF based on template_name, data, and optional pagination
    const pdfBuffer = await generatePDF(template_name, data, pagination);

    // Set headers to return PDF directly
    res.setHeader('Content-Type', 'application/pdf');
    res.setHeader('Content-Disposition', `attachment; filename="${template_name}_document.pdf"`);
    res.send(pdfBuffer);

  } catch (error) {
    console.error('Error generating PDF:', error);
    res.status(500).json({ error: error.message });
  }
});

// Generate PDF from parser output (new endpoint for parser integration)
app.post('/pdf/parser', async (req, res) => {
  try {
    const parserOutput = req.body;

    // Validate required fields
    if (!parserOutput.orgId) {
      return res.status(400).json({ error: 'orgId is required' });
    }

    if (!parserOutput.transactions || !Array.isArray(parserOutput.transactions)) {
      return res.status(400).json({ error: 'transactions array is required' });
    }

    // Transform parser data to template-compatible format
    console.log(`Processing parser data for orgId: ${parserOutput.orgId}`);
    const { template_name, data, pagination } = transformParserData(parserOutput);

    console.log(`Mapped to template: ${template_name}, transactions: ${parserOutput.transactions.length}`);

    // Generate PDF
    const pdfBuffer = await generatePDF(template_name, data, pagination);

    // Set headers to return PDF directly
    res.setHeader('Content-Type', 'application/pdf');
    res.setHeader('Content-Disposition', `attachment; filename="${template_name}_${parserOutput.orgId}_${Date.now()}.pdf"`);
    res.send(pdfBuffer);

  } catch (error) {
    console.error('Error generating PDF from parser data:', error);
    res.status(500).json({ error: error.message });
  }
});

app.get('/health', (req, res) => {
  res.json({ status: 'ok' });
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`PDF Generator Server is running on http://localhost:${PORT}`);
});

// Start RabbitMQ consumer
console.log('Starting RabbitMQ consumer...');
startConsumer().catch(error => {
  console.error('Failed to start RabbitMQ consumer:', error);
  // Don't exit - keep HTTP API running even if RabbitMQ is not available
});
