package model

import "time"

type LibraryItem struct {
	ID             int64      `json:"id"`
	ParentID       *int64     `json:"parent_id"`
	OriginalParent *int64     `json:"original_parent_id,omitempty"`
	Kind           string     `json:"kind"`
	Name           string     `json:"name"`
	MimeType       string     `json:"mime_type,omitempty"`
	Size           int64      `json:"size"`
	Tags           []string   `json:"tags"`
	Pinned         bool       `json:"pinned"`
	CurrentVersion int        `json:"current_version"`
	ErrorProblemID *int       `json:"error_problem_id,omitempty"`
	ReviewEnabled  bool       `json:"review_enabled"`
	ReviewCount    int        `json:"review_count"`
	ReviewStage    int        `json:"review_stage"`
	LastReview     *time.Time `json:"last_review,omitempty"`
	NextReview     string     `json:"next_review,omitempty"`
	BlobHash       string     `json:"blob_hash,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

type LibraryVersion struct {
	ID        int64     `json:"id"`
	ItemID    int64     `json:"item_id"`
	Version   int       `json:"version"`
	BlobHash  string    `json:"blob_hash"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateLibraryItemRequest struct {
	ParentID      *int64   `json:"parent_id"`
	Kind          string   `json:"kind"`
	Name          string   `json:"name"`
	MimeType      string   `json:"mime_type"`
	Tags          []string `json:"tags"`
	ReviewEnabled bool     `json:"review_enabled"`
}

type UpdateLibraryItemRequest struct {
	Name          *string   `json:"name"`
	Tags          *[]string `json:"tags"`
	Pinned        *bool     `json:"pinned"`
	ParentID      *int64    `json:"parent_id"`
	Conflict      string    `json:"conflict"`
	ReviewEnabled *bool     `json:"review_enabled"`
}

type SaveLibraryContentRequest struct {
	Content     string `json:"content"`
	BaseVersion int    `json:"base_version"`
	Checkpoint  bool   `json:"checkpoint"`
	Force       bool   `json:"force"`
}

type ErrorProblem struct {
	ID          int      `json:"id"`           // 编号（整数）
	Subject     string   `json:"subject"`      // 科目，比如"数学"（字符串）
	Title       string   `json:"title"`        // 标题（字符串）
	Question    string   `json:"question"`     // 题目内容（支持 Markdown）
	Wrong       string   `json:"wrong"`        // 错误答案
	Correct     string   `json:"correct"`      // 正确答案
	Reason      string   `json:"reason"`       // 错误原因
	Tags        []string `json:"tags"`         // 标签列表，比如 ["函数","积分"]
	ReasonTags  []string `json:"reason_tags"`  // 错误原因标签
	Created     string   `json:"created"`      // 创建时间 "2024-06-17 10:30:00"
	ReviewCount int      `json:"review_count"` // 复习了几次
	LastReview  *string  `json:"last_review"`  // 上次复习时间
	ReviewStage int      `json:"review_stage"` // 当前在艾宾浩斯第几阶段
	NextReview  string   `json:"next_review"`  // 下次该哪天复习 "2024-06-18"
}

// Config 是用户设置
type Config struct {
	MineruToken string `json:"mineru_token"` // OCR 服务的 token
	Username    string `json:"username"`     // 用户名
}

// AddErrorRequest 是创建错题时的请求体
type AddErrorRequest struct {
	ParentID   *int64   `json:"parent_id,omitempty"`
	Subject    string   `json:"subject"`
	Question   string   `json:"question"`
	Title      string   `json:"title"`
	Wrong      string   `json:"wrong"`
	Correct    string   `json:"correct"`
	Reason     string   `json:"reason"`
	Tags       []string `json:"tags"`
	ReasonTags []string `json:"reason_tags"`
}

// UpdateErrorRequest 是更新错题时的请求体
// 所有字段都用指针，因为前端可能只传部分字段
// nil 表示"这个字段没传，不要更新"
type UpdateErrorRequest struct {
	Subject    *string   `json:"subject"`
	Title      *string   `json:"title"`
	Question   *string   `json:"question"`
	Wrong      *string   `json:"wrong"`
	Correct    *string   `json:"correct"`
	Reason     *string   `json:"reason"`
	Tags       *[]string `json:"tags"`
	ReasonTags *[]string `json:"reason_tags"`
}

// DailyPushResult 是每日推送的返回数据
type DailyPushResult struct {
	Date         string            `json:"date"`
	TotalErrors  int               `json:"total_errors"`
	Reviewed     int               `json:"reviewed"`
	DueCount     int               `json:"due_count"`
	OverdueCount int               `json:"overdue_count"`
	Knowledge    map[string]string `json:"knowledge"`
	WeakErrors   []ErrorProblem    `json:"weak_errors"`
	Advice       string            `json:"advice"`
	DueNotes     []LibraryItem     `json:"due_notes,omitempty"`
	TopTags      []string          `json:"top_tags,omitempty"`
}

type LearningActivityDay struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type LearningActivityResponse struct {
	StartDate  string                `json:"start_date"`
	EndDate    string                `json:"end_date"`
	Total      int                   `json:"total"`
	ActiveDays int                   `json:"active_days"`
	Days       []LearningActivityDay `json:"days"`
}
