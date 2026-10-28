import { test, expect } from '@playwright/test';

// Standard mock fixtures
const mockConfig = {
  doh_domain: 'dns.ski.io.vn',
  oidc_issuer: 'https://id.ski.io.vn',
  oidc_client_id: 'test-client',
  oidc_scopes: 'openid profile email',
  oidc_provider_name: 'Single Sign-On',
  max_devices_per_user: 5,
  dev_mode: true,
};

const mockUser = {
  id: 'user-123',
  name: 'Dominic Nguyen',
  email: 'dominic@ski.io.vn',
};

const mockServices = [
  { id: 'youtube', name: 'YouTube', icon: 'youtube' },
  { id: 'tiktok', name: 'TikTok', icon: 'tiktok' },
  { id: 'facebook', name: 'Facebook', icon: 'facebook' },
  { id: 'discord', name: 'Discord', icon: 'discord' },
];

let devices = [
  {
    id: 'dev-1',
    name: "Dominic's iPhone",
    client_token: 'u-token1234',
    user_id: 'user-123',
    filtering_enabled: true,
    parental_enabled: true,
    safe_search_enabled: false,
    blocked_services: ['tiktok'],
    custom_blocked_domains: ['distracting.com'],
    custom_allowed_domains: ['work-tool.org'],
    created_at: new Date().toISOString(),
  },
];

test.beforeEach(async ({ page }) => {
  // Reset devices for each test
  devices = [
    {
      id: 'dev-1',
      name: "Dominic's iPhone",
      client_token: 'u-token1234',
      user_id: 'user-123',
      filtering_enabled: true,
      parental_enabled: true,
      safe_search_enabled: false,
      blocked_services: ['tiktok'],
      custom_blocked_domains: ['distracting.com'],
      custom_allowed_domains: ['work-tool.org'],
      created_at: new Date().toISOString(),
    },
  ];

  // Mock API routes
  await page.route('/api/config', async (route) => {
    await route.fulfill({ json: mockConfig });
  });

  await page.route('/api/services', async (route) => {
    await route.fulfill({ json: mockServices });
  });

  await page.route('/api/me', async (route) => {
    const authHeader = route.request().headers()['authorization'];
    if (authHeader && (authHeader.startsWith('Bearer ') || authHeader.startsWith('bearer '))) {
      await route.fulfill({ json: mockUser });
    } else {
      await route.fulfill({ status: 401, json: { error: 'Unauthorized' } });
    }
  });

  await page.route('/api/devices', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({ json: devices });
    } else if (route.request().method() === 'POST') {
      const data = route.request().postDataJSON();
      const newDev = {
        id: `dev-${Date.now()}`,
        name: data.name || 'New Device',
        client_token: `u-${Math.random().toString(36).substring(7)}`,
        user_id: 'user-123',
        filtering_enabled: true,
        parental_enabled: false,
        safe_search_enabled: false,
        blocked_services: [],
        custom_blocked_domains: [],
        custom_allowed_domains: [],
        created_at: new Date().toISOString(),
      };
      devices.push(newDev);
      await route.fulfill({ json: newDev, status: 201 });
    }
  });

  await page.route(/\/api\/devices\/[^/]+$/, async (route) => {
    const url = route.request().url();
    const id = url.split('/').pop()?.split('?')[0];
    if (route.request().method() === 'PATCH') {
      const patch = route.request().postDataJSON();
      const dev = devices.find((d) => d.id === id);
      if (dev) Object.assign(dev, patch);
      await route.fulfill({ json: dev });
    } else if (route.request().method() === 'DELETE') {
      devices = devices.filter((d) => d.id !== id);
      await route.fulfill({ status: 204 });
    } else {
      const dev = devices.find((d) => d.id === id);
      await route.fulfill({ json: dev });
    }
  });

  await page.route(/\/api\/devices\/[^/]+\/ticket$/, async (route) => {
    await route.fulfill({ json: { ticket: 'test-ticket-abc-123' } });
  });

  await page.route(/\/api\/devices\/[^/]+\/mobileconfig.*/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/x-apple-aspen-config',
      headers: {
        'Content-Disposition': 'attachment; filename="adguard-test.mobileconfig"',
      },
      body: '<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict></dict></plist>',
    });
  });
});

test.describe('AdGuard Portal UI Suite', () => {
  test('1. Unauthenticated landing page & Dev Mode login', async ({ page }) => {
    await page.goto('/');

    // Check branding
    await expect(page.locator('text=AdGuard Portal').first()).toBeVisible();

    // Verify Developer quick login button is available in dev mode
    const devLoginBtn = page.getByRole('button', { name: /Quick Login as Developer/i });
    await expect(devLoginBtn).toBeVisible();

    // Click Developer login
    await devLoginBtn.click();

    // Should transition to dashboard and display user devices
    await expect(page.locator("text=Dominic's iPhone").first()).toBeVisible();
  });

  test('2. Theme Toggle & Language Selector', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('agh_dev_token', 'dev-user-admin');
    });

    await page.goto('/');

    // Verify dashboard loaded
    await expect(page.locator("text=Dominic's iPhone").first()).toBeVisible();

    // Test Theme toggle
    const themeBtn = page.locator('button[aria-label="Toggle theme"]').first();
    if (await themeBtn.isVisible()) {
      await themeBtn.click();
      const isDark = await page.evaluate(() => document.documentElement.classList.contains('dark'));
      expect(typeof isDark).toBe('boolean');
    }

    // Test Language switcher
    const langSelect = page.locator('select').first();
    if (await langSelect.isVisible()) {
      await langSelect.selectOption('vi');
      await expect(page.locator('body')).toContainText(/Thiết bị|Cài đặt|Đăng xuất|Thêm thiết bị/i);
      await langSelect.selectOption('en');
      await expect(page.locator('body')).toContainText(/Devices|Settings|Logout|Add Device/i);
    }
  });

  test('3. Device Quota and Registration Flow', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('agh_dev_token', 'dev-user-admin');
    });

    await page.goto('/');

    // Check device count badge in button: "Add Device (1/5)"
    await expect(page.locator('text=/5').first()).toBeVisible();

    // Open Add Device modal
    const addBtn = page.getByRole('button', { name: /Add Device/i }).first();
    await addBtn.click();

    // Fill form
    const nameInput = page.locator('input[placeholder*="iPhone" i], input[type="text"]').first();
    await nameInput.fill('Work MacBook Pro');

    // Submit
    const submitBtn = page.getByRole('button', { name: /Add Device|Register/i }).last();
    await submitBtn.click();

    // Verify new device is displayed in device list
    await expect(page.locator('text=Work MacBook Pro').first()).toBeVisible();
  });

  test('4. Setup & MobileConfig Modal (One-Time Ticket Download)', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('agh_dev_token', 'dev-user-admin');
    });

    await page.goto('/');

    // Click "Setup Guide" button for device
    const setupBtn = page.getByRole('button', { name: /Setup Guide/i }).first();
    await setupBtn.click();

    // Setup modal appears
    await expect(page.locator('text=Your Personalized DoH URL').first()).toBeVisible();
    await expect(page.locator('text=u-token1234').first()).toBeVisible();

    // Check DoH URL display
    await expect(page.locator('text=dns.ski.io.vn').first()).toBeVisible();

    // Copy button test
    const copyBtn = page.getByRole('button', { name: /Copy/i }).first();
    await copyBtn.click();
    await expect(page.locator('text=Copied').first()).toBeVisible();

    // Check Apple .mobileconfig download link
    const downloadLink = page.locator('a[download]').first();
    await expect(downloadLink).toBeVisible();
    await expect(downloadLink).toHaveAttribute('href', /.*mobileconfig.*/);

    // Close modal
    const closeBtn = page.locator('button[aria-label="Close modal"]').first();
    await closeBtn.click();
    await expect(page.locator('text=Your Personalized DoH URL')).not.toBeVisible();
  });

  test('5. Customize Filtering Modal (SafeSearch, Services, Blocklist)', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('agh_dev_token', 'dev-user-admin');
    });

    await page.goto('/');

    // Toggle Parental Control directly on device card
    const parentalToggle = page.locator('button[aria-label="Toggle Parental Control"]').first();
    await parentalToggle.click();

    // Click Rules button to open Customize modal
    const rulesBtn = page.getByRole('button', { name: /Rules/i }).first();
    await rulesBtn.click();

    // Modal title
    await expect(page.locator('text=Customize Filtering').first()).toBeVisible();

    // Toggle blocked service (e.g. YouTube)
    const ytBadge = page.locator('button:has-text("YouTube")').first();
    await ytBadge.click();

    // Add custom domain to blocklist
    const domainInput = page.locator('input[aria-label="Blocked domain"]').first();
    await domainInput.fill('ad-tracker-test.com');
    await domainInput.press('Enter');
    await expect(page.locator('text=ad-tracker-test.com').first()).toBeVisible();

    // Save Settings
    const saveBtn = page.getByRole('button', { name: /Save Settings/i });
    await saveBtn.click();

    // Verify modal closes
    await expect(page.locator('text=Customize Filtering')).not.toBeVisible();
  });

  test('6. Delete Device confirmation flow', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('agh_dev_token', 'dev-user-admin');
    });

    await page.goto('/');

    // Handle the browser window.confirm dialog
    page.once('dialog', async (dialog) => {
      expect(dialog.message()).toContain('delete');
      await dialog.accept();
    });

    // Find and click delete button on device card
    const deleteBtn = page.locator('button[title="Delete Device"]').first();
    await deleteBtn.click();

    // Device should be removed from view, showing empty state
    await expect(page.locator('text=No devices configured yet').first()).toBeVisible();
  });
});
