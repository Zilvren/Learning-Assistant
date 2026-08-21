package service

import (
	"archive/zip"
	"bytes"
	"context"
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

	"study-tracker-go/internal/repository"
)

const mineruBaseURL = "https://mineru.net/api/v4"

const (
	OCRMaxUploadSize     = 50 * 1024 * 1024
	ocrResultZipMaxSize  = 200 * 1024 * 1024
	ocrMarkdownMaxSize   = 5 * 1024 * 1024
	ocrImageMaxSize      = 12 * 1024 * 1024
	ocrTotalImageMaxSize = 24 * 1024 * 1024
)

var ocrHTTPClient = &http.Client{Timeout: 60 * time.Second}

// OCRImageBytes 对图片字节数据进行 OCR 识别，返回 Markdown
func OCRImageBytes(ctx context.Context, imageBytes []byte, fileName string) (string, error) {
	return OCRFileBytes(ctx, imageBytes, fileName, "image/png")
}

// OCRFileBytes 接收 API 已校验的源文件名和 MIME 类型。保留这些值对 PDF 至关重要：若把所有上传都作为 PNG 发送，MinerU 会拒绝原本有效的文档。
// OCRFileBytes 对上传的图片字节执行 OCR，并返回可写入笔记的识别文本。
func OCRFileBytes(ctx context.Context, imageBytes []byte, fileName, mimeType string) (string, error) {
	task, err := createOCRTask(ctx, imageBytes, fileName, mimeType)
	if err != nil {
		return "", err
	}
	if err := processOCRTask(ctx, task.ID); err != nil {
		return "", err
	}
	completed, err := GetOCRTask(ctx, task.ID)
	if err != nil {
		return "", err
	}
	return completed.ResultMarkdown, nil
}

// StartOCRTask 会在向客户端返回 202 前持久化上传内容；实际的 MinerU 轮询在请求生命周期外运行，可从 OCR 任务中心查看或重试。
func StartOCRTask(ctx context.Context, imageBytes []byte, fileName, mimeType string) (repository.OCRTask, error) {
	task, err := createOCRTask(ctx, imageBytes, fileName, mimeType)
	if err != nil {
		return task, err
	}
	asyncCtx := asyncOCRContext(ctx)
	go func(id int64) { _ = processOCRTask(asyncCtx, id) }(task.ID)
	return task, nil
}

// createOCRTask 在业务层中执行当前流程或局部处理。
func createOCRTask(ctx context.Context, imageBytes []byte, fileName, mimeType string) (repository.OCRTask, error) {
	if len(imageBytes) == 0 || len(imageBytes) > OCRMaxUploadSize {
		return repository.OCRTask{}, fmt.Errorf("OCR 文件必须在 1B 到 50MB 之间")
	}
	fileName, normalizedMimeType, err := normalizeOCRSource(fileName, mimeType)
	if err != nil {
		return repository.OCRTask{}, err
	}
	repos, err := repositories(ctx)
	if err != nil {
		return repository.OCRTask{}, err
	}
	inputHash, _, err := repository.StoreBlob(bytes.NewReader(imageBytes))
	if err != nil {
		return repository.OCRTask{}, err
	}
	task := repository.OCRTask{
		Provider:       "mineru",
		Status:         "queued",
		SourceFilename: fileName,
		MimeType:       normalizedMimeType,
		FileSize:       int64(len(imageBytes)),
		InputBlobHash:  inputHash,
	}
	taskID, err := repos.OCRTasks.Create(ctx, task)
	if err != nil {
		return repository.OCRTask{}, err
	}
	task.ID = taskID
	return task, nil
}

// processOCRTask 在业务层中执行当前流程或局部处理。
func processOCRTask(ctx context.Context, taskID int64) error {
	repos, err := repositories(ctx)
	if err != nil {
		return err
	}
	task, err := repos.OCRTasks.Get(ctx, taskID)
	if err != nil {
		return err
	}
	imageBytes, err := repository.ReadBlob(task.InputBlobHash)
	if err != nil {
		markOCRFailed(ctx, repos, taskID, err)
		return err
	}

	token, err := getMinerUToken(ctx)
	if err != nil {
		markOCRFailed(ctx, repos, taskID, err)
		return err
	}

	batchID, uploadURL, err := createMinerUBatch(ctx, token, task.SourceFilename)
	if err != nil {
		markOCRFailed(ctx, repos, taskID, err)
		return err
	}
	if err := repos.OCRTasks.Update(ctx, taskID, repository.OCRTask{
		Status:  "uploading",
		BatchID: batchID,
	}); err != nil {
		return err
	}

	// 上传图片到 MinerU 返回的预签名 URL
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(imageBytes))
	if err != nil {
		return err
	}
	resp, err := ocrHTTPClient.Do(req)
	if err != nil {
		markOCRFailed(ctx, repos, taskID, err)
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		err := fmt.Errorf("上传图片失败：HTTP %d", resp.StatusCode)
		markOCRFailed(ctx, repos, taskID, err)
		return err
	}

	if err := repos.OCRTasks.Update(ctx, taskID, repository.OCRTask{Status: "processing"}); err != nil {
		return err
	}

	// 轮询等待识别完成
	zipURL, err := pollMinerUResult(ctx, token, batchID)
	if err != nil {
		markOCRFailed(ctx, repos, taskID, err)
		return err
	}

	markdown, err := downloadAndExtractMarkdown(ctx, zipURL)
	if err != nil {
		markOCRFailed(ctx, repos, taskID, err)
		return err
	}
	now := time.Now()
	if err := repos.OCRTasks.Update(ctx, taskID, repository.OCRTask{
		Status:         "succeeded",
		ResultMarkdown: markdown,
		FinishedAt:     &now,
	}); err != nil {
		return err
	}
	return nil
}

// asyncOCRContext 在业务层中执行当前流程或局部处理。
func asyncOCRContext(ctx context.Context) context.Context {
	result := context.Background()
	if app, err := appFor(ctx); err == nil {
		result = ContextWithApp(result, app)
	}
	if userID, ok := UserIDFromContext(ctx); ok {
		result = ContextWithUserID(result, userID)
	}
	return result
}

// GetOCRTask 在业务层中执行当前流程或局部处理。
func GetOCRTask(ctx context.Context, id int64) (repository.OCRTask, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return repository.OCRTask{}, err
	}
	return repos.OCRTasks.Get(ctx, id)
}

// ListOCRTasks 在业务层中执行当前流程或局部处理。
func ListOCRTasks(ctx context.Context) ([]repository.OCRTask, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return nil, err
	}
	return repos.OCRTasks.List(ctx, 30)
}

// RetryOCRTask 在业务层中执行当前流程或局部处理。
func RetryOCRTask(ctx context.Context, id int64) (repository.OCRTask, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return repository.OCRTask{}, err
	}
	task, err := repos.OCRTasks.Get(ctx, id)
	if err != nil {
		return task, err
	}
	if task.Status != "failed" {
		return task, fmt.Errorf("只有失败的 OCR 任务可以重试")
	}
	if task.InputBlobHash == "" {
		return task, fmt.Errorf("原始 OCR 文件已不可用")
	}
	if err := repos.OCRTasks.Update(ctx, id, repository.OCRTask{Status: "queued"}); err != nil {
		return task, err
	}
	task.Status = "queued"
	asyncCtx := asyncOCRContext(ctx)
	go func() { _ = processOCRTask(asyncCtx, id) }()
	return task, nil
}

// normalizeOCRSource 在业务层中构造、编码或标准化数据。
func normalizeOCRSource(fileName, mimeType string) (string, string, error) {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	if fileName == "." || fileName == "" {
		return "", "", fmt.Errorf("OCR 文件名无效")
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	allowed := map[string]string{
		".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
		".webp": "image/webp", ".gif": "image/gif", ".pdf": "application/pdf",
	}
	expected, ok := allowed[ext]
	if !ok {
		return "", "", fmt.Errorf("OCR 仅支持图片和 PDF 文件")
	}
	if mimeType != "" && mimeType != "application/octet-stream" && !strings.EqualFold(strings.Split(mimeType, ";")[0], expected) {
		return "", "", fmt.Errorf("OCR 文件类型与扩展名不匹配")
	}
	return fileName, expected, nil
}

// getMinerUToken 获取 MinerU Token（优先 config.json，其次环境变量）
func getMinerUToken(ctx context.Context) (string, error) {
	config, _ := loadConfig(ctx)
	token := strings.TrimSpace(config.MineruToken)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("MINERU_TOKEN"))
	}
	if token == "" {
		return "", fmt.Errorf("MinerU token not configured")
	}
	return token, nil
}

// markOCRFailed 在业务层中完成本文件定义的局部处理。
func markOCRFailed(ctx context.Context, repos repository.Repositories, taskID int64, err error) {
	now := time.Now()
	_ = repos.OCRTasks.Update(ctx, taskID, repository.OCRTask{
		Status:       "failed",
		ErrorMessage: err.Error(),
		FinishedAt:   &now,
	})
}

// createMinerUBatch 在业务层中创建或更新相应状态。
func createMinerUBatch(ctx context.Context, token string, fileName string) (batchID string, uploadURL string, err error) {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mineruBaseURL+"/file-urls/batch", bytes.NewReader(jsonBody))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := ocrHTTPClient.Do(req)
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

// pollMinerUResult 在业务层中完成本文件定义的局部处理。
func pollMinerUResult(ctx context.Context, token string, batchID string) (string, error) {
	deadline := time.Now().Add(5 * time.Minute)
	var lastErr error

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		zipURL, state, taskID, err := queryBatchResult(ctx, token, batchID)
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
			zipURL, state, err := queryTaskResult(ctx, token, taskID)
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

// queryBatchResult 在业务层中完成本文件定义的局部处理。
func queryBatchResult(ctx context.Context, token string, batchID string) (zipURL string, state string, taskID string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mineruBaseURL+"/extract-results/batch/"+batchID, nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := ocrHTTPClient.Do(req)
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

// queryTaskResult 在业务层中完成本文件定义的局部处理。
func queryTaskResult(ctx context.Context, token string, taskID string) (zipURL string, state string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mineruBaseURL+"/extract/task/"+taskID, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := ocrHTTPClient.Do(req)
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

// downloadAndExtractMarkdown 在业务层中完成本文件定义的局部处理。
func downloadAndExtractMarkdown(ctx context.Context, zipURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zipURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := ocrHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载 OCR 结果失败：HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, ocrResultZipMaxSize+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > ocrResultZipMaxSize {
		return "", fmt.Errorf("OCR 结果文件过大")
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}

	imageMap := map[string]string{}
	markdown := ""
	var totalImageBytes int64

	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, "full.md") {
			content, err := readOCRZipFile(file, ocrMarkdownMaxSize)
			if err != nil {
				return "", err
			}
			markdown = string(content)
			continue
		}
		if strings.Contains(file.Name, "images/") && !file.FileInfo().IsDir() {
			content, err := readOCRZipFile(file, ocrImageMaxSize)
			if err != nil {
				continue
			}
			totalImageBytes += int64(len(content))
			if totalImageBytes > ocrTotalImageMaxSize {
				return "", fmt.Errorf("OCR 结果图片过大")
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

// readOCRZipFile 在业务层中读取并整理所需数据。
func readOCRZipFile(file *zip.File, maxSize int64) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, maxSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxSize {
		return nil, fmt.Errorf("OCR 结果中的 %s 过大", filepath.Base(file.Name))
	}
	return data, nil
}

// replaceMarkdownImages 在业务层中创建或更新相应状态。
func replaceMarkdownImages(markdown string, imageMap map[string]string) string {
	re := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	return re.ReplaceAllStringFunc(markdown, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		src := parts[2]
		if dataURI, ok := imageMap[src]; ok {
			return `![` + parts[1] + `](` + dataURI + `)`
		}
		base := filepath.Base(src)
		for name, dataURI := range imageMap {
			if filepath.Base(name) == base {
				return `![` + parts[1] + `](` + dataURI + `)`
			}
		}
		return match
	})
}
