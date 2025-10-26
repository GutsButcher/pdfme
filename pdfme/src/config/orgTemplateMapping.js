/**
 * Organization ID to Template Mapping
 * Maps orgId from parser output to the appropriate template
 */

const ORG_TEMPLATE_MAP = {
  '266': 'new-template',
  // Add more orgId mappings here as needed
  // '123': 'invoice_template',
  // '456': 'receipt_template',
};

/**
 * Get template name for a given organization ID
 * @param {string} orgId - Organization ID from parser
 * @returns {string} Template name
 * @throws {Error} If orgId is not mapped
 */
function getTemplateForOrg(orgId) {
  const templateName = ORG_TEMPLATE_MAP[orgId];

  if (!templateName) {
    throw new Error(`No template mapping found for orgId: ${orgId}`);
  }

  return templateName;
}

/**
 * Check if orgId has a template mapping
 * @param {string} orgId - Organization ID
 * @returns {boolean}
 */
function hasTemplateMapping(orgId) {
  return orgId in ORG_TEMPLATE_MAP;
}

module.exports = {
  ORG_TEMPLATE_MAP,
  getTemplateForOrg,
  hasTemplateMapping,
};
