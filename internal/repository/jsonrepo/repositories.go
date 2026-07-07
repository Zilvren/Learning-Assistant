package jsonrepo

import base "study-tracker-go/internal/repository"

type Repositories struct {
	subjects  *SubjectRepository
	errors    *ErrorRepository
	settings  *SettingsRepository
	knowledge *KnowledgeRepository
	ocrTasks  *OCRTaskRepository
	backup    *BackupRepository
	auth      *AuthRepository
}

func NewRepositories() base.Repositories {
	repos := &Repositories{}
	repos.subjects = &SubjectRepository{}
	repos.errors = &ErrorRepository{}
	repos.settings = &SettingsRepository{}
	repos.knowledge = &KnowledgeRepository{}
	repos.ocrTasks = &OCRTaskRepository{}
	repos.auth = &AuthRepository{}
	repos.backup = &BackupRepository{
		subjects:  repos.subjects,
		errors:    repos.errors,
		settings:  repos.settings,
		knowledge: repos.knowledge,
	}
	return base.Repositories{
		Auth:      repos.auth,
		Subjects:  repos.subjects,
		Errors:    repos.errors,
		Settings:  repos.settings,
		Knowledge: repos.knowledge,
		OCRTasks:  repos.ocrTasks,
		Backup:    repos.backup,
	}
}
