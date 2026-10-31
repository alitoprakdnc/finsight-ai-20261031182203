/// <reference types="cypress" />

import { DocumentStatus } from '../../src/types/document';

describe('Document Upload and Processing Flow', () => {
  const fixtureFile = 'sample-invoice.pdf';
  const tenantId = 'tenant-test';
  const documentId = 'doc-12345';

  beforeEach(() => {
    // Custom command defined in support/commands.ts to perform OIDC login
    cy.login();

    // Preserve auth token between tests
    Cypress.Cookies.preserveOnce('session_id', 'auth_token');

    // Intercept the API call that creates a document record and returns a pre‑signed URL
    cy.intercept('POST', '/api/v1/documents', (req) => {
      req.reply({
        statusCode: 201,
        body: {
          id: documentId,
          uploadUrl: `http://minio.local/${tenantId}/${documentId}/upload`,
          status: DocumentStatus.Pending,
        },
      });
    }).as('createDocument');

    // Intercept the direct PUT to MinIO (the pre‑signed URL). Accept any URL pattern.
    cy.intercept('PUT', '**/upload', {
      statusCode: 200,
    }).as('uploadToStorage');

    // Intercept the GET that the UI polls for document status.
    // First call returns Pending, second call returns Processed.
    let pollCount = 0;
    cy.intercept('GET', `/api/v1/documents/${documentId}`, (req) => {
      pollCount += 1;
      if (pollCount < 2) {
        req.reply({
          statusCode: 200,
          body: {
            id: documentId,
            status: DocumentStatus.Pending,
            fileName: fixtureFile,
            extractedData: null,
          },
        });
      } else {
        req.reply({
          statusCode: 200,
          body: {
            id: documentId,
            status: DocumentStatus.Processed,
            fileName: fixtureFile,
            extractedData: {
              vendor: 'Acme Corp',
              total: 1234.56,
              currency: 'USD',
              lineItems: [
                { description: 'Consulting Services', amount: 1000.0 },
                { description: 'Software License', amount: 234.56 },
              ],
              tax: 0.0,
            },
          },
        });
      }
    }).as('pollDocument');

    // Intercept webhook dispatch (optional, just to avoid external calls)
    cy.intercept('POST', '/api/v1/webhooks/**', {
      statusCode: 200,
    }).as('webhookDispatch');
  });

  it('should upload a document, show processing status and display extracted data', () => {
    // Navigate to the documents page
    cy.visit('/documents');

    // Ensure the page loaded and the upload dropzone is visible
    cy.get('[data-cy=upload-dropzone]').should('be.visible');

    // Use the custom command to upload a file via the dropzone component
    // The command handles the drag‑drop interaction and waits for the upload to complete.
    cy.uploadDocument(fixtureFile);

    // Wait for the backend to receive the create‑document request
    cy.wait('@createDocument')
      .its('response.statusCode')
      .should('eq', 201);

    // Wait for the file to be PUT to MinIO
    cy.wait('@uploadToStorage')
      .its('response.statusCode')
      .should('eq', 200);

    // After successful upload the UI should show a new document card with a "Pending" badge
    cy.get('[data-cy=document-list]')
      .contains(fixtureFile)
      .parents('[data-cy=document-card]')
      .as('newDocCard');

    cy.get('@newDocCard')
      .find('[data-cy=status-badge]')
      .should('contain.text', DocumentStatus.Pending);

    // The UI polls for status; wait for the second poll which returns "Processed"
    cy.wait('@pollDocument'); // first poll (Pending)
    cy.wait('@pollDocument'); // second poll (Processed)

    // Verify the status badge updates to "Processed"
    cy.get('@newDocCard')
      .find('[data-cy=status-badge]')
      .should('contain.text', DocumentStatus.Processed);

    // Verify that extracted data appears in the detail view
    cy.get('@newDocCard')
      .find('[data-cy=view-details]')
      .click();

    // The detail page should show vendor, total and line items
    cy.get('[data-cy=extracted-vendor]').should('contain.text', 'Acme Corp');
    cy.get('[data-cy=extracted-total]').should('contain.text', '$1,234.56');
    cy.get('[data-cy=line-items]')
      .should('contain.text', 'Consulting Services')
      .and('contain.text', '$1,000.00')
      .and('contain.text', 'Software License')
      .and('contain.text', '$234.56');

    // A success toast should be displayed after processing completes
    cy.get('[data-cy=toast]')
      .should('be.visible')
      .and('contain.text', 'Document processed successfully');

    // Ensure no unexpected webhook errors were logged
    cy.wait('@webhookDispatch').its('response.statusCode').should('eq', 200);
  });
});