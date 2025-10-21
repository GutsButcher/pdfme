const { generate } = require('@pdfme/generator');
const { text, image } = require('@pdfme/schemas');
const fs = require('fs').promises;
const path = require('path');

const TEMPLATES_DIR = path.join(__dirname, '../../templates');

/**
 * Load template based on template name
 * @param {string} templateName - Name of the template file (without .json extension)
 * @returns {Promise<Object>} Template object
 */
async function loadTemplate(templateName) {
  const templatePath = path.join(TEMPLATES_DIR, `${templateName}.json`);

  try {
    const templateData = await fs.readFile(templatePath, 'utf-8');
    return JSON.parse(templateData);
  } catch (error) {
    if (error.code === 'ENOENT') {
      throw new Error(`Template not found: ${templateName}`);
    }
    throw new Error(`Failed to load template: ${error.message}`);
  }
}

/**
 * Generate PDF using pdfme
 * @param {string} templateName - Name of the template to use
 * @param {Object} data - Data to populate in the template
 * @returns {Promise<Buffer>} PDF buffer
 */
async function generatePDF(templateName, data) {
  // Load the template based on template name
  const template = await loadTemplate(templateName);

  // Convert custom fonts from base64 strings to Buffers
  if (template.fonts) {
    const fonts = {};
    for (const [fontName, fontData] of Object.entries(template.fonts)) {
      // If font is a base64 string, convert to Buffer
      if (typeof fontData === 'string') {
        fonts[fontName] = Buffer.from(fontData, 'base64');
      } else {
        fonts[fontName] = fontData;
      }
    }
    template.fonts = fonts;
  }

  // Prepare inputs array - pdfme expects an array of objects
  // Each object represents data for one page
  const inputs = Array.isArray(data) ? data : [data];

  try {
    // Generate PDF using pdfme
    const pdf = await generate({
      template,
      inputs,
      plugins: { text, image },
    });

    return Buffer.from(pdf);
  } catch (error) {
    throw new Error(`PDF generation failed: ${error.message}`);
  }
}

module.exports = {
  generatePDF,
  loadTemplate,
};
