const express = require('express');
const { generatePDF } = require('./services/pdfGenerator');

const app = express();
const PORT = 3000;

app.use(express.json());

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
});
