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
	// ErrorProblemID 保留给内部旧版错题桥接，绝不能从公开的创建条目 API 接收。
	ErrorProblemID *int `json:"-"`
}

type UpdateLibraryItemRequest struct {
	Name     *string   `json:"name"`
	Tags     *[]string `json:"tags"`
	Pinned   *bool     `json:"pinned"`
	ParentID *int64    `json:"parent_id"`
	// ParentSet 用于区分省略 parent_id 与显式 JSON null；后者表示“移动到根文件夹”。
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
	MineruToken     string                  `json:"mineru_token"` // OCR 服务的 token
	DeepSeekToken   string                  `json:"deepseek_token"`
	DeepSeekModel   string                  `json:"deepseek_model"`
	AIChatContext   []AIConversationMessage `json:"ai_chat_context,omitempty"`
	AIConversations []AIConversation        `json:"ai_conversations,omitempty"`
	AITurns         []AITurn                `json:"ai_turns,omitempty"`
	Username        string                  `json:"username"` // 用户名
	DailyGoal       DailyGoalSettings       `json:"daily_goal"`
}

// DailyGoalSettings 保存用户可重复执行的每日学习目标。零值被刻意视为有效，以便现有安装逐步启用。
type DailyGoalSettings struct {
	ReviewTarget       int `json:"review_target"`
	FocusTargetMinutes int `json:"focus_target_minutes"`
	NoteTarget         int `json:"note_target"`
}

type ReviewRequest struct {
	Rating string `json:"rating"`
}

// NormalizeReviewRating 接受复习界面提供的四个选项，并将省略值视为“good”以兼容旧客户端。
func NormalizeReviewRating(value string) string {
	switch value {
	case "again", "hard", "good", "easy":
		return value
	default:
		return "good"
	}
}

// NextReview 安排简单透明的间隔重复进程；评分会改变阶段，而不是只把每次复习标为完成。
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

// AIChatMessage 是应用级对话历史。它刻意不包含提供商专属字段；实际的 DeepSeek 请求由业务层中的官方 OpenAI 客户端构建。
type AIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIConversationMessage 是在浏览器中恢复 AI 对话时使用的持久化表示；它将仅展示的元数据与 AIChatRequest 发送的精简提供商历史分离。
type AIConversationMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	Scope      string         `json:"scope,omitempty"`
	Model      string         `json:"model,omitempty"`
	Sources    []AIChatSource `json:"sources,omitempty"`
	Audit      []AIToolAudit  `json:"audit,omitempty"`
	Incomplete bool           `json:"incomplete,omitempty"`
}

// AIConversationMemory 是与聊天正文分离的长期工作记忆。它只保存可核对的目标、决定和未完成事项，不包含模型推理。
type AIConversationMemory struct {
	Goal       string   `json:"goal,omitempty"`
	Completed  []string `json:"completed,omitempty"`
	Decisions  []string `json:"decisions,omitempty"`
	References []string `json:"references,omitempty"`
	Blockers   []string `json:"blockers,omitempty"`
	NextStep   string   `json:"next_step,omitempty"`
}

// AIConversation 是一个范围独立且可持久化的 AI 对话。每条记录拥有自己的消息和资料范围，因此切换对话不会将前一对话的上下文泄露到下一次请求。
type AIConversation struct {
	ID               string                  `json:"id"`
	Title            string                  `json:"title"`
	FolderID         *int64                  `json:"folder_id,omitempty"`
	ItemIDs          []int64                 `json:"item_ids,omitempty"`
	ChatOnly         bool                    `json:"chat_only,omitempty"`
	Messages         []AIConversationMessage `json:"messages"`
	ContextSummary   string                  `json:"context_summary,omitempty"`
	ContextMemory    *AIConversationMemory   `json:"context_memory,omitempty"`
	HarnessSessionID string                  `json:"harness_session_id,omitempty"`
	ArchivedAt       *time.Time              `json:"archived_at,omitempty"`
}

type AIChatRequest struct {
	Message          string                `json:"message"`
	History          []AIChatMessage       `json:"history"`
	FolderID         *int64                `json:"folder_id,omitempty"`
	ItemIDs          []int64               `json:"item_ids,omitempty"`
	ChatOnly         bool                  `json:"chat_only,omitempty"`
	ConversationID   string                `json:"conversation_id,omitempty"`
	HarnessSessionID string                `json:"harness_session_id,omitempty"`
	ContextSummary   string                `json:"context_summary,omitempty"`
	ContextMemory    *AIConversationMemory `json:"context_memory,omitempty"`
	CompactContext   bool                  `json:"compact_context,omitempty"`
	TurnID           string                `json:"-"`
}

// AITurnStatus 描述一个可恢复 AI 任务在应用侧的生命周期；不包含模型隐藏推理。
type AITurnStatus string

const (
	AITurnQueued          AITurnStatus = "queued"
	AITurnRunning         AITurnStatus = "running"
	AITurnWaitingApproval AITurnStatus = "waiting_approval"
	AITurnCompleted       AITurnStatus = "completed"
	AITurnFailed          AITurnStatus = "failed"
	AITurnCancelled       AITurnStatus = "cancelled"
)

// AIWriteApproval 是模型尝试产生副作用前交给用户确认的写入意图；正文仅保存于该用户的私有设置中。
type AIWriteApproval struct {
	ID              string     `json:"id"`
	Tool            string     `json:"tool"`
	Path            string     `json:"path"`
	ItemID          int64      `json:"item_id,omitempty"`
	BaseVersion     int        `json:"base_version,omitempty"`
	OriginalContent string     `json:"original_content,omitempty"`
	Content         string     `json:"content"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
}

// AIToolAudit 记录工具的名称、结果和时间，不保存资料正文或模型推理。
type AIToolAudit struct {
	Tool      string    `json:"tool"`
	Outcome   string    `json:"outcome"`
	CreatedAt time.Time `json:"created_at"`
}

// AITurnEvent 是浏览器可重放的进度事件。Answer 永远是服务端过滤后的用户可见答案快照。
type AITurnEvent struct {
	Sequence  int64            `json:"sequence"`
	Type      string           `json:"type"`
	Status    AITurnStatus     `json:"status,omitempty"`
	Answer    string           `json:"answer,omitempty"`
	Sources   []AIChatSource   `json:"sources,omitempty"`
	Audit     *AIToolAudit     `json:"audit,omitempty"`
	Approval  *AIWriteApproval `json:"approval,omitempty"`
	Error     string           `json:"error,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}

// AITurn 持久化 AI 任务的用户可见状态、答案、引用与审计，支持浏览器重连后继续订阅。
type AITurn struct {
	ID               string                `json:"id"`
	ConversationID   string                `json:"conversation_id"`
	Status           AITurnStatus          `json:"status"`
	Answer           string                `json:"answer,omitempty"`
	Model            string                `json:"model,omitempty"`
	Sources          []AIChatSource        `json:"sources,omitempty"`
	ContextSummary   string                `json:"context_summary,omitempty"`
	ContextMemory    *AIConversationMemory `json:"context_memory,omitempty"`
	HarnessSessionID string                `json:"harness_session_id,omitempty"`
	Error            string                `json:"error,omitempty"`
	Approval         *AIWriteApproval      `json:"approval,omitempty"`
	Audit            []AIToolAudit         `json:"audit,omitempty"`
	Events           []AITurnEvent         `json:"events,omitempty"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
}

type AIChatSource struct {
	SourceType string `json:"source_type"`
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Excerpt    string `json:"excerpt,omitempty"`
}

type AIChatResponse struct {
	Answer           string                `json:"answer"`
	Model            string                `json:"model"`
	Sources          []AIChatSource        `json:"sources"`
	Incomplete       bool                  `json:"incomplete"`
	HarnessSessionID string                `json:"harness_session_id,omitempty"`
	ContextSummary   string                `json:"context_summary,omitempty"`
	ContextMemory    *AIConversationMemory `json:"context_memory,omitempty"`
}

// HarnessStatus 报告所需 Agent 运行时是否就绪；AI 功能没有备用的直接提供商路径。
type HarnessStatus struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason,omitempty"`
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
