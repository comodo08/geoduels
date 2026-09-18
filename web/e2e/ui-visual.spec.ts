import { expect, test } from '@playwright/test';

test('lobby baseline visual', async ({ page }) => {
  await page.goto('/');
  await expect(page).toHaveScreenshot('lobby.png', { fullPage: true });
});
