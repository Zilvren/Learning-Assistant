package service

import "context"

// GetAllSubjects 在业务层中读取并整理所需数据。
func GetAllSubjects(ctx context.Context) ([]string, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return nil, err
	}
	return repos.Subjects.List(ctx)
}

// AddSubject 在业务层中创建或更新相应状态。
func AddSubject(ctx context.Context, name string) ([]string, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return nil, err
	}
	return repos.Subjects.Create(ctx, name)
}

// SubjectExists 在业务层中完成本文件定义的局部处理。
func SubjectExists(ctx context.Context, name string) bool {
	repos, err := repositories(ctx)
	if err != nil {
		return false
	}
	ok, err := repos.Subjects.Exists(ctx, name)
	return err == nil && ok
}

// DeleteSubject 在业务层中删除、清理或撤销相应状态。
func DeleteSubject(ctx context.Context, name string) ([]string, error) {
	repos, err := repositories(ctx)
	if err != nil {
		return nil, err
	}
	return repos.Subjects.Delete(ctx, name)
}
