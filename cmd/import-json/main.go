package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	base "study-tracker-go/internal/repository"
	jsonrepo "study-tracker-go/internal/repository/jsonrepo"
	postgresrepo "study-tracker-go/internal/repository/postgres"
)

func main() {
	var dataDir string
	var databaseURL string
	var username string
	var dryRun bool
	var replace bool

	flag.StringVar(&dataDir, "data-dir", "data", "JSON 数据目录")
	flag.StringVar(&databaseURL, "database-url", os.Getenv("TRACKER_DATABASE_URL"), "PostgreSQL 连接字符串")
	flag.StringVar(&username, "username", "local", "导入到指定 PostgreSQL 用户")
	flag.BoolVar(&dryRun, "dry-run", false, "只检查并打印导入数量，不写入数据库")
	flag.BoolVar(&replace, "replace", false, "替换目标用户已有数据")
	flag.Parse()

	if err := run(dataDir, databaseURL, username, dryRun, replace); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(dataDir string, databaseURL string, username string, dryRun bool, replace bool) error {
	ctx := context.Background()
	base.SetDataDir(dataDir)

	source := jsonrepo.NewRepositories()
	data, err := source.Backup.Export(ctx)
	if err != nil {
		return err
	}

	subjectCount := 0
	errorCount := 0
	knowledgeCount := 0
	if data.Subjects != nil {
		subjectCount = len(*data.Subjects)
	}
	if data.Errors != nil {
		errorCount = len(*data.Errors)
	}
	if data.Knowledge != nil {
		for _, items := range *data.Knowledge {
			knowledgeCount += len(items)
		}
	}

	fmt.Printf("准备导入：subjects=%d errors=%d knowledge_items=%d\n", subjectCount, errorCount, knowledgeCount)
	if dryRun {
		fmt.Println("dry-run 已完成，未写入数据库")
		return nil
	}

	pool, err := postgresrepo.NewPool(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	userID, err := postgresrepo.EnsureUser(ctx, pool, username)
	if err != nil {
		return err
	}
	target := postgresrepo.NewRepositories(pool, userID)

	hasData, err := target.Backup.HasData(ctx)
	if err != nil {
		return err
	}
	if hasData && !replace {
		return fmt.Errorf("目标 PostgreSQL 用户已有数据；如需覆盖请添加 --replace")
	}
	if err := target.Backup.Import(ctx, data); err != nil {
		return err
	}
	fmt.Println("导入完成")
	return nil
}
