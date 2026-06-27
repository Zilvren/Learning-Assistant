package service

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"study-tracker-go/models"
	"study-tracker-go/store"
)

const mineruBaseURL = "https://mineru.net/api/v4"

// OCRImageBytes 对图片字节数据进行 OCR 识别，返回 Markdown
func OCRImageBytes(imageBytes []byte, fileName string) (string, error) {
	token, err := getMinerUToken()
	if err != nil {
		return "", err
	}

	batchID, uploadURL, err := createMinerUBatch(token, fileName)
	if err != nil {
		return "", err
	}

	// 上传图片到 MinerU 返回的预签名 URL
	req, err := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(imageBytes))
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("上传图片失败：HTTP %d", resp.StatusCode)
	}

	// 轮询等待识别完成
	zipURL, err := pollMinerUResult(token, batchID)
	if err != nil {
		return "", err
	}

	return downloadAndExtractMarkdown(zipURL)
}

// getMinerUToken 获取 MinerU Token（优先 config.json，其次环境变量）
func getMinerUToken() (string, error) {
	var config models.Config
	_ = store.LoadJSON("config.json", &config)
	token := strings.TrimSpace(config.MineruToken)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("MINERU_TOKEN"))
	}
	if token == "" {
		return "", fmt.Errorf("MinerU token not configured")
	}
	return token, nil
}

func createMinerUBatch(token string, fileName string) (batchID string, uploadURL string, err error) {
	body := map[string]interface{}{
		"files":          []map[string]string{{"name": fileName}},
		"model_version":  "vlm",
		"enable_formula": true,
		"enable_table":   false,
		"language":       "ch",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest(http.MethodPost, mineruBaseURL+"/file-urls/batch", bytes.NewReader(jsonBody))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("创建 MinerU 批次失败：HTTP %d", resp.StatusCode)
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			BatchID  string   `json:"batch_id"`
			FileURLs []string `json:"file_urls"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", "", err
	}
	if result.Code != 0 {
		return "", "", fmt.Errorf("MinerU error: %s", result.Msg)
	}
	if result.Data.BatchID == "" || len(result.Data.FileURLs) == 0 {
		return "", "", fmt.Errorf("MinerU 没有返回上传地址")
	}
	return result.Data.BatchID, result.Data.FileURLs[0], nil
}

func pollMinerUResult(token string, batchID string) (string, error) {
	deadline := time.Now().Add(5 * time.Minute)
	var lastErr error

	for time.Now().Before(deadline) {
		time.Sleep(3 * time.Second)

		zipURL, state, taskID, err := queryBatchResult(token, batchID)
		if err != nil {
			lastErr = err
			continue
		}
		if zipURL != "" {
			return zipURL, nil
		}
		if state == "failed" {
			return "", fmt.Errorf("MinerU OCR 失败")
		}
		if taskID != "" {
			zipURL, state, err := queryTaskResult(token, taskID)
			if err != nil {
				lastErr = err
				continue
			}
			if zipURL != "" {
				return zipURL, nil
			}
			if state == "failed" {
				return "", fmt.Errorf("MinerU OCR 失败")
			}
		}
	}

	if lastErr != nil {
		return "", fmt.Errorf("MinerU OCR 超时，最后一次错误：%w", lastErr)
	}
	return "", fmt.Errorf("MinerU OCR 超时")
}

func queryBatchResult(token string, batchID string) (zipURL string, state string, taskID string, err error) {
	req, err := http.NewRequest(http.MethodGet, mineruBaseURL+"/extract-results/batch/"+batchID, nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("查询 MinerU 批次失败：HTTP %d", resp.StatusCode)
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			ExtractResult []struct {
				State      string `json:"state"`
				FullZipURL string `json:"full_zip_url"`
				TaskID     string `json:"task_id"`
			} `json:"extract_result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", "", err
	}
	if result.Code != 0 {
		return "", "", "", fmt.Errorf("MinerU error: %s", result.Msg)
	}
	if len(result.Data.ExtractResult) == 0 {
		return "", "", "", nil
	}
	item := result.Data.ExtractResult[0]
	if item.State == "done" {
		return item.FullZipURL, item.State, item.TaskID, nil
	}
	return "", item.State, item.TaskID, nil
}

func queryTaskResult(token string, taskID string) (zipURL string, state string, err error) {
	req, err := http.NewRequest(http.MethodGet, mineruBaseURL+"/extract/task/"+taskID, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("查询 MinerU 任务失败：HTTP %d", resp.StatusCode)
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			State      string `json:"state"`
			FullZipURL string `json:"full_zip_url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}
	if result.Code != 0 {
		return "", "", fmt.Errorf("MinerU error: %s", result.Msg)
	}
	if result.Data.State == "done" {
		return result.Data.FullZipURL, result.Data.State, nil
	}
	return "", result.Data.State, nil
}

func downloadAndExtractMarkdown(zipURL string) (string, error) {
	resp, err := http.Get(zipURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载 OCR 结果失败：HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}

	imageMap := map[string]string{}
	markdown := ""

	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, "full.md") {
			content, err := readOCRZipFile(file)
			if err != nil {
				return "", err
			}
			markdown = string(content)
			continue
		}
		if strings.Contains(file.Name, "images/") && !file.FileInfo().IsDir() {
			content, err := readOCRZipFile(file)
			if err != nil {
				continue
			}
			ext := strings.ToLower(filepath.Ext(file.Name))
			mime := "image/png"
			if ext == ".jpg" || ext == ".jpeg" {
				mime = "image/jpeg"
			}
			imageMap[file.Name] = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(content)
		}
	}

	if markdown == "" {
		return "", fmt.Errorf("OCR 结果中没有 full.md")
	}

	return replaceMarkdownImages(markdown, imageMap), nil
}

func readOCRZipFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func replaceMarkdownImages(markdown string, imageMap map[string]string) string {
	re := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	return re.ReplaceAllStringFunc(markdown, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		src := parts[2]
		if dataURI, ok := imageMap[src]; ok {
			return `<img src="` + dataURI + `" width="400">`
		}
		base := filepath.Base(src)
		for name, dataURI := range imageMap {
			if filepath.Base(name) == base {
				return `<img src="` + dataURI + `" width="400">`
			}
		}
		return match
	})
}
