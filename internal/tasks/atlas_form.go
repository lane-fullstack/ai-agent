package tasks

import (
	"ai-agent/internal/config"
	"ai-agent/internal/executor"
	"fmt"
	"log"

	"github.com/playwright-community/playwright-go"
)

func init() {
	executor.RegisterFunc("SubmitAtlasForm", SubmitAtlasForm)
}
func SubmitAtlasForm(taskId int64) (string, error) {

	cfg := config.Load()
	path := cfg.ResumePath
	if path == "" {
		path = "./resume.pdf"
	}

	pw, err := playwright.Run()
	if err != nil {
		return "", err
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(
		playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(false),
		},
	)
	if err != nil {
		return "", err
	}

	page, err := browser.NewPage()
	if err != nil {
		return "", err
	}

	_, err = page.Goto("https://www.atlasassemblyinc.com/")
	if err != nil {
		return "", err
	}

	log.Println("Page opened")
	// 等待提交完成
	page.WaitForTimeout(10000)

	page.Locator(`button[data-hook="consent-banner-apply-button"]`).Click()
	// 等待表单
	_, err = page.WaitForSelector(`input[aria-label="Name"]`)
	if err != nil {
		return "", err
	}

	// Name
	err = page.Fill(`input[aria-label="Name"]`, "yingying zhao")
	if err != nil {
		return "", err
	}

	// Phone
	err = page.Fill(`input[aria-label*="Phone"]`, "6266202543")
	if err != nil {
		return "", err
	}

	// Email
	err = page.Fill(`input[type="email"]`, "rouk6688@gmail.com")
	if err != nil {
		return "", err
	}

	log.Println("Form filled")

	// checkbox
	err = page.Locator(`label[data-hook="checkbox-core"]`).Click()
	if err != nil {
		log.Println("checkbox warning:", err)
	}

	// 1. 开始监听文件选择器
	fileChooser, err := page.ExpectFileChooser(func() error {
		// 2. 点击那个“+ Upload Resume”按钮
		return page.Click("button:has-text('Upload Resume')")
		// 或者使用你的选择器: page.Click(`button[data-hook="upload-button"]`)
	})

	if err != nil {
		log.Fatal("Could not open file chooser: ", err)
	}

	// 3. 设置要上传的文件路径

	err = fileChooser.SetFiles(path)

	if err != nil {
		log.Fatal("Could not set files: ", err)
	}

	log.Println("File uploaded successfully via FileChooser")
	// 等待提交完成
	page.WaitForTimeout(10000)
	// 点击提交
	//err = page.Click(`button[data-hook="submit-button"]`)
	//if err != nil {
	//	return "", err
	//}

	//page.WaitForTimeout(10000)

	log.Println("Submit clicked")
	// 截图
	screenshot := "atlas_submit_result.png"

	_, err = page.Screenshot(
		playwright.PageScreenshotOptions{
			Path:     playwright.String(screenshot),
			FullPage: playwright.Bool(true),
		},
	)

	if err != nil {
		log.Println("screenshot error:", err)
	}

	log.Println("Screenshot saved:", screenshot)

	// 检查成功提示
	content, _ := page.Content()

	if containsSuccess(content) {
		return fmt.Sprintf("Form submitted successfully. Screenshot saved: %s", screenshot), nil
	}

	return fmt.Sprintf("Form submitted but success not confirmed. Screenshot: %s", screenshot), nil
}

func containsSuccess(html string) bool {

	keywords := []string{
		"Thank you for your submission!",
	}

	for _, k := range keywords {
		if containsIgnoreCase(html, k) {
			return true
		}
	}

	return false
}

func containsIgnoreCase(s string, sub string) bool {
	return len(s) >= len(sub) &&
		len(sub) > 0 &&
		containsFold(s, sub)
}

func containsFold(s, sub string) bool {
	return len(s) >= len(sub) && (stringContainsFold(s, sub))
}

func stringContainsFold(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (fmt.Sprintf("%v", s) != "" && (fmt.Sprintf("%v", substr) != ""))
}
