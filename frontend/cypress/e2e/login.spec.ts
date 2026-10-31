/// <reference types="cypress" />

/**
 * Cypress E2E test for the login flow.
 *
 * The test verifies that:
 *   1. The user can initiate the OIDC login from the landing page.
 *   2. The application exchanges the OIDC `code` for a JWT.
 *   3. The JWT is persisted in `localStorage` under the key `auth_token`.
 *   4. The UI reflects the authenticated state (user menu displays the user name).
 *
 * The OIDC provider is stubbed using `cy.intercept` so the test runs
 * deterministically without external network calls.
 */

describe('Login Flow', () => {
  const ACCESS_TOKEN = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.testpayload.signature';
  const REFRESH_TOKEN = 'refresh.test.token';
  const USER = {
    id: 'user-123',
    name: 'Test User',
    email: 'test.user@example.com',
    role: 'Accountant',
  };

  beforeEach(() => {
    // Ensure a clean state before each test.
    cy.clearLocalStorage();

    // Stub the OIDC authorization endpoint – the app redirects here when the
    // "Sign in" button is clicked. We respond with a 302 to the callback URL
    // containing a dummy `code` query param.
    cy.intercept('GET', '/api/auth/login', (req) => {
      req.reply({
        statusCode: 302,
        headers: {
          location: '/api/auth/callback?code=test-auth-code',
        },
      });
    }).as('oidcLogin');

    // Stub the token exchange endpoint that the backend calls after the
    // callback. The request body contains the `code` from the query string.
    cy.intercept('POST', '/api/auth/token', (req) => {
      // Validate that the request contains the expected code – helps catch
      // regressions if the client changes the payload shape.
      expect(req.body).to.have.property('code', 'test-auth-code');
      req.reply({
        statusCode: 200,
        body: {
          access_token: ACCESS_TOKEN,
          refresh_token: REFRESH_TOKEN,
          expires_in: 3600,
        },
      });
    }).as('tokenExchange');

    // After a successful token exchange the frontend fetches the current
    // user profile to populate the UI. Stub that call.
    cy.intercept('GET', '/api/v1/user/me', {
      statusCode: 200,
      body: USER,
    }).as('fetchUser');
  });

  it('should log in via OIDC and store the JWT', () => {
    // 1️⃣ Visit the landing page.
    cy.visit('/');

    // 2️⃣ Click the "Sign in" button – the app should redirect to the OIDC login
    //    endpoint, which we have stubbed to immediately redirect back to the
    //    callback URL.
    cy.contains('button', /sign in/i).click();
    cy.wait('@oidcLogin');

    // 3️⃣ The app follows the redirect and lands on the callback route.
    //    Verify that the URL contains the `code` query param.
    cy.url().should('include', '/api/auth/callback?code=test-auth-code');

    // 4️⃣ The callback triggers the token exchange request.
    cy.wait('@tokenExchange');

    // 5️⃣ Verify that the JWT is persisted in localStorage under the expected key.
    cy.window().then((win) => {
      const storedToken = win.localStorage.getItem('auth_token');
      expect(storedToken).to.equal(ACCESS_TOKEN);
    });

    // 6️⃣ The UI should now display the authenticated state – a user menu with
    //    the logged‑in user's name.
    cy.wait('@fetchUser');
    cy.get('[data-testid="user-menu"]')
      .should('be.visible')
      .and('contain', USER.name);
  });
});