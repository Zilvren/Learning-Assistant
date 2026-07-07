package jsonrepo

import (
	"context"

	base "study-tracker-go/internal/repository"
)

type BackupRepository struct {
	subjects  *SubjectRepository
	errors    *ErrorRepository
	settings  *SettingsRepository
	knowledge *KnowledgeRepository
}

func (r *BackupRepository) Export(ctx context.Context) (base.BackupData, error) {
	subjects, err := r.subjects.List(ctx)
	if err != nil {
		return base.BackupData{}, err
	}
	errors, err := r.errors.List(ctx, base.ErrorFilter{})
	if err != nil {
		return base.BackupData{}, err
	}
	config, err := r.settings.Load(ctx)
	if err != nil {
		return base.BackupData{}, err
	}
	knowledge, err := r.knowledge.Load(ctx)
	if err != nil {
		return base.BackupData{}, err
	}
	return base.BackupData{
		Subjects:  &subjects,
		Errors:    &errors,
		Config:    &config,
		Knowledge: &knowledge,
	}, nil
}

func (r *BackupRepository) Import(ctx context.Context, data base.BackupData) error {
	if data.Subjects != nil {
		if err := r.subjects.Replace(ctx, *data.Subjects); err != nil {
			return err
		}
	}
	if data.Errors != nil {
		if err := r.errors.Replace(ctx, *data.Errors); err != nil {
			return err
		}
	}
	if data.Config != nil {
		if err := r.settings.Save(ctx, *data.Config); err != nil {
			return err
		}
	}
	if data.Knowledge != nil {
		if err := r.knowledge.Replace(ctx, *data.Knowledge); err != nil {
			return err
		}
	}
	return nil
}

func (r *BackupRepository) HasData(ctx context.Context) (bool, error) {
	subjects, err := r.subjects.List(ctx)
	if err != nil {
		return false, err
	}
	if len(subjects) > 0 {
		return true, nil
	}
	errors, err := r.errors.HasAny(ctx)
	if err != nil {
		return false, err
	}
	return errors, nil
}
