package model

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
}
