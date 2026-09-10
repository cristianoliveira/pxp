// Coordinator-owned checks for A/B candidates. This validates one fresh source tree;
// it never reads agent-submitted screenshots or names either condition.
async function checkCandidate(page, options) {
  const expected = [
    ["Wire – New Features", "Uploading 1.2MB..."],
    ["IMG_20240303_142512.jpg", "Uploading 1.2MB..."],
    ["Wire – New Features", "Couldn't upload file"],
    ["Design Feedback", "Uploaded 1.2MB"],
    ["Wire – Features Slides", "Uploaded 1.2MB"],
    ["Notes for the pitch", "Uploaded 1.2MB"],
    ["Wire – New Features", "Uploaded 1.2MB"],
  ];
  const describe = async () => {
    const rows = page.getByRole("listitem");
    const rowCount = await rows.count();
    let content = rowCount === expected.length;
    for (let index = 0; content && index < expected.length; index++) {
      for (const text of expected[index]) {
        content = await rows.nth(index).getByText(text, { exact: true }).isVisible();
        if (!content) break;
      }
    }
    return { content, rowCount };
  };
  const capture = async (path) => {
    await page.evaluate(() => document.fonts.ready);
    await page.screenshot({ path, omitBackground: true, animations: "disabled", caret: "hide" });
  };
  const open = async () => {
    await page.goto(options.url);
    await page.getByRole("region").first().waitFor();
    return describe();
  };

  const initial = await open();
  await capture(options.initial);

  const cancel = page.getByRole("button", { name: "Cancel Wire – New Features", exact: true }).first();
  let interactions = await cancel.count() === 1;
  if (interactions) await cancel.click();
  const cancelAll = page.getByRole("button", { name: /cancel all/i }).first();
  interactions = interactions && await cancelAll.count() === 1;
  if (interactions) await cancelAll.click();
  const afterCancel = page.getByRole("listitem");
  interactions = interactions && await afterCancel.count() === 5;
  const retry = page.getByRole("button", { name: "Retry Wire – New Features", exact: true });
  interactions = interactions && await retry.count() === 1;
  if (interactions) await retry.click();
  interactions = interactions
    && await page.getByText("Couldn't upload file", { exact: true }).count() === 0
    && await page.getByRole("listitem").count() === 5;

  await open();
  const toggle = page.getByRole("button", { name: "Collapse upload list", exact: true });
  let keyboard = await toggle.count() === 1;
  if (keyboard) {
    await toggle.focus();
    await page.keyboard.press("Enter");
    keyboard = await page.getByRole("listitem").count() === 0
      && await page.getByRole("button", { name: "Expand upload list", exact: true }).count() === 1;
  }
  if (keyboard) await page.keyboard.press("Enter");
  keyboard = keyboard && (await describe()).content;

  const final = await open();
  await capture(options.final);
  return {
    content: initial.content && final.content,
    interactions,
    keyboard,
    initialRows: initial.rowCount,
    finalRows: final.rowCount,
    capture: await page.evaluate(() => ({ width: innerWidth, height: innerHeight, dpr: devicePixelRatio, userAgent: navigator.userAgent })),
  };
}
