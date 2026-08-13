package jsonrepo

import base "study-tracker-go/internal/repository"

type Repositories struct {
	store     *base.JSONStore
	subjects  *SubjectRepository
	errors    *ErrorRepository
	settings  *SettingsRepository
	knowledge *KnowledgeRepository
	ocrTasks  *OCRTaskRepository
	activity  *ActivityRepository
	relations *LearningRelationRepository
	backup    *BackupRepository
	auth      *AuthRepository
	library   *LibraryRepository
}

// NewRepositories 在存储层中创建所需对象并完成初始化。
func NewRepositories() base.Repositories {
	store := base.DefaultJSONStore()
	repos := &Repositories{store: store}
	repos.subjects = &SubjectRepository{store: store}
	repos.errors = &ErrorRepository{store: store}
	repos.settings = &SettingsRepository{store: store}
	repos.knowledge = &KnowledgeRepository{store: store}
	repos.ocrTasks = &OCRTaskRepository{store: store}
	repos.activity = &ActivityRepository{store: store}
	repos.relations = &LearningRelationRepository{store: store}
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
		Activity:  repos.activity,
		Relations: repos.relations,
		Backup:    repos.backup,
		Library:   repos.library,
	}
}
