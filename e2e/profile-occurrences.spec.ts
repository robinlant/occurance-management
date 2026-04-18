import { test, expect, Page } from '@playwright/test';

// Helper: log in as admin and set English language cookie
async function login(page: Page) {
  // Set English language cookie before any navigation
  await page.context().addCookies([
    { name: 'dr-lang', value: 'en', domain: 'localhost', path: '/' },
  ]);
  await page.request.post('/login', {
    form: { email: 'admin@test.com', password: 'password123' },
  });
}

// Helper: extract CSRF token from a page
async function getCSRF(page: Page, url: string): Promise<string> {
  await page.goto(url);
  const csrfInput = page.locator('input[name="_csrf"]').first();
  const token = await csrfInput.getAttribute('value');
  if (!token) throw new Error('CSRF token not found on ' + url);
  return token;
}

// Helper: create a duty via POST and return the duty ID
async function createDuty(
  page: Page,
  opts: { title: string; date: string; min?: number; max?: number },
): Promise<number> {
  // GET the new duty form to grab the CSRF token
  const csrf = await getCSRF(page, '/duties/new');

  const res = await page.request.post('/duties', {
    form: {
      title: opts.title,
      date: opts.date,
      min_participants: String(opts.min ?? 1),
      max_participants: String(opts.max ?? 5),
      group_id: '',
      allow_over_limit: '',
      _csrf: csrf,
    },
    maxRedirects: 0,
  });
  // The handler redirects to /duties on success
  expect([200, 302]).toContain(res.status());

  // Include past duties because some tests intentionally create past entries.
  await page.goto('/duties?hide_past=0');
  const link = page.locator(`a:has(.occ-title:has-text("${opts.title}"))`).first();
  const href = await link.getAttribute('href');
  if (!href) throw new Error('Could not find duty link for ' + opts.title);
  const id = parseInt(href.split('/').pop()!, 10);
  return id;
}

// Helper: sign up for a duty via POST
async function signUpForDuty(page: Page, occId: number): Promise<void> {
  // Visit the detail page to get a CSRF token
  const csrf = await getCSRF(page, `/duties/${occId}`);

  const res = await page.request.post(`/duties/${occId}/signup`, {
    form: { _csrf: csrf },
  });
  expect([200, 302]).toContain(res.status());
}

test.describe('Profile duties', () => {
  test('shows "Your duties" section on profile page', async ({ page }) => {
    await login(page);
    await page.goto('/profile');
    // The card-title containing "Your duties" should be visible
    const heading = page.locator('.card-title', { hasText: 'Your duties' });
    await expect(heading).toBeVisible();
  });

  test('shows empty message when user has no duties', async ({ page }) => {
    await login(page);
    await page.goto('/profile');
    // When the user hasn't signed up for anything, the empty message is shown
    const emptyMsg = page.locator('text=No duties yet.');
    await expect(emptyMsg).toBeVisible();
  });

  test('shows duties after signing up', async ({ page }) => {
    await login(page);

    // Create a future duty
    const futureDate = '2027-06-15T10:00';
    const occId = await createDuty(page, {
      title: 'Future Duty A',
      date: futureDate,
    });

    // Sign up for it
    await signUpForDuty(page, occId);

    // Go to profile
    await page.goto('/profile');

    // The empty message should be gone
    await expect(page.locator('text=No duties yet.')).not.toBeVisible();

    // The duty should appear
    const occItem = page.locator('.occ-item', { hasText: 'Future Duty A' });
    await expect(occItem).toBeVisible();
  });

  test('upcoming duties appear before past ones', async ({ page }) => {
    await login(page);

    // Create a past duty
    const pastDate = '2025-01-10T09:00';
    const pastId = await createDuty(page, {
      title: 'Past Duty B',
      date: pastDate,
    });
    await signUpForDuty(page, pastId);

    // Create a future duty
    const futureDate = '2027-07-20T14:00';
    const futureId = await createDuty(page, {
      title: 'Future Duty B',
      date: futureDate,
    });
    await signUpForDuty(page, futureId);

    // Visit profile
    await page.goto('/profile');

    // Collect the duty titles in order
    const titles = await page.locator('.occ-item .occ-title').allTextContents();
    const cleaned = titles.map((t) => t.replace(/\s+/g, ' ').trim());

    const futureIdx = cleaned.findIndex((t) => t.includes('Future Duty B'));
    const pastIdx = cleaned.findIndex((t) => t.includes('Past Duty B'));

    expect(futureIdx).toBeGreaterThanOrEqual(0);
    expect(pastIdx).toBeGreaterThanOrEqual(0);
    // Future (upcoming) should come before past
    expect(futureIdx).toBeLessThan(pastIdx);
  });

  test('each duty shows title, date, and participant count', async ({
    page,
  }) => {
    await login(page);

    // Create a duty with specific participant limits
    const occDate = '2027-08-01T08:00';
    const occId = await createDuty(page, {
      title: 'Detail Check Duty',
      date: occDate,
      min: 2,
      max: 4,
    });
    await signUpForDuty(page, occId);

    await page.goto('/profile');

    const occItem = page.locator('.occ-item', { hasText: 'Detail Check Duty' });
    await expect(occItem).toBeVisible();

    // Check that the title is present
    await expect(occItem.locator('.occ-title')).toContainText('Detail Check Duty');

    // Check that the date section is present (occ-date)
    const dateText = await occItem.locator('.occ-date').textContent();
    expect(dateText).toBeTruthy();

    // Check that the participant count is shown (format: count/min-max people)
    const slotsText = await occItem.locator('.occ-slots').allTextContents();
    const joined = slotsText.join(' ');
    // Should contain "1/2–4 people" (1 participant signed up, min 2, max 4)
    expect(joined).toMatch(/1\/2.*4\s*people/);
  });

  test('duties link to the detail page', async ({ page }) => {
    await login(page);

    // Create a duty and sign up
    const occId = await createDuty(page, {
      title: 'Linked Duty',
      date: '2027-09-01T10:00',
    });
    await signUpForDuty(page, occId);

    await page.goto('/profile');

    // Find the link wrapping the duty
    const link = page.locator(`a[href="/duties/${occId}"]`);
    await expect(link).toBeVisible();

    // Click and verify navigation
    await link.click();
    await expect(page).toHaveURL(new RegExp(`/duties/${occId}`));
    // Should be on the detail page showing the title
    await expect(page.locator('h2')).toContainText('Linked Duty');
  });

  test('public profile also shows duties section', async ({ browser }) => {
    // Create a second user via admin, sign them up for a duty,
    // then view their public profile from the admin account.
    const ctx = await browser.newContext({ baseURL: 'http://localhost:3991' });
    const page = await ctx.newPage();
    await login(page);

    // Create a participant user via admin panel
    const csrfUsers = await getCSRF(page, '/users');
    await page.request.post('/users', {
      form: {
        name: 'PublicUser',
        email: 'publicuser@test.com',
        password: 'password123',
        role: 'participant',
        _csrf: csrfUsers,
      },
    });

    // Create a duty and assign the new user to it
    const occId = await createDuty(page, {
      title: 'Public Profile Duty',
      date: '2027-10-01T11:00',
    });

    // Assign user 2 (PublicUser) to this duty
    const csrfAssign = await getCSRF(page, `/duties/${occId}`);
    await page.request.post(`/duties/${occId}/assign`, {
      form: { user_id: '2', _csrf: csrfAssign },
    });

    // Visit the public profile of user 2
    await page.goto('/profile/2');

    // Should show the duties section (public profile says "Duties", not "Your duties")
    const heading = page.locator('.card-title', { hasText: 'Duties' });
    await expect(heading).toBeVisible();

    // Should show the assigned duty
    await expect(page.locator('.occ-item', { hasText: 'Public Profile Duty' })).toBeVisible();

    await ctx.close();
  });

  test('participant names on duty detail link to their profile', async ({ browser }) => {
    const ctx = await browser.newContext({ baseURL: 'http://localhost:3991' });
    const page = await ctx.newPage();
    await login(page);

    // Create a participant user
    const csrfUsers = await getCSRF(page, '/users');
    await page.request.post('/users', {
      form: {
        name: 'ClickableUser',
        email: 'clickable@test.com',
        password: 'password123',
        role: 'participant',
        _csrf: csrfUsers,
      },
    });

    // Create a duty and assign both users
    const occId = await createDuty(page, {
      title: 'Clickable Test',
      date: '2027-11-20T10:00',
    });
    await signUpForDuty(page, occId);

    // Assign the new user too
    const csrfAssign = await getCSRF(page, `/duties/${occId}`);
    await page.request.post(`/duties/${occId}/assign`, {
      form: { user_id: '3', _csrf: csrfAssign },
    });

    // Go to duty detail
    await page.goto(`/duties/${occId}`);

    // The participant name should be a link to their profile
    const participantLink = page.locator('#participant-section a[href="/profile/3"]');
    await expect(participantLink).toBeVisible();
    await expect(participantLink).toContainText('ClickableUser');

    // Click it and verify navigation to profile
    await participantLink.click();
    await expect(page).toHaveURL(/\/profile\/3/);

    await ctx.close();
  });
});
