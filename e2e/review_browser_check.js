// Playwright browser checks for the local visual-review workflow.
//
// The repository does not pin a Playwright test runner. This helper follows the
// existing browser-check convention: call `runReviewBrowserChecks(page, {url})`
// from a Playwright runner or an equivalent browser harness. Its final-decision
// check intercepts the request; it never submits or approves a live review.
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
  await page.locator('#canvas').focus();
  await addBlankPin(page);
  await page.keyboard.press('Shift+ArrowRight');
  await addBlankPin(page);
  await page.keyboard.press('Shift+ArrowRight');
  await addBlankPin(page);
}

async function checkImplementationContext(page, url) {
  await page.route('**/api/session', async (route) => {
    const response = await route.fetch();
    const session = await response.json();
    session.context = {
      title: 'Spacing <review>',
      what_changed: 'First line\nSecond line',
      what_to_test: '<script>not executable</script>',
      expected_outcome: 'The button aligns.',
      limitations: 'Desktop only.',
      source_reference: 'commit:abc123',
    };
    await route.fulfill({
      response,
      contentType: 'application/json',
      body: JSON.stringify(session),
    });
  });
  await page.goto(url);
  await page.locator('#canvas').waitFor();
  assert.equal(await page.locator('#context-heading').textContent(), 'Spacing <review>');
  assert.equal(await page.locator('[data-context-field="what_changed"]').textContent(), 'First line\nSecond line');
  assert.equal(await page.locator('[data-context-field="what_to_test"]').textContent(), '<script>not executable</script>');
  assert.equal(await page.locator('script').count(), 1);
  assert.equal(await page.locator('#implementation-context').getAttribute('open'), null);
  assert.equal(await page.locator('#context-title').textContent(), 'Show implementation context');
  await page.locator('#implementation-context summary').click();
  assert.equal(await page.locator('#implementation-context').getAttribute('open'), '');
  assert.equal(await page.locator('#context-title').textContent(), 'Hide implementation context');
  await page.locator('#implementation-context summary').press('Enter');
  assert.equal(await page.locator('#implementation-context').getAttribute('open'), null);
  await page.locator('#implementation-context summary').press('Space');
  assert.equal(await page.locator('#implementation-context').getAttribute('open'), '');
  await page.unroute('**/api/session');
  return {title: await page.locator('#context-heading').textContent()};
}

async function checkTabOrder(page, url) {
  await resetDraft(page, url);
  await focusBody(page);
  const expected = [
    'Implementation context',
    'Reference',
    'Current',
    'Overlay',
    'canvas',
    'Return to canvas',
    'Pin',
    'Rectangle',
    'Type',
    'Note',
    'General note',
  ];
  const actual = [];
  for (const name of expected) {
    await page.keyboard.press('Tab');
    actual.push(await page.locator(':focus').getAttribute('id') || await page.locator(':focus').innerText());
    if (name === 'Implementation context') {
      assert.equal(await page.locator(':focus').getAttribute('id'), 'context-title');
      assert.equal(await page.locator(':focus').textContent(), 'Show implementation context');
    }
    else if (name === 'canvas') assert.equal(actual.at(-1), 'canvas');
    else if (name === 'General note') assert.equal(await page.locator(':focus').getAttribute('id'), 'notes');
    else if (name === 'Type' || name === 'Note') assert.equal(await page.locator(':focus').getAttribute('id'), name === 'Type' ? 'type' : 'annotation-note');
    else if (name === 'Return to canvas') assert.equal(await page.locator(':focus').getAttribute('aria-label'), name);
    else assert.equal(await page.locator(':focus').textContent(), name);
  }
  return actual;
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

  await page.getByRole('button', {name: 'Remove annotation 1', exact: true}).focus();
  await page.keyboard.press('Enter');
  const afterRemove = await listText(page);
  assert.equal(afterRemove.length, 2);
  assert.match(afterRemove[0], /@ 297,238/);
  assert.match(await page.locator(':focus').textContent(), /@ 297,238/);

  return {created, afterRemove};
}

async function checkCanvasViewShortcuts(page, url) {
  await resetDraft(page, url);
  await page.locator('#canvas').focus();
  await page.keyboard.press('Enter');
  const currentText = (await listText(page))[0];
  const coordinate = currentText.match(/@ ([^—]+)/)?.[1];
  assert.ok(coordinate, `missing current annotation coordinate: ${currentText}`);

  await page.keyboard.press('Shift+R');
  assert.equal(await page.locator('#view-status').textContent(), 'Viewing Reference');
  assert.equal(await page.locator('#canvas').getAttribute('aria-label'), 'Reference screenshot annotation canvas');
  assert.equal(await page.locator(':focus').getAttribute('id'), 'canvas');
  assert.equal((await listText(page))[0], currentText);

  await page.keyboard.press('Enter');
  assert.match((await listText(page))[0], new RegExp(`Reference point @ ${coordinate}`));
  await page.keyboard.press('Shift+O');
  assert.equal(await page.locator('#view-status').textContent(), 'Viewing Overlay');
  await page.keyboard.press('Shift+C');
  assert.equal(await page.locator('#view-status').textContent(), 'Viewing Current');
  assert.equal(await page.locator(':focus').getAttribute('id'), 'canvas');

  await resetDraft(page, url);
  await page.locator('#type').selectOption('rectangle');
  await page.locator('#canvas').focus();
  await page.keyboard.press('Enter');
  await page.keyboard.press('Shift+R');
  assert.equal(await page.locator('#view-status').textContent(), 'Viewing Current');
  assert.equal(await page.locator('#interaction-status').textContent(), 'Finish or cancel the rectangle before switching views.');
  assert.equal(await page.locator('#annotations li').count(), 0);
  await page.keyboard.press('Escape');
  await page.keyboard.press('Shift+R');
  assert.equal(await page.locator('#view-status').textContent(), 'Viewing Reference');

  return {coordinate, annotations: await listText(page)};
}

async function checkAnnotationModes(page, url) {
  await resetDraft(page, url);
  await page.locator('#type').selectOption('rectangle');
  await page.locator('#canvas').focus();
  await page.keyboard.press('Enter');
  await page.locator('#type').selectOption('point');
  const draft = await page.evaluate(() => {
    const key = Object.keys(localStorage).find((item) => item.startsWith('pxp.review.draft.'));
    return key ? JSON.parse(localStorage.getItem(key)) : null;
  });
  assert.equal(draft.keyboardRectangleStart, null);
  assert.equal(await page.locator('#interaction-status').textContent(), 'Pin mode selected.');
  await page.keyboard.press('Enter');
  assert.match((await listText(page))[0], /Current point/);

  await page.locator('[data-annotation-mode="rectangle"]').click();
  assert.equal(await page.locator('#type').inputValue(), 'rectangle');
  assert.equal(await page.locator('[data-annotation-mode="rectangle"]').getAttribute('aria-pressed'), 'true');
  assert.equal(await page.locator('[data-annotation-mode="point"]').getAttribute('aria-pressed'), 'false');
  return {mode: await page.locator('#type').inputValue()};
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
  await page.keyboard.press('g');
  await page.locator('#notes').focus();
  await page.keyboard.press('c');
  assert.equal(await page.locator(':focus').getAttribute('id'), 'notes');
  await returnButton.focus();
  await page.keyboard.press('c');
  assert.equal(await page.locator(':focus').getAttribute('id'), 'return-to-canvas');

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

  await page.getByRole('button', {name: 'Edit annotation 1', exact: true}).focus();
  await page.keyboard.press('Enter');
  await page.keyboard.press('Control+A');
  await page.keyboard.type('saved note');
  await page.keyboard.press('Enter');
  assert.match((await listText(page))[0], /saved note/);

  await page.getByRole('button', {name: 'Edit annotation 2', exact: true}).focus();
  await page.keyboard.press('Enter');
  await page.keyboard.press('Control+A');
  await page.keyboard.type('cancelled note');
  await page.keyboard.press('Escape');
  assert.doesNotMatch((await listText(page))[1], /cancelled note/);
  assert.equal(await page.locator(':focus').textContent(), 'Edit note');

  await page.getByRole('button', {name: 'Remove annotation 1', exact: true}).focus();
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

async function checkDecisionConfirmation(page, url) {
  await resetDraft(page, url);
  await page.locator('#notes').fill('Keep this draft intact while reviewing the decision.');
  const draftBefore = await page.evaluate(() => {
    const key = Object.keys(localStorage).find((item) => item.startsWith('pxp.review.draft.'));
    return key ? localStorage.getItem(key) : null;
  });

  await page.locator('#submit').click();
  assert.equal(await page.locator('#decision-dialog').getAttribute('open'), '');
  assert.equal(await page.locator('#decision-dialog').getAttribute('aria-modal'), 'true');
  assert.equal(await page.locator('#decision-confirm').textContent(), 'Send feedback to agent');
  assert.match(await page.locator('#decision-summary').textContent(), /Round 1/);
  assert.match(await page.locator('#decision-summary').textContent(), /1 general note/);
  assert.equal(await page.locator(':focus').getAttribute('id'), 'decision-confirm');
  await page.keyboard.press('Tab');
  assert.equal(await page.locator(':focus').getAttribute('id'), 'decision-back');
  await page.keyboard.press('Escape');
  assert.equal(await page.locator('#decision-dialog').getAttribute('open'), null);
  assert.equal(await page.locator(':focus').getAttribute('id'), 'submit');
  assert.equal(await page.locator('#notes').inputValue(), 'Keep this draft intact while reviewing the decision.');
  assert.equal(await page.evaluate(() => {
    const key = Object.keys(localStorage).find((item) => item.startsWith('pxp.review.draft.'));
    return key ? localStorage.getItem(key) : null;
  }), draftBefore);

  await page.locator('#approve').click();
  assert.equal(await page.locator('#decision-dialog').getAttribute('open'), null);
  assert.equal(await page.locator('#status').textContent(), 'Remove notes and annotations before approving.');
  await page.locator('#notes').fill('');
  await page.locator('#approve').focus();
  await page.keyboard.press('Enter');
  assert.equal(await page.locator('#decision-dialog').getAttribute('open'), '');
  assert.equal(await page.locator('#decision-confirm').textContent(), 'Approve and finish');
  assert.match(await page.locator('#decision-summary').textContent(), /Review ends/);
  await page.locator('#decision-back').click();
  assert.equal(await page.locator(':focus').getAttribute('id'), 'approve');

  let submittedPayload = null;
  await page.route('**/api/feedback', async (route) => {
    submittedPayload = JSON.parse(route.request().postData() || '{}');
    await route.fulfill({status: 200, contentType: 'application/json', body: '{}'});
  });
  await page.locator('#notes').fill('Submit only after confirmation.');
  await page.locator('#submit').click();
  await page.locator('#decision-confirm').click();
  await page.waitForTimeout(50);
  assert.equal(await page.locator('#decision-dialog').getAttribute('open'), null);
  assert.equal(await page.locator('#status').textContent(), 'Feedback saved. You may close this page.');
  assert.equal(submittedPayload.decision, 'submitted');
  assert.equal(submittedPayload.notes, 'Submit only after confirmation.');
  await page.unroute('**/api/feedback');

  return {draftPreserved: true, submittedPayload, submitAction: 'Send feedback to agent', approveAction: 'Approve and finish'};
}

async function runReviewBrowserChecks(page, options) {
  assert.ok(options && options.url, 'runReviewBrowserChecks requires options.url');
  const context = await checkImplementationContext(page, options.url);
  const tabOrder = await checkTabOrder(page, options.url);
  const modes = await checkAnnotationModes(page, options.url);
  const viewShortcuts = await checkCanvasViewShortcuts(page, options.url);
  const returnToCanvas = await checkReturnToCanvas(page, options.url);
  const inline = await checkInlineAnnotationNote(page, options.url);
  const keyboard = await checkKeyboardViewsAndLifo(page, options.url);
  const editing = await checkEditCancelAndRectangleEscape(page, options.url);
  const draft = await checkDraftAndValidation(page, options.url);
  const decisionConfirmation = await checkDecisionConfirmation(page, options.url);
  return {context, tabOrder, modes, viewShortcuts, returnToCanvas, inline, keyboard, editing, draft, decisionConfirmation};
}

module.exports = {runReviewBrowserChecks};
