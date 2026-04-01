import { test, expect, type Page, type BrowserContext } from '@playwright/test';

test.use({ baseURL: 'http://localhost:3993' });

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Log in as the given user, return cookies for API calls. */
async function login(page: Page, email: string, password: string) {
  await page.goto('/login');
  await page.fill('input[name="email"]', email);
  await page.fill('input[name="password"]', password);
  await page.click('button[type="submit"]');
  await page.waitForURL(/\/(dashboard|occurrences|profile)?$/);
}

/** Set English locale cookie */
async function setEnglish(context: BrowserContext) {
  await context.addCookies([
    { name: 'dr-lang', value: 'en', domain: 'localhost', path: '/' },
  ]);
}

/** Extract CSRF token from a page (hidden input) */
async function getCSRF(page: Page, url: string): Promise<string> {
  await page.goto(url);
  const csrf = await page.inputValue('input[name="_csrf"]');
  return csrf;
}

/** Create an occurrence via API. Returns the occurrence ID. */
async function createOccurrence(
  page: Page,
  opts: {
    title: string;
    date: string; // YYYY-MM-DDTHH:mm
    minParticipants?: number;
    maxParticipants?: number;
    groupId?: number;
  },
): Promise<number> {
  // Get CSRF from the new occurrence form
  const csrf = await getCSRF(page, '/occurrences/new');

  const formData: Record<string, string> = {
    title: opts.title,
    date: opts.date,
    min_participants: String(opts.minParticipants ?? 1),
    max_participants: String(opts.maxParticipants ?? 5),
    _csrf: csrf,
  };
  if (opts.groupId !== undefined) {
    formData['group_id'] = String(opts.groupId);
  }

  // Submit the form via a POST
  const response = await page.request.post('/occurrences', {
    form: formData,
  });
  // The server redirects to /occurrences on success
  expect(response.status()).toBeLessThan(400);

  // Navigate to occurrences list and find the occurrence we just created
  await page.goto('/occurrences');
  const link = page.locator(`a:has-text("${opts.title}")`).first();
  const href = await link.getAttribute('href');
  const id = parseInt(href!.split('/').pop()!, 10);
  return id;
}

/** Sign up for an occurrence via API */
async function signUpForOccurrence(page: Page, occId: number) {
  const csrf = await getCSRF(page, `/occurrences/${occId}`);
  const response = await page.request.post(`/occurrences/${occId}/signup`, {
    form: { _csrf: csrf },
  });
  // HTMX endpoint: 200 expected
  expect(response.status()).toBeLessThan(400);
}

/** Withdraw from an occurrence via API */
async function withdrawFromOccurrence(page: Page, occId: number) {
  const csrf = await getCSRF(page, `/occurrences/${occId}`);
  const response = await page.request.post(`/occurrences/${occId}/withdraw`, {
    form: { _csrf: csrf },
  });
  expect(response.status()).toBeLessThan(400);
}

/** Delete an occurrence via API (admin) */
async function deleteOccurrence(page: Page, occId: number) {
  const csrf = await getCSRF(page, `/occurrences/${occId}`);
  const response = await page.request.post(`/occurrences/${occId}/delete`, {
    form: { _csrf: csrf },
  });
  expect(response.status()).toBeLessThan(400);
}

/** Create a user via admin API. Returns nothing (we look up the ID later). */
async function createUser(
  page: Page,
  opts: { name: string; email: string; password: string; role: string },
) {
  const csrf = await getCSRF(page, '/users');
  const response = await page.request.post('/users', {
    form: {
      name: opts.name,
      email: opts.email,
      password: opts.password,
      role: opts.role,
      _csrf: csrf,
    },
  });
  expect(response.status()).toBeLessThan(400);
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe('Profile occurrences — edge cases and bugs', () => {
  test.beforeEach(async ({ context }) => {
    await setEnglish(context);
  });

  // 1. Empty state
  test('empty state shows "No occurrences" message', async ({ page }) => {
    await login(page, 'bugadmin@test.com', 'password123');
    await page.goto('/profile');

    // The profile page should contain "No occurrences yet."
    const noOccText = page.locator('text=No occurrences yet.');
    await expect(noOccText).toBeVisible();
  });

  // 2. Occurrence with no group
  test('occurrence without group renders correctly on profile (no badge, no crash)', async ({
    page,
  }) => {
    await login(page, 'bugadmin@test.com', 'password123');

    // Create occurrence without group_id
    const occId = await createOccurrence(page, {
      title: 'NoGroupOcc',
      date: '2027-06-15T10:00',
    });

    await signUpForOccurrence(page, occId);
    await page.goto('/profile');

    // The occurrence should appear on the profile
    await expect(page.locator('text=NoGroupOcc')).toBeVisible();

    // There should be no crash — page loaded successfully
    // Verify no group badge next to this occurrence
    const occItem = page.locator('.occ-item:has-text("NoGroupOcc")');
    await expect(occItem).toBeVisible();

    // The badge inside this item should not exist (no group badge)
    // Group badges have classes like badge-accent or badge-group-*
    const groupBadge = occItem.locator('.badge:not(.badge-gray)');
    await expect(groupBadge).toHaveCount(0);

    // Cleanup
    await deleteOccurrence(page, occId);
  });

  // 3. Withdrawn occurrence should not appear on profile
  test('withdrawn occurrence does not appear on profile', async ({ page }) => {
    await login(page, 'bugadmin@test.com', 'password123');

    const occId = await createOccurrence(page, {
      title: 'WithdrawTestOcc',
      date: '2027-07-20T14:00',
    });

    await signUpForOccurrence(page, occId);

    // Verify it appears first
    await page.goto('/profile');
    await expect(page.locator('text=WithdrawTestOcc')).toBeVisible();

    // Withdraw
    await withdrawFromOccurrence(page, occId);

    // Verify it is gone from profile
    await page.goto('/profile');
    await expect(page.locator('text=WithdrawTestOcc')).not.toBeVisible();

    // Cleanup
    await deleteOccurrence(page, occId);
  });

  // 4. Multiple occurrences ordering: future (earliest first), then past (newest first)
  test('occurrences are ordered: future earliest-first, then past newest-first', async ({
    page,
  }) => {
    await login(page, 'bugadmin@test.com', 'password123');

    // Create past occurrences (dates in the past)
    const past1 = await createOccurrence(page, {
      title: 'Past1-Jan',
      date: '2025-01-10T09:00',
    });
    const past2 = await createOccurrence(page, {
      title: 'Past2-Mar',
      date: '2025-03-15T09:00',
    });
    const past3 = await createOccurrence(page, {
      title: 'Past3-May',
      date: '2025-05-20T09:00',
    });

    // Create future occurrences
    const future1 = await createOccurrence(page, {
      title: 'Future1-Dec',
      date: '2027-12-01T09:00',
    });
    const future2 = await createOccurrence(page, {
      title: 'Future2-Aug',
      date: '2027-08-15T09:00',
    });

    // Sign up for all
    for (const id of [past1, past2, past3, future1, future2]) {
      await signUpForOccurrence(page, id);
    }

    await page.goto('/profile');

    // Collect all occurrence titles from the profile in order
    const titles = await page.locator('.occ-title').allTextContents();
    const trimmedTitles = titles.map((t) => t.trim().split('\n')[0].trim());

    // Expected order:
    // Future earliest first: Future2-Aug (2027-08), Future1-Dec (2027-12)
    // Past newest first: Past3-May (2025-05), Past2-Mar (2025-03), Past1-Jan (2025-01)
    const future2Idx = trimmedTitles.findIndex((t) => t.includes('Future2-Aug'));
    const future1Idx = trimmedTitles.findIndex((t) => t.includes('Future1-Dec'));
    const past3Idx = trimmedTitles.findIndex((t) => t.includes('Past3-May'));
    const past2Idx = trimmedTitles.findIndex((t) => t.includes('Past2-Mar'));
    const past1Idx = trimmedTitles.findIndex((t) => t.includes('Past1-Jan'));

    expect(future2Idx).toBeGreaterThanOrEqual(0);
    expect(future1Idx).toBeGreaterThanOrEqual(0);
    expect(past3Idx).toBeGreaterThanOrEqual(0);
    expect(past2Idx).toBeGreaterThanOrEqual(0);
    expect(past1Idx).toBeGreaterThanOrEqual(0);

    // Future before past
    expect(future2Idx).toBeLessThan(past3Idx);
    expect(future1Idx).toBeLessThan(past3Idx);

    // Future: earliest first
    expect(future2Idx).toBeLessThan(future1Idx);

    // Past: newest first
    expect(past3Idx).toBeLessThan(past2Idx);
    expect(past2Idx).toBeLessThan(past1Idx);

    // Cleanup
    for (const id of [past1, past2, past3, future1, future2]) {
      await deleteOccurrence(page, id);
    }
  });

  // 5. Same-day occurrences — multiple on the same date should all appear
  test('multiple occurrences on the same date all appear on profile', async ({
    page,
  }) => {
    await login(page, 'bugadmin@test.com', 'password123');

    const sameDayOcc1 = await createOccurrence(page, {
      title: 'SameDay-Morning',
      date: '2027-09-10T08:00',
    });
    const sameDayOcc2 = await createOccurrence(page, {
      title: 'SameDay-Afternoon',
      date: '2027-09-10T14:00',
    });
    const sameDayOcc3 = await createOccurrence(page, {
      title: 'SameDay-Evening',
      date: '2027-09-10T19:00',
    });

    for (const id of [sameDayOcc1, sameDayOcc2, sameDayOcc3]) {
      await signUpForOccurrence(page, id);
    }

    await page.goto('/profile');

    await expect(page.locator('text=SameDay-Morning')).toBeVisible();
    await expect(page.locator('text=SameDay-Afternoon')).toBeVisible();
    await expect(page.locator('text=SameDay-Evening')).toBeVisible();

    // Cleanup
    for (const id of [sameDayOcc1, sameDayOcc2, sameDayOcc3]) {
      await deleteOccurrence(page, id);
    }
  });

  // 6. Profile redirect — /profile/:own_id redirects to /profile
  test('visiting /profile/:own_id redirects to /profile', async ({ page }) => {
    await login(page, 'bugadmin@test.com', 'password123');

    // Admin user was seeded with id=1
    const response = await page.goto('/profile/1');

    // Should redirect to /profile (without the ID)
    expect(page.url()).toMatch(/\/profile\/?$/);
    // It should NOT contain /profile/1
    expect(page.url()).not.toContain('/profile/1');
  });

  // 7. Occurrence deleted after signup — profile should not crash
  test('profile does not crash when a signed-up occurrence is deleted', async ({
    page,
  }) => {
    await login(page, 'bugadmin@test.com', 'password123');

    const occId = await createOccurrence(page, {
      title: 'DeleteAfterSignup',
      date: '2027-10-05T10:00',
    });

    await signUpForOccurrence(page, occId);

    // Verify it appears
    await page.goto('/profile');
    await expect(page.locator('text=DeleteAfterSignup')).toBeVisible();

    // Delete the occurrence as admin
    await deleteOccurrence(page, occId);

    // Profile should load without errors
    const response = await page.goto('/profile');
    expect(response!.status()).toBe(200);

    // The deleted occurrence should not appear
    await expect(page.locator('text=DeleteAfterSignup')).not.toBeVisible();
  });

  // 8. Large number of occurrences — profile loads without issues
  test('profile handles 20+ occurrences without issues', async ({ page }) => {
    await login(page, 'bugadmin@test.com', 'password123');

    const ids: number[] = [];
    for (let i = 0; i < 22; i++) {
      const month = String((i % 12) + 1).padStart(2, '0');
      const day = String((i % 28) + 1).padStart(2, '0');
      const year = i < 12 ? '2027' : '2028';
      const id = await createOccurrence(page, {
        title: `BulkOcc-${i}`,
        date: `${year}-${month}-${day}T10:00`,
      });
      ids.push(id);
      await signUpForOccurrence(page, id);
    }

    const response = await page.goto('/profile');
    expect(response!.status()).toBe(200);

    // Check that occurrences are rendered
    const occItems = page.locator('.occ-item');
    const count = await occItems.count();
    expect(count).toBeGreaterThanOrEqual(22);

    // Cleanup
    for (const id of ids) {
      await deleteOccurrence(page, id);
    }
  });

  // 9. Public profile shows other user's occurrences, not current user's
  test("public profile shows other user's occurrences, not the current user's", async ({
    page,
  }) => {
    await login(page, 'bugadmin@test.com', 'password123');

    // Create user B with correct role
    await createUser(page, {
      name: 'UserB',
      email: 'userb@test.com',
      password: 'password123',
      role: 'participant',
    });

    // Create two occurrences: one for admin, one for user B
    const adminOcc = await createOccurrence(page, {
      title: 'AdminOnlyOcc',
      date: '2027-11-01T09:00',
    });
    const sharedOcc = await createOccurrence(page, {
      title: 'UserBOcc',
      date: '2027-11-15T09:00',
      maxParticipants: 10,
    });

    // Admin signs up for their occurrence
    await signUpForOccurrence(page, adminOcc);

    // The admin user is ID 1, so UserB should be ID 2.
    // Assign user B to the occurrence via admin assign endpoint
    const csrf = await getCSRF(page, `/occurrences/${sharedOcc}`);
    await page.request.post(`/occurrences/${sharedOcc}/assign`, {
      form: { user_id: '2', _csrf: csrf },
    });

    // Now visit user B's public profile
    await page.goto('/profile/2');

    // Should see UserBOcc
    await expect(page.locator('text=UserBOcc')).toBeVisible();

    // Should NOT see AdminOnlyOcc (that's only the admin's)
    await expect(page.locator('text=AdminOnlyOcc')).not.toBeVisible();

    // Verify it shows user B's name in the page title
    await expect(page.locator('.topbar-title', { hasText: 'UserB' })).toBeVisible();

    // Cleanup
    await deleteOccurrence(page, adminOcc);
    await deleteOccurrence(page, sharedOcc);
  });
});
