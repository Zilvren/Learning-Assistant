package repository

import (
	"context"
	"time"

	models "study-tracker-go/internal/model"
)

type Repositories struct {
	Auth      AuthRepository
	Subjects  SubjectRepository
	Errors    ErrorRepository
	Settings  SettingsRepository
	Knowledge KnowledgeRepository
	OCRTasks  OCRTaskRepository
	Activity  ActivityRepository
	Relations LearningRelationRepository
	Backup    BackupRepository
	Library   LibraryRepository
}

type LibraryFilter struct {
	ParentID   *int64
	All        bool
	Kind       string
	Query      string
	Trashed    bool
	Tag        string
	ReviewOnly bool
	DueOnly    bool
}

type LibraryRepository interface {
	List(ctx context.Context, filter LibraryFilter) ([]models.LibraryItem, error)
	Get(ctx context.Context, id int64) (models.LibraryItem, error)
	Create(ctx context.Context, req models.CreateLibraryItemRequest, content []byte) (models.LibraryItem, error)
	Update(ctx context.Context, id int64, req models.UpdateLibraryItemRequest) (models.LibraryItem, error)
	SaveContent(ctx context.Context, id int64, content []byte, baseVersion int, checkpoint, force bool) (models.LibraryItem, error)
	ReadContent(ctx context.Context, id int64) ([]byte, models.LibraryItem, error)
	Trash(ctx context.Context, id int64) error
	Restore(ctx context.Context, id int64) (models.LibraryItem, error)
	Purge(ctx context.Context, id int64) error
	Batch(ctx context.Context, action string, ids []int64, parentID *int64) error
	Duplicate(ctx context.Context, id int64, parentID *int64) (models.LibraryItem, error)
	Versions(ctx context.Context, id int64) ([]models.LibraryVersion, error)
	RestoreVersion(ctx context.Context, id, versionID int64) (models.LibraryItem, error)
	ListTags(ctx context.Context) ([]string, error)
	DueReviews(ctx context.Context, day time.Time) ([]models.LibraryItem, error)
	Review(ctx context.Context, id int64, reviewedAt time.Time, intervals []int) (models.LibraryItem, error)
	ReviewWithRating(ctx context.Context, id int64, reviewedAt time.Time, rating string) (models.LibraryItem, error)
	EnsureLegacy(ctx context.Context, errors []models.ErrorProblem, subjects []string) error
	Cleanup(ctx context.Context, before time.Time) error
}

type AuthRepository interface {
	CreateUser(ctx context.Context, username string, email string, passwordHash string, emailVerified bool) (models.User, error)
	DeleteUnverifiedUser(ctx context.Context, id int64) error
	FindUserByAccount(ctx context.Context, account string) (models.AuthUser, error)
	FindUserByID(ctx context.Context, id int64) (models.AuthUser, error)
	CreateEmailVerificationToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	ConsumeEmailVerificationToken(ctx context.Context, tokenHash string) (models.AuthUser, error)
	TouchLastLogin(ctx context.Context, id int64) error
	CreateRefreshToken(ctx context.Context, userID int64, tokenHash string, userAgent string, ipAddress string, expiresAt time.Time) error
	FindRefreshToken(ctx context.Context, tokenHash string) (userID int64, expiresAt time.Time, revoked bool, err error)
	ConsumeRefreshToken(ctx context.Context, tokenHash string) (userID int64, expiresAt time.Time, err error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
}

type SubjectRepository interface {
	List(ctx context.Context) ([]string, error)
	Exists(ctx context.Context, name string) (bool, error)
	Create(ctx context.Context, name string) ([]string, error)
	Delete(ctx context.Context, name string) ([]string, error)
	Replace(ctx context.Context, subjects []string) error
}

type ErrorFilter struct {
	Subject   string
	Keyword   string
	Tag       string
	ReasonTag string
}

type ErrorRepository interface {
	Create(ctx context.Context, item models.ErrorProblem) (models.ErrorProblem, error)
	List(ctx context.Context, filter ErrorFilter) ([]models.ErrorProblem, error)
	Get(ctx context.Context, id int) (models.ErrorProblem, error)
	Update(ctx context.Context, id int, req models.UpdateErrorRequest) error
	Delete(ctx context.Context, id int) error
	Review(ctx context.Context, id int, reviewedAt time.Time, intervals []int) (models.ErrorProblem, error)
	ReviewWithRating(ctx context.Context, id int, reviewedAt time.Time, rating string) (models.ErrorProblem, error)
	ListTags(ctx context.Context) ([]string, error)
	Replace(ctx context.Context, errors []models.ErrorProblem) error
	HasAny(ctx context.Context) (bool, error)
}

type SettingsRepository interface {
	Load(ctx context.Context) (models.Config, error)
	Save(ctx context.Context, config models.Config) error
}

type KnowledgeRepository interface {
	Load(ctx context.Context) (map[string][]string, error)
	Replace(ctx context.Context, knowledge map[string][]string) error
}

type OCRTask struct {
	ID             int64
	Provider       string
	Status         string
	SourceFilename string
	MimeType       string
	FileSize       int64
	BatchID        string
	TaskID         string
	ResultMarkdown string
	InputBlobHash  string
	ErrorMessage   string
	Metadata       map[string]interface{}
	FinishedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type OCRTaskRepository interface {
	Create(ctx context.Context, task OCRTask) (int64, error)
	Update(ctx context.Context, id int64, task OCRTask) error
	Get(ctx context.Context, id int64) (OCRTask, error)
	List(ctx context.Context, limit int) ([]OCRTask, error)
}

type ActivityRepository interface {
	Record(ctx context.Context, event models.ActivityEvent) error
	List(ctx context.Context, startDate, endDate time.Time) ([]models.ActivityEvent, error)
}

type LearningRelationRepository interface {
	List(ctx context.Context, sourceType string, sourceID int64) ([]models.LearningRelation, error)
	Create(ctx context.Context, relation models.LearningRelation) (models.LearningRelation, error)
	Delete(ctx context.Context, id int64) error
}

type BackupData struct {
	Errors    *[]models.ErrorProblem
	Subjects  *[]string
	Config    *models.Config
	Knowledge *map[string][]string
	Library   *LibraryBackup
	Activity  *ActivityBackup
	Relations *RelationBackup
	Blobs     map[string][]byte
}

// LibraryBackup is the portable representation stored in library.json inside
// a backup archive. Item and version IDs describe relationships within the
// archive only; repositories allocate new database IDs during import.
type LibraryBackup struct {
	SchemaVersion int                     `json:"schema_version"`
	NextID        int64                   `json:"next_id"`
	NextVersionID int64                   `json:"next_version_id"`
	Items         []models.LibraryItem    `json:"items"`
	Versions      []models.LibraryVersion `json:"versions"`
}

// ActivityBackup and RelationBackup keep supplemental learning data portable
// while preserving the JSON store's next-id state.
type ActivityBackup struct {
	NextID int64                  `json:"next_id"`
	Events []models.ActivityEvent `json:"events"`
}

type RelationBackup struct {
	NextID    int64                     `json:"next_id"`
	Relations []models.LearningRelation `json:"relations"`
}

type BackupRepository interface {
	Export(ctx context.Context) (BackupData, error)
	Import(ctx context.Context, data BackupData) error
	HasData(ctx context.Context) (bool, error)
}
