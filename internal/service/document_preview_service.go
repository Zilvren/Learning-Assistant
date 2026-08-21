package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	models "study-tracker-go/internal/model"
)

const officePreviewMaxSize = 25 << 20
const officePreviewMaxUncompressed = 60 << 20
const officePreviewMaxEntries = 400
const officePreviewMaxRatio = 100

// GetDocumentPreview 在业务层中执行当前流程或局部处理。
func GetDocumentPreview(ctx context.Context, id int64) (models.DocumentPreview, error) {
	body, item, err := ReadLibraryContent(ctx, id)
	if err != nil {
		return models.DocumentPreview{}, err
	}
	if len(body) > officePreviewMaxSize {
		return models.DocumentPreview{}, fmt.Errorf("文档超过 25MB，暂不支持在线文本预览")
	}
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return models.DocumentPreview{}, fmt.Errorf("文档格式无效")
	}
	if err := validateOfficeArchive(reader); err != nil {
		return models.DocumentPreview{}, err
	}
	preview := models.DocumentPreview{Title: item.Name, Pages: []models.DocumentPreviewPage{}}
	switch item.MimeType {
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		preview.Kind = "docx"
		content, err := zipText(reader, "word/document.xml", "p")
		if err != nil {
			return preview, err
		}
		preview.Pages = append(preview.Pages, models.DocumentPreviewPage{Title: "文档内容", Lines: limitLines(content, 500)})
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		preview.Kind = "pptx"
		for _, name := range zipNames(reader, "ppt/slides/slide", ".xml") {
			content, err := zipText(reader, name, "p")
			if err != nil {
				continue
			}
			preview.Pages = append(preview.Pages, models.DocumentPreviewPage{Title: "幻灯片 " + slideNumber(name), Lines: limitLines(content, 120)})
			if len(preview.Pages) >= 30 {
				break
			}
		}
		if len(preview.Pages) == 0 {
			return preview, fmt.Errorf("PPTX 中没有可预览的文字")
		}
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		preview.Kind = "xlsx"
		pages, err := spreadsheetPreview(reader)
		if err != nil {
			return preview, err
		}
		preview.Pages = pages
	default:
		return preview, fmt.Errorf("此文件类型不支持在线预览")
	}
	return preview, nil
}

// validateOfficeArchive 在业务层中执行当前流程或局部处理。
func validateOfficeArchive(reader *zip.Reader) error {
	if len(reader.File) > officePreviewMaxEntries {
		return fmt.Errorf("文档包含过多内容，暂不支持在线预览")
	}
	var total uint64
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.UncompressedSize64 > officePreviewMaxUncompressed || total > officePreviewMaxUncompressed-file.UncompressedSize64 {
			return fmt.Errorf("文档解压后的内容过大，暂不支持在线预览")
		}
		if file.UncompressedSize64 > 0 && (file.CompressedSize64 == 0 || file.UncompressedSize64/file.CompressedSize64 > officePreviewMaxRatio) {
			return fmt.Errorf("文档压缩比例异常，无法安全预览")
		}
		total += file.UncompressedSize64
	}
	return nil
}

// zipFile 在业务层中执行当前流程或局部处理。
func zipFile(reader *zip.Reader, name string) (*zip.File, error) {
	for _, file := range reader.File {
		if file.Name == name {
			return file, nil
		}
	}
	return nil, fmt.Errorf("文档缺少 %s", filepath.Base(name))
}

// zipText 在业务层中执行当前流程或局部处理。
func zipText(reader *zip.Reader, name, breakElement string) ([]string, error) {
	file, err := zipFile(reader, name)
	if err != nil {
		return nil, err
	}
	stream, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	decoder := xml.NewDecoder(io.LimitReader(stream, officePreviewMaxSize))
	lines, current := []string{}, strings.Builder{}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch value := token.(type) {
		case xml.CharData:
			current.WriteString(string(value))
		case xml.EndElement:
			if value.Name.Local == breakElement {
				line := strings.TrimSpace(current.String())
				if line != "" {
					lines = append(lines, line)
				}
				current.Reset()
			}
		}
	}
	if line := strings.TrimSpace(current.String()); line != "" {
		lines = append(lines, line)
	}
	return lines, nil
}

// zipNames 在业务层中执行当前流程或局部处理。
func zipNames(reader *zip.Reader, prefix, suffix string) []string {
	result := []string{}
	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, prefix) && strings.HasSuffix(file.Name, suffix) {
			result = append(result, file.Name)
		}
	}
	sort.Slice(result, func(i, j int) bool { return zipNameIndex(result[i]) < zipNameIndex(result[j]) })
	return result
}

// zipNameIndex 在业务层中执行当前流程或局部处理。
func zipNameIndex(name string) int {
	value := strings.TrimSuffix(filepath.Base(name), ".xml")
	value = strings.TrimPrefix(value, "slide")
	value = strings.TrimPrefix(value, "sheet")
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

// slideNumber 在业务层中执行当前流程或局部处理。
func slideNumber(name string) string {
	if index := zipNameIndex(name); index > 0 {
		return strconv.Itoa(index)
	}
	return strings.TrimSuffix(filepath.Base(name), ".xml")
}

// limitLines 在业务层中执行当前流程或局部处理。
func limitLines(lines []string, maximum int) []string {
	if len(lines) <= maximum {
		return lines
	}
	return append(append([]string{}, lines[:maximum]...), "…其余内容请下载文件查看")
}

// spreadsheetPreview 在业务层中执行当前流程或局部处理。
func spreadsheetPreview(reader *zip.Reader) ([]models.DocumentPreviewPage, error) {
	shared, _ := zipText(reader, "xl/sharedStrings.xml", "si")
	pages := []models.DocumentPreviewPage{}
	for _, name := range zipNames(reader, "xl/worksheets/sheet", ".xml") {
		file, err := zipFile(reader, name)
		if err != nil {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			continue
		}
		rows, parseErr := parseSheet(stream, shared)
		stream.Close()
		if parseErr != nil {
			continue
		}
		pages = append(pages, models.DocumentPreviewPage{Title: "工作表 " + slideNumber(name), Rows: rows})
		if len(pages) >= 8 {
			break
		}
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("XLSX 中没有可预览的工作表")
	}
	return pages, nil
}

// parseSheet 在业务层中执行当前流程或局部处理。
func parseSheet(source io.Reader, shared []string) ([][]string, error) {
	decoder := xml.NewDecoder(io.LimitReader(source, officePreviewMaxSize))
	rows := [][]string{}
	currentRow, currentCell := []string{}, ""
	cellType, value := "", strings.Builder{}
	inCell := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch node := token.(type) {
		case xml.StartElement:
			switch node.Name.Local {
			case "row":
				currentRow = []string{}
			case "c":
				inCell, currentCell, cellType = true, "", ""
				value.Reset()
				for _, attr := range node.Attr {
					if attr.Name.Local == "r" {
						currentCell = attr.Value
					}
					if attr.Name.Local == "t" {
						cellType = attr.Value
					}
				}
			case "v", "t":
				if inCell {
					value.Reset()
				}
			}
		case xml.CharData:
			if inCell {
				value.WriteString(string(node))
			}
		case xml.EndElement:
			switch node.Name.Local {
			case "c":
				column := spreadsheetColumn(currentCell)
				for len(currentRow) <= column {
					currentRow = append(currentRow, "")
				}
				text := strings.TrimSpace(value.String())
				if cellType == "s" {
					if index, err := strconv.Atoi(text); err == nil && index >= 0 && index < len(shared) {
						text = shared[index]
					}
				}
				currentRow[column] = text
				inCell = false
			case "row":
				if len(currentRow) > 0 {
					rows = append(rows, currentRow)
				}
				if len(rows) >= 120 {
					return rows, nil
				}
			}
		}
	}
	return rows, nil
}

// spreadsheetColumn 在业务层中执行当前流程或局部处理。
func spreadsheetColumn(reference string) int {
	column := 0
	for _, letter := range reference {
		if letter < 'A' || letter > 'Z' {
			break
		}
		column = column*26 + int(letter-'A'+1)
	}
	if column == 0 {
		return 0
	}
	return column - 1
}
