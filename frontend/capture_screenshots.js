
import puppeteer from 'puppeteer';
import fs from 'fs';
import path from 'path';

(async () => {
  const browser = await puppeteer.launch({
    headless: "new",
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });
  const page = await browser.newPage();
  
  // Set viewport for a nice desktop capability
  await page.setViewport({width: 1280, height: 800});

  try {
    // 1. Investors Tab
    console.log("Navigating to Investors...");
    await page.goto('http://localhost:5173', {waitUntil: 'networkidle0'});
    
    // Click "Investors" in nav
    // Assuming text content or link. I'll click the link with href="/investors" or text "Investors"
    const investorsLink = await page.$('a[href="/investors"]');
    if (investorsLink) {
        await investorsLink.click();
        await new Promise(r => setTimeout(r, 1000)); // Wait for render
        
        const shotPath = path.resolve('../docs/images/investors_tab.png');
        await page.screenshot({path: shotPath});
        console.log(`Saved ${shotPath}`);
    } else {
        console.error("Could not find Investors link");
    }

    // 2. Gmail Tab
    console.log("Navigating to Gmail...");
    const gmailLink = await page.$('a[href="/gmail"]');
    if (gmailLink) {
        await gmailLink.click();
        await new Promise(r => setTimeout(r, 1000)); // Wait for render
        
        const shotPath = path.resolve('../docs/images/gmail_tab.png');
        await page.screenshot({path: shotPath});
        console.log(`Saved ${shotPath}`);
    } else {
       console.error("Could not find Gmail link");
    }

  } catch (e) {
    console.error(e);
  } finally {
    await browser.close();
  }
})();
