const { generate } = require('@pdfme/generator');
const { text, image } = require('@pdfme/schemas');
const { PDFDocument } = require('pdf-lib');
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
 * Group fields by Y position to detect rows
 * @param {Object} baseSchema - The base page schema
 * @param {string} itemPrefix - Field prefix to look for
 * @returns {Array} Array of row groups, each containing fields at the same Y position
 */
function groupFieldsByPosition(baseSchema, itemPrefix) {
  const Y_TOLERANCE = 1.0; // Fields within 1 unit are considered same row

  // Find all fields with the prefix and their Y positions
  const prefixFields = Object.entries(baseSchema)
    .filter(([key]) => key.startsWith(itemPrefix))
    .map(([key, field]) => ({
      key,
      field,
      y: field.position.y
    }));

  if (prefixFields.length === 0) return [];

  // Group by Y position
  const rows = [];
  const processedYs = new Set();

  prefixFields.forEach(({ key, field, y }) => {
    // Check if this Y position is already processed (within tolerance)
    const existingRow = rows.find(row =>
      Math.abs(row.y - y) <= Y_TOLERANCE
    );

    if (existingRow) {
      existingRow.fields.push({ key, field });
    } else {
      rows.push({
        y: y,
        fields: [{ key, field }]
      });
    }
  });

  // Sort rows by Y position (top to bottom)
  rows.sort((a, b) => a.y - b.y);

  return rows;
}

/**
 * Generate schemas for pagination using position-based detection
 * @param {Object} baseSchema - The base page schema
 * @param {Object} data - The data object
 * @param {Object} pagination - Pagination settings {itemPrefix, itemsPerPage}
 * @returns {Array} Array of schemas, one per page
 */
function generatePaginatedSchemas(baseSchema, data, pagination) {
  const { itemPrefix, itemsPerPage } = pagination;

  // Group fields by Y position to detect rows
  const rows = groupFieldsByPosition(baseSchema, itemPrefix);

  if (rows.length === 0) {
    // No repeating items, return single page
    return [baseSchema];
  }

  const totalRows = rows.length;
  const pagesNeeded = Math.ceil(totalRows / itemsPerPage);

  console.log(`Pagination: Detected ${totalRows} rows by position, creating ${pagesNeeded} page(s)`);

  // Single page schema - pdfme will handle multiple pages via inputs array
  // We keep the original schema as-is
  return [baseSchema];
}

/**
 * Generate PDF using pdfme
 * @param {string} templateName - Name of the template to use
 * @param {Object} data - Data to populate in the template
 * @param {Object} pagination - Optional pagination settings {itemPrefix, itemsPerPage}
 * @returns {Promise<Buffer>} PDF buffer
 */
async function generatePDF(templateName, data, pagination = null) {
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

  // Note: No need to modify template.schemas or basePdf
  // The basePdf should always have 1 page
  // pdfme will automatically create multiple pages based on the inputs array

  // Prepare inputs array - pdfme expects an array of objects
  // Each object represents data for one page
  let inputs;

  if (pagination && pagination.itemPrefix && pagination.itemsPerPage) {
    // Use position-based detection
    const { itemPrefix, itemsPerPage } = pagination;
    const baseSchema = template.schemas[0];

    // Detect template structure from schema (how many rows per page)
    const templateRows = groupFieldsByPosition(baseSchema, itemPrefix);

    if (templateRows.length === 0) {
      inputs = [data];
    } else {
      // Count items in DATA (not schema)
      const dataKeys = Object.keys(data).filter(key => key.startsWith(itemPrefix));
      const uniqueRows = new Set();

      // Group data keys by their row identifier (position in template)
      dataKeys.forEach(key => {
        // Try to find which template row this key matches
        const matchingRow = templateRows.findIndex(row =>
          row.fields.some(f => {
            // Check if field name pattern matches (same suffix)
            const templateSuffix = f.key.replace(/\d+/g, '');
            const dataSuffix = key.replace(/\d+/g, '');
            return templateSuffix === dataSuffix;
          })
        );

        if (matchingRow >= 0) {
          // Extract row identifier from key (the number part)
          const match = key.match(/(\d+)/);
          if (match) {
            uniqueRows.add(parseInt(match[1]));
          }
        }
      });

      const totalDataRows = uniqueRows.size;
      const rowsPerPage = templateRows.length; // Use itemsPerPage or template row count
      const pagesNeeded = Math.ceil(totalDataRows / itemsPerPage);

      console.log(`Position-based: Template has ${rowsPerPage} rows, data has ${totalDataRows} items, creating ${pagesNeeded} page(s)`);

      inputs = [];

      for (let pageNum = 0; pageNum < pagesNeeded; pageNum++) {
        const pageData = {};
        const startItem = pageNum * itemsPerPage + 1;
        const endItem = (pageNum + 1) * itemsPerPage;

        // Add all non-repeating fields (headers) to each page
        Object.entries(data).forEach(([key, value]) => {
          if (!key.startsWith(itemPrefix)) {
            pageData[key] = value;
          }
        });

        // Add page numbers (Cpage = current page, Mpage = max pages)
        pageData.Cpage = (pageNum + 1).toString();
        pageData.Mpage = pagesNeeded.toString();

        // Map data items to template positions
        Object.entries(data).forEach(([key, value]) => {
          if (!key.startsWith(itemPrefix)) return;

          // Extract item number from data key
          const match = key.match(/(\d+)/);
          if (!match) return;

          const itemNum = parseInt(match[1]);

          if (itemNum >= startItem && itemNum <= endItem) {
            // Map to template position on this page
            const positionOnPage = itemNum - startItem + 1;

            // Find template field that matches this data field's pattern
            const keySuffix = key.replace(/\d+/g, '');
            const templateKey = templateRows
              .flatMap(row => row.fields)
              .find(f => f.key.replace(/\d+/g, '') === keySuffix && f.key.match(/(\d+)/) && parseInt(f.key.match(/(\d+)/)[1]) === positionOnPage);

            if (templateKey) {
              pageData[templateKey.key] = value;
            }
          }
        });

        inputs.push(pageData);
      }

      console.log(`Mapped ${totalDataRows} data rows to ${inputs.length} page(s)`);
    }
  } else {
    inputs = Array.isArray(data) ? data : [data];
  }

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
