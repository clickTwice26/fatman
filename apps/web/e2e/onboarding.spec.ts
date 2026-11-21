import { expect, test } from "@playwright/test";

const MAILPIT_URL = process.env.MAILPIT_URL ?? "http://localhost:8025";

async function fetchOtpFromMailpit(email: string): Promise<string> {
  for (let attempt = 0; attempt < 20; attempt++) {
    const res = await fetch(
      `${MAILPIT_URL}/api/v1/search?query=${encodeURIComponent(email)}`,
    );
    if (res.ok) {
      const json = (await res.json()) as {
        messages?: Array<{ Snippet?: string }>;
      };
      const snippet = json.messages?.[0]?.Snippet ?? "";
      const match = snippet.match(/\b(\d{6})\b/);
      if (match?.[1]) {
        return match[1];
      }
    }
    await new Promise((r) => setTimeout(r, 250));
  }
  throw new Error(`Timed out waiting for OTP email to ${email} in Mailpit`);
}

test("passwordless email-OTP sign-up, 3-step onboarding, and session persistence", async ({
  page,
}) => {
  const unique = `${Date.now()}_${Math.floor(Math.random() * 10000)}`;
  const email = `e2e.${unique}@example.com`;
  const username = `stride_${unique.slice(-6)}`;

  await page.goto("/auth");
  await expect(
    page.getByRole("heading", { name: /join or sign in with email/i }),
  ).toBeVisible();

  await page.getByLabel(/email address/i).fill(email);
  await page.getByRole("button", { name: /send 6-digit code/i }).click();

  await expect(page.getByRole("heading", { name: /check your inbox/i })).toBeVisible();

  const otpCode = await fetchOtpFromMailpit(email);
  await page.getByLabel(/6-digit verification code/i).fill(otpCode);
  await page.getByRole("button", { name: /verify & continue/i }).click();

  // Redirects new member to onboarding wizard
  await expect(page).toHaveURL(/\/onboarding$/);
  await page.getByLabel(/username/i).fill(username);
  await page.getByLabel(/display name/i).fill("Taylor Stride");
  await page.getByRole("button", { name: /next: privacy & goals/i }).click();

  await expect(page.getByRole("heading", { name: /your weight privacy/i })).toBeVisible();
  await page.getByLabel(/starting weight/i).fill("96");
  await page.getByLabel(/goal weight/i).fill("82");
  await page.getByRole("button", { name: /next: quest focus/i }).click();

  await expect(page.getByRole("heading", { name: /what habits do you want to build/i })).toBeVisible();
  await page.getByRole("button", { name: /complete setup/i }).click();

  // Redirects to dashboard with personalized welcome
  await expect(page).toHaveURL(/\/dashboard$/);
  await expect(page.getByRole("heading", { name: /welcome back, taylor stride/i })).toBeVisible();
  await expect(page.getByText(`@${username}`)).toBeVisible();

  // Reloading preserves session via HttpOnly refresh cookie
  await page.reload();
  await expect(page.getByRole("heading", { name: /welcome back, taylor stride/i })).toBeVisible();
});
