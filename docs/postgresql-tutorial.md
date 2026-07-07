# PostgreSQL 使用教程

本文档面向 Study Tracker 项目，目标是让你自己完成 PostgreSQL 的基础使用、建库、建表、插入测试数据，并理解后续如何把 Go 后端接到 PostgreSQL。

本文不会直接替你改 Go 代码，而是一步一步教你自己操作。

## 1. 先理解 PostgreSQL 在项目里的作用

当前 Go 版本默认还是本地 JSON 存储，数据主要在 `data/` 目录里。

PostgreSQL 版本的目标是把这些数据迁移到数据库：

```text
data/errors.json       -> error_problems
data/subjects.json     -> subjects
data/config.json       -> user_settings
复习记录                -> review_records
OCR 任务                -> ocr_tasks
附件/图片/PDF           -> attachments
```

建好 PostgreSQL 表之后，程序不会自动使用数据库。后续还需要在 Go 项目里实现 PostgreSQL repository。

## 2. 基本概念

你可以这样理解 PostgreSQL：

```text
PostgreSQL 服务
  └── 数据库 database
        └── 数据表 table
              └── 行 row
              └── 列 column
```

对于本项目：

```text
数据库：study_tracker

主要数据表：
users
subjects
error_problems
tags
review_records
ocr_tasks
attachments
knowledge_items
```

## 3. 找到 psql 命令

你当前电脑上的 `psql.exe` 路径一般是：

```powershell
C:\Program Files\PostgreSQL\18\bin\psql.exe
```

因为路径里有空格，所以在 PowerShell 中执行时要写成：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres
```

这里的 `&` 是 PowerShell 的执行运算符，用来执行带引号的程序路径。

## 4. 连接 PostgreSQL

打开 PowerShell，输入：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres
```

然后输入你安装 PostgreSQL 时设置的 `postgres` 用户密码。

注意：输入密码时屏幕不会显示字符，这是正常现象。

如果成功，你会看到：

```text
postgres=#
```

退出：

```sql
\q
```

## 5. 创建项目数据库

创建数据库：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\createdb.exe" -U postgres study_tracker
```

如果提示输入密码，输入 `postgres` 用户密码。

连接到项目数据库：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d study_tracker
```

成功后会看到：

```text
study_tracker=#
```

## 6. 常用 psql 命令

进入 `psql` 后，这些命令很常用：

```sql
\l
```

查看所有数据库。

```sql
\c study_tracker
```

切换到 `study_tracker` 数据库。

```sql
\dt
```

查看当前数据库里的所有表。

```sql
\d users
```

查看 `users` 表结构。

```sql
\d error_problems
```

查看错题表结构。

```sql
\q
```

退出 `psql`。

## 7. 执行项目建表 SQL

本项目的 PostgreSQL 建表脚本在：

```text
C:\Users\Knock\Desktop\gotest\server-go\migrations\001_init_postgres.sql
```

执行建表：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d study_tracker -f "C:\Users\Knock\Desktop\gotest\server-go\migrations\001_init_postgres.sql"
```

执行完成后检查表：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d study_tracker -c "\dt"
```

你应该能看到类似这些表：

```text
users
user_settings
subjects
error_problems
tags
error_problem_tags
review_records
ocr_tasks
attachments
knowledge_items
refresh_tokens
```

## 8. 插入第一组测试数据

先插入一个用户：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d study_tracker -c "INSERT INTO users (username, email, password_hash) VALUES ('test', 'test@example.com', 'fake_hash') RETURNING id;"
```

假设返回的用户 `id` 是 `1`。

插入一个科目：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d study_tracker -c "INSERT INTO subjects (user_id, name, sort_order) VALUES (1, '数学', 1) RETURNING id;"
```

假设返回的科目 `id` 是 `1`。

插入一条错题：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d study_tracker -c "INSERT INTO error_problems (user_id, subject_id, title, question, wrong_answer, correct_answer, reason, difficulty, source, next_review_at) VALUES (1, 1, '函数单调性', '判断 f(x)=x^2 在 R 上是否单调', '单调递增', '不是单调函数', '忽略了 x<0 时递减', 3, 'manual', CURRENT_DATE);"
```

查询错题：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d study_tracker -c "SELECT id, title, difficulty, next_review_at FROM error_problems;"
```

## 9. 学会 JOIN 查询

错题表里只保存 `subject_id`，科目名称在 `subjects` 表里。

要同时查出错题和科目名称，需要 `JOIN`：

```sql
SELECT
  p.id,
  p.title,
  s.name AS subject_name,
  p.difficulty,
  p.next_review_at
FROM error_problems p
LEFT JOIN subjects s ON s.id = p.subject_id
WHERE p.user_id = 1
  AND p.deleted_at IS NULL
ORDER BY p.created_at DESC;
```

用 PowerShell 执行：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d study_tracker -c "SELECT p.id, p.title, s.name AS subject_name, p.difficulty, p.next_review_at FROM error_problems p LEFT JOIN subjects s ON s.id = p.subject_id WHERE p.user_id = 1 AND p.deleted_at IS NULL ORDER BY p.created_at DESC;"
```

## 10. 创建项目专用数据库用户

开发时可以先用 `postgres` 用户，但项目长期运行不建议用超级用户。

先进入数据库：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d study_tracker
```

然后执行：

```sql
CREATE USER study_tracker_app WITH PASSWORD 'your_password_here';

GRANT CONNECT ON DATABASE study_tracker TO study_tracker_app;
GRANT USAGE ON SCHEMA public TO study_tracker_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO study_tracker_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO study_tracker_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO study_tracker_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT USAGE, SELECT ON SEQUENCES TO study_tracker_app;
```

以后 Go 项目可以使用这个用户连接数据库。

## 11. Go 项目的连接字符串

后续 Go 后端接 PostgreSQL 时，可以使用类似这样的连接字符串：

```text
postgres://study_tracker_app:your_password_here@localhost:5432/study_tracker?sslmode=disable
```

或者拆成环境变量：

```text
DB_HOST=localhost
DB_PORT=5432
DB_USER=study_tracker_app
DB_PASSWORD=your_password_here
DB_NAME=study_tracker
DB_SSLMODE=disable
```

本地开发一般使用：

```text
sslmode=disable
```

服务器部署时再考虑 SSL。

## 12. 后续 Go 项目接入步骤

建好数据库之后，Go 项目要按这个顺序改：

```text
1. 新增数据库配置
2. 新增 PostgreSQL 连接池
3. 定义 repository 接口
4. 保留 JSON repository
5. 新增 PostgreSQL repository
6. service 层只依赖接口
7. 通过配置切换 json/postgres
8. 写 JSON 数据导入 PostgreSQL 的迁移工具
```

推荐先迁移三块核心业务：

```text
subjects
error_problems
review_records
```

原因是它们最能体现项目价值：

```text
科目管理
错题管理
间隔复习
学习统计
```

OCR、附件、用户登录可以放到第二阶段。

## 13. 常见问题

### 数据库不存在

错误：

```text
database "study_tracker" does not exist
```

原因：还没有创建数据库。

解决：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\createdb.exe" -U postgres study_tracker
```

### 输入密码后像是没有反应

输入密码时不会显示字符，继续输入完整密码后按回车即可。

### 双击窗口闪退

不要双击 `psql.exe`。应该先打开 PowerShell，再执行命令。

如果写 bat 文件，末尾加：

```bat
pause
```

### psql 不是内部或外部命令

说明 PostgreSQL 的 `bin` 目录没有加入 PATH。

可以使用完整路径：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres
```

也可以把这个目录加入系统 PATH：

```text
C:\Program Files\PostgreSQL\18\bin
```

### 建表脚本执行失败

先确认你连接的是正确数据库：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d study_tracker -c "SELECT current_database();"
```

再执行建表脚本。

如果表已经存在，说明你之前执行过迁移脚本。初学阶段可以新建另一个测试数据库练习，不要直接删除正在使用的数据。

## 14. 你现在应该掌握的最小知识集

先把这些学会就够你继续做项目：

```text
CREATE DATABASE
CREATE TABLE
PRIMARY KEY
FOREIGN KEY
INSERT
SELECT
UPDATE
DELETE
JOIN
INDEX
JSONB
EXPLAIN
```

对这个项目来说，最重要的是：

```text
users 一对多 error_problems
subjects 一对多 error_problems
error_problems 一对多 review_records
error_problems 多对多 tags
```

把这几组关系讲清楚，毕设答辩和 Go 后端实习面试都会更有底气。

