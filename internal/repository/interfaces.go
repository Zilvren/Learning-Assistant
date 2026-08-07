package repository

import (
	"context"
	"encoding/json"
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
	Backup    BackupRepository
	Library   LibraryRepository
}

type LibraryFilter struct {
	ParentID   *int64
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
	Duplicate(ctx context.Context, id int64, parentID *int64) (models.LibraryItem, error)
	Versions(ctx context.Context, id int64) ([]models.LibraryVersion, error)
	RestoreVersion(ctx context.Context, id, versionID int64) (models.LibraryItem, error)
	ListTags(ctx context.Context) ([]string, error)
	DueReviews(ctx context.Context, day time.Time) ([]models.LibraryItem, error)
	Review(ctx context.Context, id int64, reviewedAt time.Time, intervals []int) (models.LibraryItem, error)
	EnsureLegacy(ctx context.Context, errors []models.ErrorProblem, subjects []string) error
	Cleanup(ctx context.Context, before time.Time) error
}

type AuthRepository interface {
	CreateUser(ctx context.Context, username string, email string, passwordHash string) (models.User, error)
	FindUserByAccount(ctx context.Context, account string) (models.AuthUser, error)
	FindUserByID(ctx context.Context, id int64) (models.AuthUser, error)
	TouchLastLogin(ctx context.Context, id int64) error
	CreateRefreshToken(ctx context.Context, userID int64, tokenHash string, userAgent string, ipAddress string, expiresAt time.Time) error
	FindRefreshToken(ctx context.Context, tokenHash string) (userID int64, expiresAt time.Time, revoked bool, err error)
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
	Provider       string
	Status         string
	SourceFilename string
	MimeType       string
	FileSize       int64
	BatchID        string
	TaskID         string
	ResultMarkdown string
	ErrorMessage   string
	Metadata       map[string]interface{}
	FinishedAt     *time.Time
}

type OCRTaskRepository interface {
	Create(ctx context.Context, task OCRTask) (int64, error)
	Update(ctx context.Context, id int64, task OCRTask) error
}

type BackupData struct {
	Errors      *[]models.ErrorProblem
	Subjects    *[]string
	Config      *models.Config
	Knowledge   *map[string][]string
	LibraryJSON json.RawMessage
	Blobs       map[string][]byte
}

type BackupRepository interface {
	Export(ctx context.Context) (BackupData, error)
	Import(ctx context.Context, data BackupData) error
	HasData(ctx context.Context) (bool, error)
}
