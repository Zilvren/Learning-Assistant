package jsonrepo

import base "study-tracker-go/internal/repository"

type Repositories struct {
	store     *base.JSONStore
	subjects  *SubjectRepository
	errors    *ErrorRepository
	settings  *SettingsRepository
	knowledge *KnowledgeRepository
	ocrTasks  *OCRTaskRepository
	backup    *BackupRepository
	auth      *AuthRepository
	library   *LibraryRepository
}

func NewRepositories() base.Repositories {
	store := base.DefaultJSONStore()
	repos := &Repositories{store: store}
	repos.subjects = &SubjectRepository{store: store}
	repos.errors = &ErrorRepository{store: store}
	repos.settings = &SettingsRepository{store: store}
	repos.knowledge = &KnowledgeRepository{store: store}
	repos.ocrTasks = &OCRTaskRepository{}
	repos.auth = &AuthRepository{}
	repos.library = &LibraryRepository{store: store}
	repos.backup = &BackupRepository{
		store: store,
	}
	return base.Repositories{
		Auth:      repos.auth,
		Subjects:  repos.subjects,
		Errors:    repos.errors,
		Settings:  repos.settings,
		Knowledge: repos.knowledge,
		OCRTasks:  repos.ocrTasks,
		Backup:    repos.backup,
		Library:   repos.library,
	}
}
