const express = require('express');
const path = require('path');
const fs = require('fs').promises;
const { generatePDF } = require('./services/pdfGenerator');

const app = express();
const PORT = 3000;
const TEMPLATES_DIR = path.join(__dirname, '../templates');

app.use(express.json());
app.use(express.static(path.join(__dirname, '../public')));

// Serve the designer UI
app.get('/designer', (req, res) => {
  res.sendFile(path.join(__dirname, '../public/designer.html'));
});

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

// Generate PDF from template
app.post('/pdf', async (req, res) => {
  try {
    const { template_name, data } = req.body;

    if (!template_name) {
      return res.status(400).json({ error: 'template_name is required' });
    }

    if (!data) {
      return res.status(400).json({ error: 'data is required' });
    }

    // Generate PDF based on template_name and data
    const pdfBuffer = await generatePDF(template_name, data);

    // Set headers to return PDF directly
    res.setHeader('Content-Type', 'application/pdf');
    res.setHeader('Content-Disposition', `attachment; filename="${template_name}_document.pdf"`);
    res.send(pdfBuffer);

  } catch (error) {
    console.error('Error generating PDF:', error);
    res.status(500).json({ error: error.message });
  }
});

app.get('/health', (req, res) => {
  res.json({ status: 'ok' });
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`PDF Generator Server is running on http://localhost:${PORT}`);
  console.log(`Template Designer available at http://localhost:${PORT}/designer`);
});
