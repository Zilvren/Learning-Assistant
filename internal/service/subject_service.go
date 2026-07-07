package service

import "context"

func GetAllSubjects(ctx context.Context) ([]string, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return nil, err
	}
	return repos.Subjects.List(ctx)
}

func AddSubject(ctx context.Context, name string) ([]string, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return nil, err
	}
	return repos.Subjects.Create(ctx, name)
}

func SubjectExists(ctx context.Context, name string) bool {
	repos, err := repositories(ctx)
	if err != nil {
		return false
	}
	ok, err := repos.Subjects.Exists(ctx, name)
	return err == nil && ok
}

func DeleteSubject(ctx context.Context, name string) ([]string, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return nil, err
	}
	return repos.Subjects.Delete(ctx, name)
}
