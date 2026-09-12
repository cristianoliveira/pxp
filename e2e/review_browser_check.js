// Playwright browser checks for the local visual-review workflow.
//
// The repository does not pin a Playwright test runner. This helper follows the
// existing browser-check convention: call `runReviewBrowserChecks(page, {url})`
// from a Playwright runner or an equivalent browser harness. It deliberately
// never submits or approves a valid review.
const assert = require('node:assert/strict');

async function focusBody(page) {
  await page.evaluate(() => {
    document.body.tabIndex = -1;
    document.body.focus();
  });
}

async function tab(page, count) {
  for (let index = 0; index < count; index += 1) {
    await page.keyboard.press('Tab');
  }
}

async function shiftTab(page, count) {
  for (let index = 0; index < count; index += 1) {
    await page.keyboard.press('Shift+Tab');
  }
}

async function listText(page) {
  return page.locator('#annotations li').allInnerTexts();
}

async function resetDraft(page, url) {
  await page.goto(url);
  await page.locator('#canvas').waitFor();
  await page.evaluate(() => localStorage.clear());
  await page.reload();
  await page.locator('#canvas').waitFor();
}

async function addBlankPin(page) {
  await page.keyboard.press('Enter');
  await page.waitForTimeout(50);
  assert.equal(await page.locator(':focus').getAttribute('id'), 'canvas');
  assert.equal(await page.locator('#interaction-status').textContent(), 'Annotation placed. Type a key to edit its note, or press Enter to continue.');
}

async function addCurrentPins(page) {
  await focusBody(page);
  await tab(page, 1);
  await page.keyboard.press('Enter');
  await tab(page, 3);
  await addBlankPin(page);
  await page.keyboard.press('Shift+ArrowRight');
  await addBlankPin(page);
  await page.keyboard.press('Shift+ArrowRight');
  await addBlankPin(page);
}

async function viewWithKeyboard(page, name) {
  const button = page.getByRole('button', {name, exact: true});
  await button.focus();
  await page.keyboard.press('Enter');
  await page.waitForTimeout(50);
  assert.equal(await button.getAttribute('aria-pressed'), 'true');
  assert.match(await page.locator('#canvas').getAttribute('aria-label'), new RegExp(name));
}

async function checkKeyboardViewsAndLifo(page, url) {
  await resetDraft(page, url);
  await addCurrentPins(page);

  const created = await listText(page);
  assert.equal(created.length, 3);
  assert.match(created[0], /@ 307,238/);
  assert.match(created[1], /@ 297,238/);
  assert.match(created[2], /@ 287,238/);

  await viewWithKeyboard(page, 'Reference');
  await viewWithKeyboard(page, 'Overlay');
  await viewWithKeyboard(page, 'Current');

  await focusBody(page);
  await tab(page, 9);
  assert.equal(await page.locator(':focus').getAttribute('aria-label'), 'Remove annotation 1');
  await page.keyboard.press('Enter');
  const afterRemove = await listText(page);
  assert.equal(afterRemove.length, 2);
  assert.match(afterRemove[0], /@ 297,238/);
  assert.match(await page.locator(':focus').textContent(), /@ 297,238/);

  return {created, afterRemove};
}

async function checkReturnToCanvas(page, url) {
  await resetDraft(page, url);
  const returnButton = page.getByRole('button', {name: 'Return to canvas', exact: true});
  await returnButton.focus();
  await page.keyboard.press('g');
  await page.keyboard.press('c');
  assert.equal(await page.locator(':focus').getAttribute('id'), 'canvas');
  assert.equal(await page.locator('#interaction-status').textContent(), 'Focus returned to canvas.');

  await returnButton.focus();
  await page.keyboard.press('Control+g');
  await page.keyboard.press('c');
  assert.equal(await page.locator(':focus').getAttribute('id'), 'return-to-canvas');

  await page.locator('#notes').focus();
  await page.locator('#notes').fill('gc remains text');
  await page.keyboard.press('g');
  await page.keyboard.press('c');
  assert.equal(await page.locator(':focus').getAttribute('id'), 'notes');
  assert.equal(await page.locator('#notes').inputValue(), 'gc remains text');

  await returnButton.focus();
  await page.evaluate(() => {
    document.activeElement.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'g', bubbles: true, isComposing: true,
    }));
  });
  await page.keyboard.press('c');
  assert.equal(await page.locator(':focus').getAttribute('id'), 'return-to-canvas');

  await returnButton.click();
  assert.equal(await page.locator(':focus').getAttribute('id'), 'canvas');
  return {status: await page.locator('#interaction-status').textContent()};
}

async function checkInlineAnnotationNote(page, url) {
  await resetDraft(page, url);
  await page.locator('#canvas').focus();
  await page.keyboard.press('Enter');
  assert.equal(await page.locator(':focus').getAttribute('id'), 'canvas');
  await page.keyboard.press('n');
  assert.equal(await page.locator(':focus').getAttribute('id'), 'annotation-note');
  assert.equal(await page.locator('#interaction-status').textContent(), 'Editing annotation note. Press Enter to save or Escape to cancel.');
  await page.keyboard.type('ote');
  await page.keyboard.press('Enter');
  assert.match((await listText(page))[0], /note/);
  assert.equal(await page.locator(':focus').getAttribute('id'), 'canvas');

  await resetDraft(page, url);
  await page.locator('#canvas').focus();
  await page.keyboard.press('Enter');
  await page.keyboard.press('d');
  await page.keyboard.press('Escape');
  assert.doesNotMatch((await listText(page))[0], /— d/);
  assert.equal(await page.locator(':focus').getAttribute('id'), 'canvas');

  await resetDraft(page, url);
  await page.locator('#type').selectOption('rectangle');
  await page.locator('#canvas').focus();
  await page.keyboard.press('Enter');
  await page.keyboard.press('x');
  assert.equal(await page.locator(':focus').getAttribute('id'), 'canvas');
  assert.equal(await page.locator('#annotations li').count(), 0);
  await page.keyboard.press('Escape');
  assert.equal(await page.locator('#interaction-status').textContent(), 'Rectangle cancelled.');

  return {saved: (await listText(page)).length};
}

async function checkEditCancelAndRectangleEscape(page, url) {
  await resetDraft(page, url);
  await addCurrentPins(page);

  await focusBody(page);
  await tab(page, 8);
  await page.keyboard.press('Enter');
  await page.keyboard.press('Control+A');
  await page.keyboard.type('saved note');
  await page.keyboard.press('Enter');
  assert.match((await listText(page))[0], /saved note/);

  await focusBody(page);
  await tab(page, 11);
  await page.keyboard.press('Enter');
  await page.keyboard.press('Control+A');
  await page.keyboard.type('cancelled note');
  await page.keyboard.press('Escape');
  assert.doesNotMatch((await listText(page))[1], /cancelled note/);
  assert.equal(await page.locator(':focus').textContent(), 'Edit note');

  await focusBody(page);
  await tab(page, 9);
  await page.keyboard.press('Enter');
  assert.equal((await page.locator('#annotations li').count()), 2);
  assert.match(await page.locator(':focus').textContent(), /@ 297,238/);

  await page.locator('#canvas').focus();
  await page.locator('#type').focus();
  await page.keyboard.press('r');
  await page.locator('#canvas').focus();
  const beforeCancel = await page.locator('#annotations li').count();
  await page.keyboard.press('Enter');
  await page.keyboard.press('ArrowRight');
  await page.keyboard.press('Escape');
  assert.equal(await page.locator('#annotations li').count(), beforeCancel);
  assert.equal(await page.locator('#interaction-status').textContent(), 'Rectangle cancelled.');

  return {remaining: beforeCancel};
}

async function checkDraftAndValidation(page, url) {
  await resetDraft(page, url);
  await page.locator('#canvas').focus();
  await page.keyboard.press('Enter');
  await page.locator('#notes').fill('draft survives reload');
  await viewWithKeyboard(page, 'Overlay');
  const draftKey = await page.evaluate(() => Object.keys(localStorage).find((key) => key.startsWith('pxp.review.draft.')));
  assert.ok(draftKey);

  await page.reload();
  assert.equal(await page.locator('#notes').inputValue(), 'draft survives reload');
  assert.equal(await page.locator('#annotations li').count(), 1);
  assert.equal(await page.locator('#view-status').textContent(), 'Viewing Overlay');
  assert.equal(await page.locator('#submit').isDisabled(), false);
  assert.equal(await page.locator('#approve').isDisabled(), false);

  await page.evaluate(() => localStorage.clear());
  await page.reload();
  await page.locator('#submit').focus();
  await page.keyboard.press('Enter');
  assert.equal(await page.locator('#status').textContent(), 'Add a note or annotation before submitting.');
  await page.locator('#notes').fill('draft only');
  await page.locator('#approve').focus();
  await page.keyboard.press('Enter');
  assert.equal(await page.locator('#status').textContent(), 'Remove notes and annotations before approving.');

  return {draftKey};
}

async function runReviewBrowserChecks(page, options) {
  assert.ok(options && options.url, 'runReviewBrowserChecks requires options.url');
  const returnToCanvas = await checkReturnToCanvas(page, options.url);
  const inline = await checkInlineAnnotationNote(page, options.url);
  const keyboard = await checkKeyboardViewsAndLifo(page, options.url);
  const editing = await checkEditCancelAndRectangleEscape(page, options.url);
  const draft = await checkDraftAndValidation(page, options.url);
  return {returnToCanvas, inline, keyboard, editing, draft};
}

module.exports = {runReviewBrowserChecks};
