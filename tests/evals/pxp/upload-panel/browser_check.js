// Same checks for every control. No knowledge of variant names or expected failures.
async function checkPanel(page, options) {
  const expected = [
    ["Wire – New Features", "Uploading 1.2MB..."],
    ["IMG_20240303_142512.jpg", "Uploading 1.2MB..."],
    ["Wire – New Features", "Couldn't upload file"],
    ["Design Feedback", "Uploaded 1.2MB"],
    ["Wire – Features Slides", "Uploaded 1.2MB"],
    ["Notes for the pitch", "Uploaded 1.2MB"],
    ["Wire – New Features", "Uploaded 1.2MB"],
  ];
  await page.goto(options.url);
  await page.getByRole("region").first().waitFor();
  await page.evaluate(() => document.fonts.ready);
  const rows = page.getByRole("listitem");
  const rowCount = await rows.count();
  let content = rowCount === expected.length;
  if (content) {
    for (let i = 0; i < expected.length; i++) {
      for (const text of expected[i]) {
        content = content && await rows.nth(i).getByText(text, {exact:true}).isVisible();
      }
    }
  }
  const rowText = await rows.allInnerTexts();
  await page.screenshot({path:options.screenshot, omitBackground:true, animations:"disabled", caret:"hide"});
  const before = await page.getByText("Uploading 1.2MB...", {exact:true}).count();
  const retryButton = page.getByRole("button", {name:"Retry Wire – New Features", exact:true});
  let retry = false;
  if (await retryButton.count() === 1) {
    await retryButton.click();
    const failed = await page.getByText("Couldn't upload file", {exact:true}).count();
    const uploading = await page.getByText("Uploading 1.2MB...", {exact:true}).count();
    retry = failed === 0 && uploading === before + 1 && await rows.count() === rowCount;
  }
  return {
    content, retry, rowCount, rowText,
    capture:await page.evaluate(() => ({width:innerWidth,height:innerHeight,dpr:devicePixelRatio,userAgent:navigator.userAgent})),
  };
}
