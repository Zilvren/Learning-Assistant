package handlers

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	models "study-tracker-go/internal/model"
	"study-tracker-go/internal/repository"
	"study-tracker-go/internal/service"
)

const libraryMaxUploadSize = 200 << 20

func parseLibraryID(c *gin.Context) (int64, bool) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "ID格式错误"})
		return 0, false
	}
	return id, true
}
func parseParent(raw string) *int64 {
	if raw == "" || raw == "root" {
		return nil
	}
	id, e := strconv.ParseInt(raw, 10, 64)
	if e != nil {
		return nil
	}
	return &id
}

func ListLibraryItems(c *gin.Context) {
	items, e := service.ListLibrary(c.Request.Context(), repository.LibraryFilter{ParentID: parseParent(c.Query("parent_id")), Kind: c.Query("kind"), Query: c.Query("q"), Tag: c.Query("tag"), ReviewOnly: c.Query("review") == "true", DueOnly: c.Query("due") == "true", Trashed: c.Query("trashed") == "true"})
	if e != nil {
		respondError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}
func SearchLibrary(c *gin.Context) {
	items, e := service.ListLibrary(c.Request.Context(), repository.LibraryFilter{Query: c.Query("q"), Kind: c.Query("kind"), Tag: c.Query("tag")})
	if e != nil {
		respondError(c, http.StatusInternalServerError, e)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func ListLibraryTags(c *gin.Context) {
	tags, err := service.ListLibraryTags(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

func ListLibraryReviews(c *gin.Context) {
	items, err := service.DueLibraryReviews(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func ReviewLibraryNote(c *gin.Context) {
	id, ok := parseLibraryID(c)
	if !ok {
		return
	}
	item, err := service.ReviewLibraryNote(c.Request.Context(), id)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, item)
}
func GetLibraryItem(c *gin.Context) {
	id, ok := parseLibraryID(c)
	if !ok {
		return
	}
	item, e := service.GetLibraryItem(c.Request.Context(), id)
	if e != nil {
		respondError(c, http.StatusNotFound, e)
		return
	}
	c.JSON(http.StatusOK, item)
}
func CreateLibraryItem(c *gin.Context) {
	var req models.CreateLibraryItemRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}
	if req.Kind == "note" && req.MimeType == "" {
		req.MimeType = "text/markdown; charset=utf-8"
	}
	item, e := service.CreateLibraryItem(c.Request.Context(), req, []byte{})
	if e != nil {
		respondError(c, http.StatusBadRequest, e)
		return
	}
	c.JSON(http.StatusCreated, item)
}
func UpdateLibraryItem(c *gin.Context) {
	id, ok := parseLibraryID(c)
	if !ok {
		return
	}
	var req models.UpdateLibraryItemRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}
	item, e := service.UpdateLibraryItem(c.Request.Context(), id, req)
	if e != nil {
		respondError(c, http.StatusBadRequest, e)
		return
	}
	c.JSON(http.StatusOK, item)
}
func DeleteLibraryItem(c *gin.Context) {
	id, ok := parseLibraryID(c)
	if !ok {
		return
	}
	if e := service.TrashLibraryItem(c.Request.Context(), id); e != nil {
		respondError(c, http.StatusNotFound, e)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已移入回收站"})
}
func RestoreLibraryItem(c *gin.Context) {
	id, ok := parseLibraryID(c)
	if !ok {
		return
	}
	item, e := service.RestoreLibraryItem(c.Request.Context(), id)
	if e != nil {
		respondError(c, http.StatusBadRequest, e)
		return
	}
	c.JSON(http.StatusOK, item)
}
func PurgeLibraryItem(c *gin.Context) {
	id, ok := parseLibraryID(c)
	if !ok {
		return
	}
	if e := service.PurgeLibraryItem(c.Request.Context(), id); e != nil {
		respondError(c, http.StatusBadRequest, e)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已永久删除"})
}
func DuplicateLibraryItem(c *gin.Context) {
	id, ok := parseLibraryID(c)
	if !ok {
		return
	}
	var body struct {
		ParentID *int64 `json:"parent_id"`
	}
	_ = c.ShouldBindJSON(&body)
	item, e := service.DuplicateLibraryItem(c.Request.Context(), id, body.ParentID)
	if e != nil {
		respondError(c, http.StatusBadRequest, e)
		return
	}
	c.JSON(http.StatusCreated, item)
}
func GetLibraryContent(c *gin.Context) {
	id, ok := parseLibraryID(c)
	if !ok {
		return
	}
	body, item, e := service.ReadLibraryContent(c.Request.Context(), id)
	if e != nil {
		respondError(c, http.StatusNotFound, e)
		return
	}
	ct := item.MimeType
	if ct == "" {
		ct = "application/octet-stream"
	}
	c.Header("ETag", fmt.Sprintf("\"%d\"", item.CurrentVersion))
	if item.Kind == "file" {
		c.Header("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": item.Name}))
	}
	c.Data(http.StatusOK, ct, body)
}
func SaveLibraryContent(c *gin.Context) {
	id, ok := parseLibraryID(c)
	if !ok {
		return
	}
	var req models.SaveLibraryContentRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}
	item, e := service.SaveLibraryContent(c.Request.Context(), id, req)
	if e != nil {
		if strings.Contains(e.Error(), "版本冲突") {
			c.JSON(http.StatusConflict, gin.H{"detail": "内容已被其他操作更新", "code": "version_conflict"})
			return
		}
		respondError(c, http.StatusBadRequest, e)
		return
	}
	c.JSON(http.StatusOK, item)
}
func UploadLibraryFile(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, libraryMaxUploadSize)
	file, header, e := c.Request.FormFile("file")
	if e != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请选择不超过 200MB 的文件"})
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".md": true, ".txt": true, ".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true, ".pdf": true, ".docx": true, ".xlsx": true, ".pptx": true}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "不支持该文件类型"})
		return
	}
	data, e := io.ReadAll(io.LimitReader(file, libraryMaxUploadSize+1))
	if e != nil {
		respondError(c, http.StatusBadRequest, e)
		return
	}
	if len(data) > libraryMaxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "文件不能超过 200MB"})
		return
	}
	kind := "file"
	if ext == ".md" || ext == ".txt" {
		kind = "note"
	}
	ct := header.Header.Get("Content-Type")
	if ct == "" {
		ct = mime.TypeByExtension(ext)
	}
	item, e := service.CreateLibraryItem(c.Request.Context(), models.CreateLibraryItemRequest{ParentID: parseParent(c.PostForm("parent_id")), Kind: kind, Name: filepath.Base(header.Filename), MimeType: ct}, data)
	if e != nil {
		respondError(c, http.StatusBadRequest, e)
		return
	}
	c.JSON(http.StatusCreated, item)
}
func ListLibraryVersions(c *gin.Context) {
	id, ok := parseLibraryID(c)
	if !ok {
		return
	}
	items, e := service.LibraryVersions(c.Request.Context(), id)
	if e != nil {
		respondError(c, http.StatusNotFound, e)
		return
	}
	c.JSON(http.StatusOK, gin.H{"versions": items})
}
func RestoreLibraryVersion(c *gin.Context) {
	id, ok := parseLibraryID(c)
	if !ok {
		return
	}
	vid, e := strconv.ParseInt(c.Param("versionId"), 10, 64)
	if e != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "版本ID错误"})
		return
	}
	item, e := service.RestoreLibraryVersion(c.Request.Context(), id, vid)
	if e != nil {
		respondError(c, http.StatusBadRequest, e)
		return
	}
	c.JSON(http.StatusOK, item)
}

func BatchLibraryItems(c *gin.Context) {
	var req struct {
		Action   string  `json:"action"`
		IDs      []int64 `json:"ids"`
		ParentID *int64  `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "批量操作参数错误"})
		return
	}
	for _, id := range req.IDs {
		var err error
		switch req.Action {
		case "trash":
			err = service.TrashLibraryItem(c.Request.Context(), id)
		case "restore":
			_, err = service.RestoreLibraryItem(c.Request.Context(), id)
		case "purge":
			err = service.PurgeLibraryItem(c.Request.Context(), id)
		case "move":
			_, err = service.UpdateLibraryItem(c.Request.Context(), id, models.UpdateLibraryItemRequest{ParentID: req.ParentID, Conflict: "keep_both"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"detail": "不支持的批量操作"})
			return
		}
		if err != nil {
			respondError(c, http.StatusBadRequest, err)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "批量操作完成", "count": len(req.IDs)})
}
