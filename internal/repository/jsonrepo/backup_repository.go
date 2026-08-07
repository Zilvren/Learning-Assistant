package jsonrepo

import (
	"context"

	models "study-tracker-go/internal/model"
	base "study-tracker-go/internal/repository"
)

type BackupRepository struct {
	store *base.JSONStore
}

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
		result = base.BackupData{
			Subjects:  &subjects,
			Errors:    &errors,
			Config:    &config,
			Knowledge: &knowledge,
			Library:   &library,
		}
		return nil
	})
	return result, err
}

func (r *BackupRepository) Import(ctx context.Context, data base.BackupData) error {
	return r.store.Write(ctx, func(tx *base.JSONTx) error {
		if data.Subjects != nil {
			subjects := *data.Subjects
			if subjects == nil {
				subjects = []string{}
			}
			if err := tx.Save("subjects.json", subjects); err != nil {
				return err
			}
		}
		if data.Errors != nil {
			errors := *data.Errors
			if errors == nil {
				errors = []models.ErrorProblem{}
			}
			if err := tx.Save("errors.json", errors); err != nil {
				return err
			}
		}
		if data.Config != nil {
			if err := tx.Save("config.json", *data.Config); err != nil {
				return err
			}
		}
		if data.Knowledge != nil {
			knowledge := *data.Knowledge
			if knowledge == nil {
				knowledge = map[string][]string{}
			}
			if err := tx.Save("knowledge.json", knowledge); err != nil {
				return err
			}
		}
		return nil
	})
}

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
