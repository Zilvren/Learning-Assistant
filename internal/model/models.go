package model

import (
	"encoding/json"
	"time"
)

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
	// ErrorProblemID is reserved for the internal legacy-error bridge.  It must
	// never be accepted from the public create-item API.
	ErrorProblemID *int `json:"-"`
}

type UpdateLibraryItemRequest struct {
	Name     *string   `json:"name"`
	Tags     *[]string `json:"tags"`
	Pinned   *bool     `json:"pinned"`
	ParentID *int64    `json:"parent_id"`
	// ParentSet distinguishes an omitted parent_id from an explicit JSON null.
	// The latter means “move to the root folder”.
	ParentSet     bool   `json:"-"`
	Conflict      string `json:"conflict"`
	ReviewEnabled *bool  `json:"review_enabled"`
}

// UnmarshalJSON 在当前模块中完成本文件定义的局部处理。
func (r *UpdateLibraryItemRequest) UnmarshalJSON(data []byte) error {
	type request struct {
		Name          *string         `json:"name"`
		Tags          *[]string       `json:"tags"`
		Pinned        *bool           `json:"pinned"`
		ParentID      json.RawMessage `json:"parent_id"`
		Conflict      string          `json:"conflict"`
		ReviewEnabled *bool           `json:"review_enabled"`
	}
	var raw request
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Name, r.Tags, r.Pinned = raw.Name, raw.Tags, raw.Pinned
	r.Conflict, r.ReviewEnabled = raw.Conflict, raw.ReviewEnabled
	r.ParentID, r.ParentSet = nil, false
	if raw.ParentID != nil {
		r.ParentSet = true
		if string(raw.ParentID) != "null" {
			var parent int64
			if err := json.Unmarshal(raw.ParentID, &parent); err != nil {
				return err
			}
			r.ParentID = &parent
		}
	}
	return nil
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
	MineruToken   string            `json:"mineru_token"` // OCR 服务的 token
	DeepSeekToken string            `json:"deepseek_token"`
	DeepSeekModel string            `json:"deepseek_model"`
	Username      string            `json:"username"` // 用户名
	DailyGoal     DailyGoalSettings `json:"daily_goal"`
}

// DailyGoalSettings keeps the user's repeatable daily study targets. A zero
// value is intentionally valid so existing installs can opt in gradually.
type DailyGoalSettings struct {
	ReviewTarget       int `json:"review_target"`
	FocusTargetMinutes int `json:"focus_target_minutes"`
	NoteTarget         int `json:"note_target"`
}

type ReviewRequest struct {
	Rating string `json:"rating"`
}

// NormalizeReviewRating accepts the four choices exposed by the review UI and
// keeps old clients compatible by treating an omitted value as "good".
func NormalizeReviewRating(value string) string {
	switch value {
	case "again", "hard", "good", "easy":
		return value
	default:
		return "good"
	}
}

// NextReview schedules a simple, transparent spaced-repetition progression.
// A rating changes the stage rather than merely marking every review complete.
func NextReview(stage, count int, rating string, reviewedAt time.Time) (nextStage, nextCount int, nextReview string) {
	rating = NormalizeReviewRating(rating)
	nextCount = count + 1
	switch rating {
	case "again":
		nextStage = 0
	case "hard":
		nextStage = stage
	case "easy":
		nextStage = stage + 2
	default:
		nextStage = stage + 1
	}
	if nextStage < 0 {
		nextStage = 0
	}
	intervals := []int{0, 1, 3, 7, 14, 30, 60, 120}
	index := nextStage
	if index >= len(intervals) {
		index = len(intervals) - 1
	}
	return nextStage, nextCount, reviewedAt.AddDate(0, 0, intervals[index]).Format(time.DateOnly)
}

type ReviewInboxItem struct {
	SourceType  string   `json:"source_type"`
	ID          int64    `json:"id"`
	Title       string   `json:"title"`
	Subject     string   `json:"subject,omitempty"`
	Tags        []string `json:"tags"`
	NextReview  string   `json:"next_review"`
	ReviewStage int      `json:"review_stage"`
	OverdueDays int      `json:"overdue_days"`
	Preview     string   `json:"preview,omitempty"`
}

type ActivityEvent struct {
	ID        int64     `json:"id"`
	Date      string    `json:"date"`
	EventType string    `json:"event_type"`
	SourceKey string    `json:"source_key"`
	Value     int       `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

type DailyPlan struct {
	Date             string            `json:"date"`
	Goal             DailyGoalSettings `json:"goal"`
	ReviewsCompleted int               `json:"reviews_completed"`
	FocusMinutes     int               `json:"focus_minutes"`
	NotesCreated     int               `json:"notes_created"`
}

type WeeklyReport struct {
	StartDate     string         `json:"start_date"`
	EndDate       string         `json:"end_date"`
	TotalActivity int            `json:"total_activity"`
	ActiveDays    int            `json:"active_days"`
	FocusMinutes  int            `json:"focus_minutes"`
	Reviews       int            `json:"reviews"`
	NotesCreated  int            `json:"notes_created"`
	ByEventType   map[string]int `json:"by_event_type"`
	WeakSubjects  []string       `json:"weak_subjects"`
}

type LearningRelation struct {
	ID         int64     `json:"id"`
	FromType   string    `json:"from_type"`
	FromID     int64     `json:"from_id"`
	ToType     string    `json:"to_type"`
	ToID       int64     `json:"to_id"`
	Label      string    `json:"label,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	TargetName string    `json:"target_name,omitempty"`
	TargetType string    `json:"target_type,omitempty"`
	TargetID   int64     `json:"target_id,omitempty"`
}

type SearchHit struct {
	SourceType string   `json:"source_type"`
	ID         int64    `json:"id"`
	Title      string   `json:"title"`
	Subtitle   string   `json:"subtitle,omitempty"`
	Tags       []string `json:"tags"`
	Snippet    string   `json:"snippet,omitempty"`
	MatchField string   `json:"match_field"`
}

type DocumentPreview struct {
	Kind  string                `json:"kind"`
	Title string                `json:"title"`
	Pages []DocumentPreviewPage `json:"pages"`
}

type DocumentPreviewPage struct {
	Title string     `json:"title"`
	Lines []string   `json:"lines,omitempty"`
	Rows  [][]string `json:"rows,omitempty"`
}

// AIChatMessage is application-level conversation history. It deliberately
// contains no provider-specific fields; the DeepSeek request itself is built
// by the official OpenAI client in the service layer.
type AIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AIChatRequest struct {
	Message  string          `json:"message"`
	History  []AIChatMessage `json:"history"`
	FolderID *int64          `json:"folder_id,omitempty"`
	ItemIDs  []int64         `json:"item_ids,omitempty"`
}

type AIChatSource struct {
	SourceType string `json:"source_type"`
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Excerpt    string `json:"excerpt,omitempty"`
}

type AIChatResponse struct {
	Answer  string         `json:"answer"`
	Model   string         `json:"model"`
	Sources []AIChatSource `json:"sources"`
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
	StartDate      string                `json:"start_date"`
	EndDate        string                `json:"end_date"`
	Total          int                   `json:"total"`
	ActiveDays     int                   `json:"active_days"`
	Days           []LearningActivityDay `json:"days"`
	AvailableYears []int                 `json:"available_years"`
}
