const fs = require('fs').promises;
const { PDFDocument } = require('pdf-lib');

/**
 * Duplicate the first page of a base PDF to create a multi-page PDF
 * @param {string} templatePath - Path to the template JSON file
 * @param {number} numPages - Number of total pages needed
 */
async function duplicatePdfPages(templatePath, numPages = 2) {
  try {
    const templateData = await fs.readFile(templatePath, 'utf-8');
    const template = JSON.parse(templateData);

    // Decode base64 PDF
    const base64Pdf = template.basePdf.replace('data:application/pdf;base64,', '');
    const pdfBytes = Buffer.from(base64Pdf, 'base64');

    // Load PDF
    const pdfDoc = await PDFDocument.load(pdfBytes);

    const currentPages = pdfDoc.getPageCount();
    console.log(`Current PDF pages: ${currentPages}`);

    // Add pages if needed
    if (currentPages < numPages) {
      const pagesToAdd = numPages - currentPages;
      console.log(`Adding ${pagesToAdd} page(s)...`);

      for (let i = 0; i < pagesToAdd; i++) {
        // Copy first page
        const [copiedPage] = await pdfDoc.copyPages(pdfDoc, [0]);
        pdfDoc.addPage(copiedPage);
      }

      console.log(`New PDF pages: ${pdfDoc.getPageCount()}`);

      // Save PDF
      const newPdfBytes = await pdfDoc.save();
      const newBase64 = 'data:application/pdf;base64,' + Buffer.from(newPdfBytes).toString('base64');

      // Update template
      template.basePdf = newBase64;
      await fs.writeFile(templatePath, JSON.stringify(template, null, 2));

      console.log('BasePDF updated successfully');
      return true;
    } else {
      console.log('PDF already has enough pages');
      return false;
    }
  } catch (error) {
    console.error('Error duplicating PDF pages:', error.message);
    throw error;
  }
}

module.exports = { duplicatePdfPages };

// If run directly from command line
if (require.main === module) {
  const templatePath = process.argv[2] || '/app/templates/template_form.json';
  const numPages = parseInt(process.argv[3]) || 2;

  duplicatePdfPages(templatePath, numPages)
    .then(() => console.log('Done'))
    .catch(console.error);
}
