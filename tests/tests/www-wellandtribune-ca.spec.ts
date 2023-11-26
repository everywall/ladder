import { expect, test } from "@playwright/test";

const paywallText = "This article is exclusive to subscribers.";
const articleURL =
  "https://www.wellandtribune.ca/news/niagara-region/niagara-transit-commission-rejects-council-request-to-reduce-its-budget-increase/article_e9fb424c-8df5-58ae-a6c3-3648e2a9df66.html";

const ladderURL = "http://localhost:8080";
let domain = (new URL(articleURL)).host;

test(`${domain} has paywall by default`, async ({ page }) => {
  await page.goto(articleURL);
  await expect(page.getByText(paywallText)).toBeVisible();
});

test(`${domain} + Ladder doesn't have paywall`, async ({ page }) => {
  await page.goto(`${ladderURL}/${articleURL}`);
  await expect(page.getByText(paywallText)).toBeVisible();
});
