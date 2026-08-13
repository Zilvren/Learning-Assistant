package jsonrepo

import (
	"context"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

type BackupRepository struct {
	store *base.JSONStore
}

// Export 在存储层中完成本文件定义的局部处理。
func (r *BackupRepository) Export(ctx context.Context) (base.BackupData, error) {
	var result base.BackupData
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		subjects := []string{}
		errors := []models.ErrorProblem{}
		var config models.Config
		knowledge := map[string][]string{}
		library := base.LibraryBackup{
			SchemaVersion: librarySchemaVersion,
			NextID:        1,
			NextVersionID: 1,
			Items:         []models.LibraryItem{},
			Versions:      []models.LibraryVersion{},
		}
		activity := base.ActivityBackup{NextID: 1, Events: []models.ActivityEvent{}}
		relations := base.RelationBackup{NextID: 1, Relations: []models.LearningRelation{}}
		if err := tx.Load("subjects.json", &subjects); err != nil {
			return err
		}
		if err := tx.Load("errors.json", &errors); err != nil {
			return err
		}
		if err := tx.Load("config.json", &config); err != nil {
			return err
		}
		if err := tx.Load("knowledge.json", &knowledge); err != nil {
			return err
		}
		if err := tx.Load("library.json", &library); err != nil {
			return err
		}
		if err := tx.Load("activity.json", &activity); err != nil {
			return err
		}
		if err := tx.Load("relations.json", &relations); err != nil {
			return err
		}
		result = base.BackupData{
			Subjects:  &subjects,
			Errors:    &errors,
			Config:    &config,
			Knowledge: &knowledge,
			Library:   &library,
			Activity:  &activity,
			Relations: &relations,
		}
		return nil
	})
	return result, err
}

// Import 在存储层中完成本文件定义的局部处理。
func (r *BackupRepository) Import(ctx context.Context, data base.BackupData) error {
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		values := map[string]interface{}{}
		if data.Subjects != nil {
			subjects := *data.Subjects
			if subjects == nil {
				subjects = []string{}
			}
			values["subjects.json"] = subjects
		}
		if data.Errors != nil {
			errors := *data.Errors
			if errors == nil {
				errors = []models.ErrorProblem{}
			}
			values["errors.json"] = errors
		}
		if data.Config != nil {
			config := *data.Config
			if config.MineruToken == "" || config.DeepSeekToken == "" {
				var current models.Config
				if err := tx.Load("config.json", &current); err != nil {
					return err
				}
				if config.MineruToken == "" {
					config.MineruToken = current.MineruToken
				}
				if config.DeepSeekToken == "" {
					config.DeepSeekToken = current.DeepSeekToken
				}
			}
			sealedToken, err := base.SealSecret(config.MineruToken)
			if err != nil {
				return err
			}
			sealedDeepSeekToken, err := base.SealSecret(config.DeepSeekToken)
			if err != nil {
				return err
			}
			config.MineruToken = sealedToken
			config.DeepSeekToken = sealedDeepSeekToken
			values["config.json"] = config
		}
		if data.Knowledge != nil {
			knowledge := *data.Knowledge
			if knowledge == nil {
				knowledge = map[string][]string{}
			}
			values["knowledge.json"] = knowledge
		}
		if data.Library != nil {
			library := *data.Library
			if library.Items == nil {
				library.Items = []models.LibraryItem{}
			}
			if library.Versions == nil {
				library.Versions = []models.LibraryVersion{}
			}
			values["library.json"] = library
		}
		if data.Activity != nil {
			activity := *data.Activity
			if activity.NextID < 1 {
				activity.NextID = 1
			}
			if activity.Events == nil {
				activity.Events = []models.ActivityEvent{}
			}
			values["activity.json"] = activity
		}
		if data.Relations != nil {
			relations := *data.Relations
			if relations.NextID < 1 {
				relations.NextID = 1
			}
			if relations.Relations == nil {
				relations.Relations = []models.LearningRelation{}
			}
			values["relations.json"] = relations
		}
		return tx.SaveAll(values)
	})
}

// HasData 在存储层中校验输入或判断当前条件。
func (r *BackupRepository) HasData(ctx context.Context) (bool, error) {
	hasData := false
	err := r.store.Read(ctx, func(tx *base.JSONTx) error {
		subjects := []string{}
		if err := tx.Load("subjects.json", &subjects); err != nil {
			return err
		}
		if len(subjects) > 0 {
			hasData = true
			return nil
		}
		errors := []models.ErrorProblem{}
		if err := tx.Load("errors.json", &errors); err != nil {
			return err
		}
		hasData = len(errors) > 0
		return nil
	})
	return hasData, err
}
