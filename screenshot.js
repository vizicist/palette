const { chromium } = require('playwright');

(async () => {
    const browser = await chromium.launch();
    const page = await browser.newPage();
    await page.goto('http://127.0.0.1:3330/');
    await page.waitForTimeout(2000); // Wait for status polling
    await page.screenshot({ path: 'webui-screenshot.png', fullPage: true });
    console.log('Screenshot saved to webui-screenshot.png');
    await browser.close();
})();