/**
 * Parser Data Transformer
 * Converts parser output format to template-compatible format
 */

const { getTemplateForOrg } = require('../config/orgTemplateMapping');

/**
 * Transform parser output to template-compatible data
 * Converts transactions array to Tr1, Tr2, Tr3... format
 *
 * @param {Object} parserOutput - Output from parser service
 * @param {string} parserOutput.orgId - Organization ID
 * @param {string} parserOutput.name - Customer name
 * @param {string} parserOutput.address - Customer address
 * @param {string} parserOutput.cardNumber - Card number
 * @param {string} parserOutput.statementDate - Statement date
 * @param {Array} parserOutput.transactions - Array of transactions
 * @returns {Object} - Transformed data ready for PDF generation
 */
function transformParserData(parserOutput) {
  const {
    orgId,
    name,
    address,
    cardNumber,
    statementDate,
    availableBalance,
    openingBalance,
    currentBalance,
    toatalDepits,
    totalCredits,
    transactions = [],
  } = parserOutput;

  // Get template name based on orgId
  const templateName = getTemplateForOrg(orgId);

  // Initialize template-compatible data with header fields
  const templateData = {
    Cname: name || '',
    Caddress: address || '',
    CardNumber: cardNumber || '',
    StatmentDate: statementDate || '',
  };

  // Split card number into individual digits (CN1, CN2, ... CN16)
  if (cardNumber) {
    const digits = cardNumber.toString().split('');
    digits.forEach((digit, index) => {
      templateData[`CN${index + 1}`] = digit;
    });
  }

  // Add balance fields if needed (templates might use these)
  if (availableBalance !== undefined) templateData.AvailableBalance = availableBalance.toString();
  if (openingBalance !== undefined) templateData.OpeningBalance = openingBalance.toString();
  if (currentBalance !== undefined) templateData.CurrentBalance = currentBalance.toString();
  if (toatalDepits !== undefined) templateData.TotalDepits = toatalDepits.toString();
  if (totalCredits !== undefined) templateData.TotalCredits = totalCredits.toString();

  // Transform transactions array to Tr1, Tr2, Tr3... format
  // Filter out transactions that have actual meaningful data
  const validTransactions = transactions.filter(tx => {
    // Skip header rows or rows with no actual transaction data
    return tx.date && tx.description && !tx.description.includes('Account transactions');
  });

  validTransactions.forEach((transaction, index) => {
    const txNumber = index + 1; // Start from 1 (Tr1, Tr2, Tr3...)

    // Map transaction fields to template format
    templateData[`Tr${txNumber}Date`] = transaction.date || '';
    templateData[`Tr${txNumber}Pdate`] = transaction.postDate || '';
    templateData[`Tr${txNumber}Details`] = transaction.description || '';

    // Determine if it's debit or credit (using plural forms: Debits/Credits)
    if (transaction.cr === false) {
      // Debit transaction
      templateData[`Tr${txNumber}Debits`] = transaction.amountInBHD ? transaction.amountInBHD.toString() : '';
      templateData[`Tr${txNumber}Credits`] = '';
    } else if (transaction.cr === true) {
      // Credit transaction
      templateData[`Tr${txNumber}Debits`] = '';
      templateData[`Tr${txNumber}Credits`] = transaction.amountInBHD ? transaction.amountInBHD.toString() : '';
    } else {
      // Unknown type, put in debit
      templateData[`Tr${txNumber}Debits`] = transaction.amountInBHD ? transaction.amountInBHD.toString() : '';
      templateData[`Tr${txNumber}Credits`] = '';
    }

    // Debug: Log first transaction mapping
    if (txNumber === 1) {
      console.log(`  Sample mapping: Tr1Debits=${templateData.Tr1Debits}, Tr1Credits=${templateData.Tr1Credits}, amount=${transaction.amountInBHD}, cr=${transaction.cr}`);
    }

    // Add currency and amount fields if needed
    if (transaction.currency) {
      templateData[`Tr${txNumber}Currency`] = transaction.currency;
    }
    if (transaction.amount) {
      templateData[`Tr${txNumber}Amount`] = transaction.amount.toString();
    }
  });

  return {
    template_name: templateName,
    data: templateData,
    pagination: {
      itemPrefix: 'Tr',
      itemsPerPage: 15, // Default, can be configured based on template
    },
  };
}

/**
 * Get transaction count from parser output
 * @param {Object} parserOutput - Parser output
 * @returns {number} Number of valid transactions
 */
function getTransactionCount(parserOutput) {
  const { transactions = [] } = parserOutput;

  return transactions.filter(tx => {
    return tx.date && tx.description && !tx.description.includes('Account transactions');
  }).length;
}

module.exports = {
  transformParserData,
  getTransactionCount,
};
