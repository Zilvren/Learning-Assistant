# Go 后端重写教程 —— 把 FastAPI 版错题追踪器迁移到 Go

> **这份教程怎么用？** 每一步只做一件事，做完立刻验证。如果某一步验证失败，先回到上一步排查，不要继续往下。

> **这次讲解补强的阅读方式：** 代码块告诉你"该写什么"，代码块后面的说明重点解释"为什么这样写"。遇到默认值、`nil`、指针、日期字符串、路由返回格式这些看起来啰嗦的地方，不要只当语法看，它们通常是在解决三个真实问题：兼容旧数据、避免前端拿到奇怪的 `null`、让程序在文件缺失或字段缺失时还能继续跑。

这份文档按后端迁移的真实顺序组织：

```text
先让服务器和前端能打开
→ 再定义数据长什么样
→ 再做 JSON 存储
→ 再从最简单的科目功能开始
→ 最后补齐错题、复习、设置、备份、OCR、更新等完整接口
```

读每个 Part 时，都可以用同一个检查框架：

| 问题 | 你要看懂什么 |
|------|--------------|
| 这一层负责什么 | handler 管 HTTP，service 管业务，store 管文件，models 管数据结构 |
| 为什么要这样写 | 通常是为了兼容前端、兼容旧 JSON、避免空值崩溃、或者方便后续扩展 |
| 失败时会怎样 | 看 `error` 怎么返回，HTTP 状态码怎么选，默认值怎么兜底 |
| 前端依赖什么 | JSON 字段名、数组是否为 `[]`、是否包一层对象、接口路径是否一致 |

## 目录

- [Part 0：准备工作](#part-0准备工作)
- [Part 1：第一个 Web 服务器 + 前端页面](#part-1第一个-web-服务器--前端页面)
- [Part 2：数据模型（models）](#part-2数据模型models)
- [Part 3：JSON 文件读写（store）](#part-3json-文件读写store)
- [Part 4：科目管理（第一个完整功能）](#part-4科目管理第一个完整功能)
- [Part 5：错题管理——创建和查询](#part-5错题管理创建和查询)
- [Part 6：错题管理——更新和删除](#part-6错题管理更新和删除)
- [Part 7：复习功能 + 标签](#part-7复习功能--标签)
- [Part 8：每日推送](#part-8每日推送)
- [Part 9：设置接口](#part-9设置接口)
- [Part 10：备份导出导入](#part-10备份导出导入)
- [Part 11：OCR 识别](#part-11ocr-识别)
- [Part 12：版本和更新接口](#part-12版本和更新接口)
- [Part 13：测试页面 + 最终验证](#part-13测试页面--最终验证)

---

## Part 0：准备工作

### 目标

确认开发环境正确，创建项目骨架。

**为什么准备工作也要写得这么细？**

后面的错误大多不是 Go 语法本身造成的，而是路径、模块名、依赖、数据目录不一致造成的。比如 `import "study-tracker-go/service"` 能不能编译，取决于 `go.mod` 第一行是不是 `module study-tracker-go`；`store.LoadJSON("errors.json")` 能不能读到数据，取决于运行目录下有没有 `data/errors.json`。所以 Part 0 的每一步都在固定一个"后面默认成立的前提"。

| 准备项 | 后面依赖它的地方 | 如果没做好会怎样 |
|--------|------------------|------------------|
| Go 版本 | `embed`、Gin、模块编译 | 旧版本可能不支持嵌入文件或依赖无法安装 |
| 项目目录 | 所有相对路径 | `go run .` 找不到 `go.mod` 或数据目录 |
| `go.mod` 模块名 | 所有本地 `import` | 编译时报 `package ... is not in std` |
| `data/` 文件 | 科目、错题、配置读取 | 页面打开但列表为空或接口报错 |
| `service/handlers` 目录 | 三层架构 | 代码都塞进 `main.go`，后面很快失控 |

### Step 0.1：确认 Go 版本

打开 PowerShell，运行：

```powershell
go version
```

应该看到类似 `go version go1.xx.x windows/amd64`。版本号 1.21+ 就行。

### Step 0.2：确认项目目录

```powershell
cd C:\Users\Knock\Desktop\gotest\server-go
dir
```

你应该能看到 `go.mod`、`models/models.go`、`store/json_store.go` 已经存在。这些是已经写好的基础代码。

看一下 go.mod 确认模块名：

```powershell
type go.mod
```

第一行应该是 `module study-tracker-go`。**记住这个模块名**，后面所有 import 都以它开头。

### Step 0.3：安装依赖

```powershell
go mod tidy
```

这个命令会下载 `go.mod` 里声明的依赖（主要是 Gin 框架）。如果网络不通，先设置代理：

```powershell
$env:GOPROXY = "https://goproxy.cn,direct"
go mod tidy
```

### Step 0.4：创建目录结构

后面会用到 `service/` 和 `handlers/` 两个新目录：

```powershell
mkdir service, handlers
```

### Step 0.5：复制数据文件

原 FastAPI 项目的数据文件在 `C:\Users\Knock\Desktop\11408\study_tracker\data\`，复制过来后面直接用：

```powershell
Copy-Item -Recurse -Force C:\Users\Knock\Desktop\11408\study_tracker\data\* C:\Users\Knock\Desktop\gotest\server-go\data\
```

现在 `data/` 目录下应该有 `errors.json`、`subjects.json`、`config.json` 等文件。

### ✅ 验证

```powershell
dir data
```

应该看到至少 `errors.json` 和 `subjects.json`。

---

## Part 1：第一个 Web 服务器 + 前端页面

### 目标

写一个能跑起来的 Web 服务器，并且**立刻就能在浏览器里看到 Vue 前端页面**。

> **核心思路：** 先把前端页面挂上去（哪怕 API 都没写好，页面至少能打开），之后每写完一个 API，刷新浏览器就能看到对应功能生效。不用等到最后才知道前端能不能用。

**这一 Part 的核心设计：先打通"浏览器到 Go"这条链。**

这里暂时不追求 API 完整，而是先证明三件事：

1. Go 程序能监听端口。
2. 浏览器能访问 Go 程序。
3. Go 程序能把 Vue 的 `index.html`、JS、CSS 正确返回给浏览器。

这一步提前做的好处很大：后面写每个接口时，你不需要猜"是前端没加载，还是 API 写错了"，因为前端加载这件事已经单独验证过了。迁移项目最怕多个问题叠在一起排查，Part 1 就是在把问题拆开。

### Step 1.1：复制前端文件

原 Vue 项目已经 build 好了，直接复制 dist 到 Go 项目：

```powershell
Copy-Item -Recurse -Force C:\Users\Knock\Desktop\11408\study_tracker\frontend\dist C:\Users\Knock\Desktop\gotest\server-go\frontend\dist
```

确认复制成功：

```powershell
dir frontend\dist
```

应该看到 `index.html` 和 `assets/` 目录。

### Step 1.2：创建 embed.go

Go 1.16+ 可以把文件嵌入到编译后的 exe 里。创建 `embed.go`：

```go
package main

import "embed"

//go:embed frontend/dist
var frontendFS embed.FS
```

> ⚠️ `//go:embed` 和 `var frontendFS` 之间**不能有空行**！否则嵌入不会生效。

**这段代码在干什么？**

`embed.go` 的作用是**把整个前端文件夹打包进 exe 里**。

想象一下：你的 Vue 前端有几十个文件（`index.html`、`app.js`、`style.css`、图片…），如果没有 `embed.go`，你的 exe 跑到哪里都必须带着 `frontend/dist/` 这个文件夹，丢了就 404。

有了这三行代码：

| 行 | 干什么 |
|---|--------|
| `package main` | 和 `main.go` 同一个 package，所以 `frontendFS` 变量在 `main.go` 里直接用 |
| `import "embed"` | Go 内置的嵌入库，不需要额外安装 |
| `//go:embed frontend/dist` | **编译器指令**，告诉 Go 编译器：编译时把 `frontend/dist` 目录下所有文件读进来 |
| `var frontendFS embed.FS` | `frontendFS` 是一个"虚拟文件系统"，编译后这些文件就在内存里 |

对比一下有无 `embed.go` 的区别：

```text
没有 embed.go：
  server-go.exe 在电脑 A → 找不到 frontend/dist/ → 首页 404

有 embed.go：
  server-go.exe 在电脑 A → 文件在 exe 体内 → 正常显示
```

`main.go` 里怎么用它？通过 `fs.ReadFile(frontendFS, "frontend/dist/index.html")` 就能从 exe 体内读文件，语法和读硬盘文件几乎一样。对浏览器来说完全透明——它不知道文件来自硬盘还是 exe 体内。

### Step 1.3：新建 main.go

在 `server-go/` 目录下创建 `main.go`，包含三个部分：健康检查 + 前端 SPA 回退 + 启动服务器。

```go
package main

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// 健康检查
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 非 /api 路径 → 前端 SPA
	r.NoRoute(serveFrontend)

	r.Run("127.0.0.1:8000")
}

// serveFrontend 处理前端页面请求
func serveFrontend(c *gin.Context) {
	// /api 开头但没匹配到路由 → 404
	if strings.HasPrefix(c.Request.URL.Path, "/api") {
		c.JSON(http.StatusNotFound, gin.H{"detail": "接口不存在"})
		return
	}

	// 请求路径 → 文件路径
	requestPath := strings.TrimPrefix(c.Request.URL.Path, "/")
	if requestPath == "" {
		requestPath = "index.html"
	}

	filePath := path.Join("frontend/dist", requestPath)
	data, err := fs.ReadFile(frontendFS, filePath)
	if err != nil {
		// 文件不存在 → 回退到 index.html（Vue Router 是前端路由）
		data, err = fs.ReadFile(frontendFS, "frontend/dist/index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "frontend/dist/index.html not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		/*
三个参数
参数	值	含义
第1个	http.StatusOK (即 200)	HTTP 状态码
第2个	"text/html; charset=utf-8"	Content-Type 响应头
第3个	data (即 []byte)	要写入响应体的原始字节内容
等价于标准库的

w.WriteHeader(200)
w.Header().Set("Content-Type", "text/html; charset=utf-8")
w.Write(data)
和 c.HTML() / c.String() / c.JSON() 的区别
c.JSON() — 把结构体/Map 序列化为 JSON 写到 body，自动设 Content-Type: application/json
c.String() — 把字符串写到 body，自动设 Content-Type: text/plain
c.HTML() — 用模板引擎渲染 HTML 写到 body
c.Data() — 把已有的 []byte 原样写入 body，Content-Type 你手动指定
在你代码里的上下文 (第 44-50 行)

// 当请求的文件（比如 /some/page）在 frontend/dist 中找不到时
// 退回到 index.html——因为 Vue Router 是前端路由，
// 浏览器端会自己根据 URL 渲染对应页面
data, err = fs.ReadFile(frontendFS, "frontend/dist/index.html")
// ...
c.Data(http.StatusOK, "text/html; charset=utf-8", data)
这里用 c.Data() 是因为 frontendFS 是 embed.FS，fs.ReadFile 返回的就是 []byte——内容已经是现成的 HTML 文件数据，直接吐给浏览器就行，不需要模板渲染也不需要序列化。*/
		return
	}

	contentType := mime.TypeByExtension(path.Ext(filePath))
/*
逐行解释

contentType := mime.TypeByExtension(path.Ext(filePath))
path.Ext(filePath) — 取文件扩展名，比如 "app.js" → ".js"，"style.css" → ".css"
mime.TypeByExtension() — 把扩展名映射为 MIME 类型，比如 .js → "text/javascript; charset=utf-8"，.css → "text/css; charset=utf-8"，.svg → "image/svg+xml"

if contentType == "" {
    contentType = "application/octet-stream"
}
如果扩展名不认识（比如 .wasm、.woff2 或自定义后缀），mime.TypeByExtension 返回空字符串
此时兜底为 "application/octet-stream" — 通用二进制流类型，浏览器会尝试下载而不是渲染*/
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Data(http.StatusOK, contentType, data)
}
```

**逐行解释：**

| 代码 | 什么意思 |
|------|---------|
| `package main` | Go 的入口，可执行程序必须是 `package main` |
| `gin.Default()` | 创建一个 Gin 路由器，自带日志和恢复中间件 |
| `r.GET("/api/health", ...)` | 注册 GET 接口 |
| `c.JSON(200, gin.H{...})` | 返回 JSON 响应 |
| `r.NoRoute(serveFrontend)` | 所有没匹配到的请求都交给 `serveFrontend` 处理 |
| `fs.ReadFile(frontendFS, ...)` | 从嵌入的虚拟文件系统读文件 |
| `mime.TypeByExtension(...)` | 根据扩展名自动返回正确的 Content-Type（`.js` → `text/javascript` 等） |
| `r.Run("127.0.0.1:8000")` | 启动服务器 |

**这段代码在干什么？**

`main.go` 是整个后端的**总入口**，相当于 Python 项目的 `app.py`。它做三件事：

**第一块：创建路由器**（第 11 行）

```go
r := gin.Default()
```

`gin.Default()` 创建一个"空的路由表"。"路由表"就像一个电话总机——你告诉它"这个号码打到张三，那个号码打到李四"，它负责转接。这里的"号码"就是 URL 路径。

**第二块：注册路由**（第 14 行 + 第 17 行）

```go
r.GET("/api/health", func(c *gin.Context) { ... })  // 健康检查
r.NoRoute(serveFrontend)                              // 前端页面
```

`r.GET("/api/health", ...)` 的意思是：**有人访问 GET /api/health 时，执行这个函数**。

`r.NoRoute(serveFrontend)` 的意思是：**上面都没匹配到的请求，统统交给 serveFrontend**。比如用户访问 `/settings`，路由表里没有 `/settings`，就走到 `serveFrontend`，然后它返回 `index.html` 让 Vue 接管。

后续每写完一个功能（比如科目管理），就是在这里加一行路由注册，比如 `r.GET("/api/subjects", handlers.GetSubjects)`。

**第三块：启动**（第 19 行）

```go
r.Run("127.0.0.1:8000")
```

开始监听本机 8000 端口，等待浏览器请求。`127.0.0.1` 表示只有本机能访问（安全）。

**`serveFrontend` 函数在干什么？**

```
浏览器请求 /index.html
  → strings.HasPrefix("/index.html", "/api") → false（不是 API）
  → strings.TrimPrefix("/index.html", "/") → "index.html"
  → path.Join("frontend/dist", "index.html") → "frontend/dist/index.html"
  → fs.ReadFile(frontendFS, "frontend/dist/index.html") → 读取文件内容
  → 返回给浏览器 ✓

浏览器请求 /subjects（Vue 前端路由）
  → HasPrefix → false
  → 尝试读取 frontend/dist/subjects → 文件不存在！
  → 回退：读取 frontend/dist/index.html → 返回
  → Vue Router 在浏览器端接管 /subjects → 渲染科目页面 ✓

浏览器请求 /api/xyz（不存在的接口）
  → HasPrefix → true
  → 返回 404 {"detail":"接口不存在"} ✓
```

**整个 main.go 可以类比为餐厅的大门：**

| main.go 的部件 | 餐厅类比 |
|---------------|---------|
| `r := gin.Default()` | 建一个空餐厅 |
| `r.GET("/api/health", ...)` | "健康检查台"——告诉客人这里可以问"开门了吗" |
| `r.GET("/api/subjects", ...)` | "科目窗口"——告诉客人这里可以点科目 |
| `r.NoRoute(serveFrontend)` | "前台大厅"——不是来点菜的客人，引导去大厅（Vue 页面） |
| `r.Run(...)` | 开门营业

### Step 1.4：运行

```powershell
go run .
```

### ✅ 验证

**终端验证：**
```powershell
curl http://127.0.0.1:8000/api/health
```
返回 `{"status":"ok"}`。

**浏览器验证（这是重点）：** 打开浏览器访问 `http://127.0.0.1:8000`

你应该能看到 **Vue 前端页面正常加载出来了**（样式、布局都对）。但页面数据区域会显示加载失败或空数据——这是正常的，因为 API 还没写。

> 🎯 **这就是前端交互验证的起点。** 之后每完成一个 Part，刷新这个页面，对应的功能模块就会从"报错/空白"变成正常显示数据。你能直观感受到进度。

用 `Ctrl+C` 停掉服务器，继续下一步。

### 💡 新手提示

- Go 的 `{` 不能换行！`func main() {` 不能写成 `func main()\n{`
- 改完代码用 `go run .` 重新启动（`. ` 表示当前目录的整个 package）
- 如果端口被占用，改 `r.Run` 的参数，比如 `"127.0.0.1:8001"`
- 每次改完代码重新启动后，记得**刷新浏览器**看效果

---

## Part 2：数据模型（models）

### 目标

定义错题、请求体、返回体等数据结构。这些结构体会被后面的所有代码使用。

**为什么要先写 models？**

Go 是静态类型语言，后面的 handler、service、store 都要围绕同一套结构体工作。先把 `ErrorProblem`、请求体、返回体定义清楚，相当于先定数据契约：

```text
前端 JSON 字段名
↔ Gin 绑定请求体
↔ service 操作结构体
↔ store 写入 JSON 文件
```

如果模型层不稳定，后面每个函数都会跟着改。尤其是 `json:"next_review"` 这种标签，它不是装饰品，而是前后端之间的字段名协议：Go 里字段必须大写才能被其他包访问，但 JSON 里前端要的是小写加下划线。

### Step 2.1：检查已有的 models.go

`models/models.go` 已经有两个结构体了。打开看看，了解 `ErrorProblem` 的字段含义：

- `ID` — 错题编号
- `Subject` — 科目（数学、英语…）
- `Title` — 标题
- `Question` — 题目内容
- `Wrong` — 错误答案
- `Correct` — 正确答案
- `Reason` — 错误原因
- `Tags` — 标签列表
- `ReasonTags` — 错误原因标签
- `Created` — 创建时间
- `ReviewCount` / `ReviewStage` / `LastReview` / `NextReview` — 艾宾浩斯复习

### Step 2.2：修改 LastReview 为指针类型

原 Python 代码中 `last_review` 可能是 `None`（还没复习过）。Go 里要表示 "可能是 null"，用指针 `*string`。

把 `LastReview` 那行从：

```go
LastReview  string   `json:"last_review"`
```

改为：

```go
LastReview  *string  `json:"last_review"`
```

> **什么是指针？** `*string` 意思是"指向 string 的指针"。它可以有三种值：指向某个字符串、或者 `nil`（相当于 Python 的 `None`）。JSON 序列化时 `nil` 会变成 `null`。

### Step 2.3：添加请求/响应结构体

在 `models.go` 文件末尾追加以下代码。**追加**，不要覆盖已有的。

```go
// AddErrorRequest 是创建错题时的请求体
type AddErrorRequest struct {
	Subject    string   `json:"subject"`
	Question   string   `json:"question"`
	Title      string   `json:"title"`
	Wrong      string   `json:"wrong"`
	Correct    string   `json:"correct"`
	Reason     string   `json:"reason"`
	Tags       []string `json:"tags"`
	ReasonTags []string `json:"reason_tags"`
}

// UpdateErrorRequest 是更新错题时的请求体
// 所有字段都用指针，因为前端可能只传部分字段
// nil 表示"这个字段没传，不要更新"
type UpdateErrorRequest struct {
	Subject    *string   `json:"subject"`
	Title      *string   `json:"title"`
	Question   *string   `json:"question"`
	Wrong      *string   `json:"wrong"`
	Correct    *string   `json:"correct"`
	Reason     *string   `json:"reason"`
	Tags       *[]string `json:"tags"`
	ReasonTags *[]string `json:"reason_tags"`
}

// DailyPushResult 是每日推送的返回数据
type DailyPushResult struct {
	Date         string            `json:"date"`
	TotalErrors  int               `json:"total_errors"`
	Reviewed     int               `json:"reviewed"`
	DueCount     int               `json:"due_count"`
	OverdueCount int               `json:"overdue_count"`
	Knowledge    map[string]string `json:"knowledge"`
	WeakErrors   []ErrorProblem    `json:"weak_errors"`
	Advice       string            `json:"advice"`
}
```

**逐行解释：**

| 代码 | 什么意思 |
|------|---------|
| `*string` | 指向字符串的指针，可以是 `nil` |
| `*[]string` | 指向字符串切片的指针 |
| `map[string]string` | 键值对，键是 string，值也是 string |

为什么 `UpdateErrorRequest` 要用指针？举个例子，前端发送：

```json
{"title": "新标题"}
```

Go 解析后：
- `Title` 指向 `"新标题"` ✅
- `Subject`、`Question` 等是 `nil` ❌（表示没传）

这样 service 层就能区分"没传"和"传了空字符串"。

**这段代码在干什么？**

这三个结构体定义了"请求和响应的数据格式"。

**`AddErrorRequest` — 创建错题时，前端发来的 JSON 长什么样：**

```text
前端 POST /api/errors 发送 →
{"subject":"数学","question":"1+1=?","wrong":"3","correct":"2"}

Gin 自动解析 →
AddErrorRequest{Subject:"数学", Question:"1+1=?", Wrong:"3", Correct:"2"}
```

对比 `ErrorProblem`：`ErrorProblem` 有 ID、创建时间、复习次数等字段，这些是**服务端生成的**，不是前端传的。所以需要一个专门的"请求体"结构体，只包含前端能传的字段。

**`UpdateErrorRequest` — 更新错题时，前端可能只传部分字段：**

这是整个项目里最重要的设计决策。为什么所有字段都是指针（`*string`、`*[]string`）？

```text
场景：用户只改了标题，前端发送 →
{"title": "新标题"}

如果字段是普通 string：
  Subject 被解析为 ""（空字符串）→ 后端不知道是"没传"还是"想清空"

如果字段是 *string（指针）：
  Subject 被解析为 nil → 后端明确知道"这个字段没传，不要改"
  Title 被解析为 &"新标题" → 后端知道"这个字段传了，要更新"
```

这就是为什么 `UpdateErrorRequest` 全用指针——**区分"没传"和"传了空字符串"**。

**`DailyPushResult` — 每日推送返回的数据格式：**

```text
前端 GET /api/daily-push →
{"date":"2026-06-18","total_errors":28,"due_count":5,...}
```

单独定义一个结构体，比用 `map[string]interface{}` 更安全——字段名写错了编译器会报错，不会等到前端发现数据不对。

### ✅ 验证

```powershell
go build .
```

如果能编译（没有报错），说明 models 没问题。

> `go build .` 会检查当前目录所有 `.go` 文件能否一起编译。我们只有 `main.go` 引用了 models，所以能通过就说明 import 路径正确。

---

## Part 3：JSON 文件读写（store）

### 目标

补全 store 层，增加两个公开函数供 handlers 使用。

**store 层的设计边界：只管文件，不管业务。**

`store` 包只应该知道"数据文件在哪、怎么读、怎么写"，不应该知道"错题标题能不能空"、"科目是否重复"、"今天哪些题到期"。这些业务规则放在 service 层。这样以后如果把 JSON 换成 SQLite，handler 和 service 大概率不用大改，只换 store 层即可。

这一层最重要的设计选择是：**文件不存在不算错误**。第一次运行程序时，`data/errors.json` 可能还没有；这时返回空数据比直接报错更符合用户预期。真正要报错的是文件存在但读不了、JSON 格式坏了、写入失败这类问题。

### Step 3.1：检查已有的 json_store.go

`store/json_store.go` 已经有：
- `LoadJSON(filename, &target)` — 读 JSON 文件
- `SaveJSON(filename, data)` — 写 JSON 文件
- `SetDataDir(dir)` — 设置数据目录

### Step 3.2：添加 DataDir() 和 Path() 函数

在 `store/json_store.go` 文件末尾追加：

```go
// DataDir 返回数据目录路径，如果目录不存在则自动创建
func DataDir() string {
	_ = os.MkdirAll(dataDir, 0755)
	return dataDir
}

// Path 返回数据文件的完整路径，如果目录不存在则自动创建
func Path(filename string) string {
	_ = os.MkdirAll(dataDir, 0755)
	return filepath.Join(dataDir, filename)
}
```

**这段代码在干什么？**

`DataDir()` 和 `Path()` 是给其他包用的"工具函数"。它们解决的问题是：**不要在每个文件里重复拼路径**。

**为什么需要它们？**

```text
场景：备份功能需要知道数据目录在哪
  没有这些函数 → 每个包自己写 filepath.Join("data", "xxx")
  有了这些函数 → 统一用 store.DataDir() 和 store.Path("errors.json")
```

如果将来想把数据目录从 `data` 改成 `user_data`，只需要改 `dataDir` 变量一处，不用满项目搜索替换。

**`os.MkdirAll` 为什么写了两次？**

两个函数里都调了 `os.MkdirAll`，看起来很重复，但这是故意为之——你只调用 `Path("errors.json")`，它也能确保目录存在，不需要先调用一次 `DataDir()`。

`_ = ` 前缀的意思是"忽略返回值"。`os.MkdirAll` 返回 `error`，但目录已存在也算正常，没必要处理这个错误。

**`filepath.Join` vs 字符串拼接：**

```go
// 不好的写法
path := "data/" + filename        // Linux 能跑，Windows 路径分隔符不对

// 好的写法
path := filepath.Join("data", filename)  // 自动适配不同系统
```

### ✅ 验证

```powershell
go build .
```

编译通过即可。

---

## Part 4：科目管理（第一个完整功能）

### 目标

走通 **三层架构的完整流程**：store → service → handler → 路由。

**为什么第一个完整功能选"科目管理"？**

科目功能足够小，但覆盖了后端最重要的完整链路：

```text
浏览器请求
→ handler 解析 HTTP
→ service 校验业务规则
→ store 读写 subjects.json
→ handler 返回前端需要的 JSON
```

这个链路跑通后，后面的错题、设置、备份本质上都是同一种结构，只是业务规则更多。先用小功能练三层架构，比一上来写错题 CRUD 更容易定位问题。

科目数据存在 `subjects.json`，格式是 `["数学","英语","物理"]`。

### Step 4.1：写 service/subject_service.go

创建 `service/subject_service.go`：

```go
package service

import (
	"fmt"
	"strings"

	"study-tracker-go/store"
)

// GetAllSubjects 获取所有科目
func GetAllSubjects() ([]string, error) {
	var subjects []string
	if err := store.LoadJSON("subjects.json", &subjects); err != nil {
		return nil, err
	}
	// 如果文件不存在，返回空切片而非 nil
	// 这样 JSON 输出是 [] 而不是 null
	if subjects == nil {
		subjects = []string{}
	}
	return subjects, nil
}
```

**这段代码在干什么？**

`GetAllSubjects` 是三层架构中"service 层"的第一个函数。它展示了 service 层的标准写法：**从 store 读数据 → 做一些处理 → 返回给 handler**。

**返回值的含义：**

```go
func GetAllSubjects() ([]string, error) {
//                   ↑          ↑
//                  科目列表    如果有错误，列表为 nil，错误不为 nil
```

Go 里几乎所有可能失败的函数都返回 `(结果, error)` 对。调用方先检查 `error`，如果不为 `nil` 说明出错了，直接向上传递。

**`nil` vs 空切片：**

```go
var subjects []string   // subjects 是 nil，JSON 输出 null
subjects = []string{}   // subjects 是空切片，JSON 输出 []
```

前端对 `null` 的处理通常不如 `[]` 好（比如 `null.map()` 会报错，`[].map()` 不会）。所以 service 层有这个责任：**永远不要让前端收到 `null` 切片**。

**`store.LoadJSON` 的第二个参数为什么是 `&subjects`：**

`&` 是取地址符。`LoadJSON` 拿到 subjects 的内存地址后，才能把 JSON 数据"写进"这个变量。如果不传地址，函数内部改的是副本，调用方拿不到结果。

在同一个文件继续追加 `AddSubject`：

```go
// AddSubject 添加一个科目
func AddSubject(name string) ([]string, error) {
	name = strings.TrimSpace(name) // 去掉前后空格
	if name == "" {
		return nil, fmt.Errorf("科目名称不能为空")
	}

	subjects, err := GetAllSubjects()
	if err != nil {
		return nil, err
	}

	// 检查是否已存在
	for _, s := range subjects {
		if s == name {
			return nil, fmt.Errorf("科目已存在")
			/*Go 的约定是：err != nil 时，其他返回值不可靠，调用者不应使用。 所以出错时返回 nil 没人看，无所谓。 */
		}
	}

	subjects = append(subjects, name)
	if err := store.SaveJSON("subjects.json", subjects); err != nil {
		return nil, err
	}
	return subjects, nil
}
```

**这段代码在干什么？**

`AddSubject` 展示了"写操作"的标准流程：**校验 → 读 → 改 → 保存 → 返回**。

**为什么先校验再操作：**

```text
好的做法：先检查输入是否合法 → 不合法直接返回错误，不碰磁盘
坏的做法：先保存再检查 → 脏数据已经写进去了
```

这叫"fail fast"——尽早发现错误，尽早返回。

**`fmt.Errorf` vs `errors.New`：**

```go
fmt.Errorf("科目已存在")            // 简单消息
fmt.Errorf("科目 %s 已存在", name)  // 支持占位符
```

有 `%s`、`%d` 等占位符需求时用 `fmt.Errorf`，只需固定消息时也能用 `errors.New("message")`。

**`append(subjects, name)`：**

`append` 往切片末尾加一个元素，并返回新切片。注意它**不会修改原切片**（可能触发扩容，返回的是新切片），所以必须把返回值赋回去。

继续追加 `SubjectExists` 和 `DeleteSubject`：

```go
// SubjectExists 检查科目是否存在
func SubjectExists(name string) bool {
	subjects, err := GetAllSubjects()
	if err != nil {
		return false
	}
	for _, s := range subjects {
		if s == name {
			return true
		}
	}
	return false
}

// DeleteSubject 删除一个科目
func DeleteSubject(name string) ([]string, error) {
	subjects, err := GetAllSubjects()
	if err != nil {
		return nil, err
	}

	found := false
	remaining := []string{}
	for _, s := range subjects {
		if s == name {
			found = true
			continue // 跳过这个，不加入 remaining
		}
		remaining = append(remaining, s)
	}

	if !found {
		return nil, fmt.Errorf("科目不存在")
	}

	if err := store.SaveJSON("subjects.json", remaining); err != nil {
		return nil, err
	}
	return remaining, nil
}
```

**这段代码在干什么？**

两个函数展示了"查找然后操作"的两种模式。

`SubjectExists` — 只读查找：问"这个科目在不在？"它被 `CreateError` 和 `UpdateError` 调用来验证用户输入的科目名是否合法。

`DeleteSubject` — 查找后修改：Go 里从切片删除的标准做法是"新建切片，把不删的放进去"。

```
原切片: ["数学", "英语", "物理"]  要删除: "英语"
遍历: 数学≠英语→加入, 英语=英语→跳过, 物理≠英语→加入
结果: ["数学", "物理"]
```

`found` 变量用来区分"找到了并删除"和"根本不存在"——如果遍历完 `found` 还是 `false`，返回 `"科目不存在"` 错误。

### Step 4.2：写 handlers/subjects.go

创建 `handlers/subjects.go`：

```go
package handlers

import (
	"net/http"

	"study-tracker-go/service"

	"github.com/gin-gonic/gin"
)

// GetSubjects 处理 GET /api/subjects
func GetSubjects(c *gin.Context) {
	subjects, err := service.GetAllSubjects()
	if err != nil {
		// 前端优先读 detail 字段，错误统一用这个格式
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	// 前端读 res.subjects，必须用这个字段名
	c.JSON(http.StatusOK, gin.H{"subjects": subjects})
}
```

**这段代码在干什么？**

handler 层是"服务员"——只做三件事：接收请求 → 调 service → 返回响应。

```
浏览器 GET /api/subjects
  → handler.GetSubjects 收到 Gin 上下文 c
  → service.GetAllSubjects() 返回 ["数学","英语","物理"]
  → c.JSON(200, gin.H{"subjects": [...]})
  → 浏览器收到 {"subjects":["数学","英语","物理"]}
```

**为什么返回格式是 `{"subjects": [...]}` 而不是裸数组 `[...]`？**

因为 Vue 前端代码写的是 `res.subjects`。如果返回裸数组，`res.subjects` 就是 `undefined`，页面显示空白。

**`gin.H` 是什么？** 它是 `map[string]interface{}` 的简写。`gin.H{"subjects": subjects}` 等于手写 `map[string]interface{}{"subjects": subjects}`。

**错误处理模式：** 每个 handler 都遵循同样的模式——先检查 `err`，有错误就返回 `{"detail": "..."}`，没错误才返回正常数据。前端统一读 `detail` 字段显示错误。

继续在同一个文件追加 `AddSubject` 和 `DeleteSubject`：

```go
// AddSubject 处理 POST /api/subjects
func AddSubject(c *gin.Context) {
	// 解析请求体中的 JSON
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}

	subjects, err := service.AddSubject(body.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"subjects": subjects})
}

// DeleteSubject 处理 DELETE /api/subjects/:name
func DeleteSubject(c *gin.Context) {
	name := c.Param("name") // 从 URL 路径中取参数
	subjects, err := service.DeleteSubject(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"subjects": subjects})
}
```

> 注意：Go 的函数名大小写完全敏感。当前仓库里如果已经有 `func Addsubject(...)`（小写 s），请把它改成 `func AddSubject(...)`，并确保 `main.go` 里注册的是 `handlers.AddSubject`。否则路由代码和 handler 名字对不上，编译时会提示找不到函数。

**逐行解释：**

| 代码 | 什么意思 |
|------|---------|
| `var body struct { ... }` | 定义一个临时结构体，只在这个函数里用 |
| `c.ShouldBindJSON(&body)` | 把请求体的 JSON 解析到 body 里 |
| `c.Param("name")` | 获取 URL 路径中的 `:name`，比如 `/api/subjects/数学` → `"数学"` |

### Step 4.4：注册路由

在 `main.go` 的 `import` 块中增加 `"study-tracker-go/handlers"`，然后在 `main()` 中加上三行路由。

你的 `main.go` 现在应该是这样（只展示结构，省去已有代码）：

```go
package main

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"study-tracker-go/handlers"  // ← 新增

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// === 科目接口（新增） ===
	r.GET("/api/subjects", handlers.GetSubjects)
	r.POST("/api/subjects", handlers.AddSubject)
	r.DELETE("/api/subjects/:name", handlers.DeleteSubject)

	r.NoRoute(serveFrontend)
	r.Run("127.0.0.1:8000")
}
// ... serveFrontend 保持不变
```

### Step 4.5：运行并验证

```powershell
go run .
```

### ✅ 验证

**curl 验证（逐接口测试）：**

```powershell
# 1. 查看所有科目
curl http://127.0.0.1:8000/api/subjects

# 2. 添加科目
curl -X POST http://127.0.0.1:8000/api/subjects -H "Content-Type: application/json" -d "{\"name\":\"测试科目\"}"

# 3. 重复添加（应报错）
curl -X POST http://127.0.0.1:8000/api/subjects -H "Content-Type: application/json" -d "{\"name\":\"测试科目\"}"

# 4. 删除
curl -X DELETE http://127.0.0.1:8000/api/subjects/测试科目
```

> curl 在 PowerShell 中 JSON 的双引号需要 `\"` 转义。

**浏览器验证（前端交互检查）：** 打开 `http://127.0.0.1:8000`

进入**科目管理页面**（通常是侧边栏或顶部导航里的"科目"或"学科"）。你会看到：

- ✅ 科目列表显示出来了（不再空白/报错）
- ✅ 添加科目功能可用
- ✅ 删除科目功能可用

> 🎯 **这是第一个里程碑：前端 + 后端科目功能联通了！** 其他页面（错题列表等）还会报错——接着往下做。

### 💡 新手提示：三层架构的数据流

当你访问 `GET /api/subjects` 时，数据是这样流动的：

```
浏览器 Vue 页面 → fetch('/api/subjects')
                       ↓
                 Gin 路由 → handlers.GetSubjects()
                                ↓
                          service.GetAllSubjects()
                                ↓
                          store.LoadJSON("subjects.json")
                                ↓
                          读取 data/subjects.json → []string
                                ↓
                          handler 包装成 {"subjects": [...]}
                                ↓
                          浏览器收到 JSON → Vue 渲染列表
```

后面所有功能都遵循这个模式：**handler 只是"服务员"，service 是"厨师"，store 是"冰箱"。**

---

## Part 5：错题管理——创建和查询

### 目标

实现错题的创建和查询。错题要比科目复杂得多，我们分两 Part 完成。

**这一 Part 为什么复杂？**

科目只是一个字符串数组，错题是一条完整记录。它既有用户输入的字段（题目、答案、原因、标签），也有服务端生成的字段（ID、创建时间、复习次数、下次复习日期）。所以错题功能的核心不是"把前端 JSON 保存起来"这么简单，而是要把一份可能不完整的请求补成一条前端能稳定使用的完整记录。

这里要记住两条线：

| 位置 | 做什么 | 解决的问题 |
|------|--------|------------|
| 创建时 `CreateError` | 给新错题补默认值 | 新数据写入时就尽量完整 |
| 查询时 `normalizeReviewFields` | 给旧错题/导入错题补缺失字段 | 兼容旧 JSON、手动编辑、备份导入 |

这就是为什么文档里会反复出现 `nil`、空切片、默认日期这些细节。它们不是为了让代码好看，而是为了保证前端永远拿到形状稳定的数据。

### Step 5.1：写 service/error_service.go（第一部分）

创建 `service/error_service.go`，先写**创建错题**的逻辑：

```go
package service

import (
	"fmt"
	"strings"
	"time"

	"study-tracker-go/models"
	"study-tracker-go/store"
)

// CreateError 创建一条新错题
func CreateError(req models.AddErrorRequest) (models.ErrorProblem, error) {
	// 清理输入
	req.Subject = strings.TrimSpace(req.Subject)
	req.Question = strings.TrimSpace(req.Question)

	// 验证科目必须存在
	if !SubjectExists(req.Subject) {
		return models.ErrorProblem{}, fmt.Errorf("无效科目")
	}
	if req.Question == "" {
		return models.ErrorProblem{}, fmt.Errorf("题目不能为空")
	}

	// 处理默认值
	if req.Wrong == "" {
		req.Wrong = "未记录"
	}
	if req.Correct == "" {
		req.Correct = "未记录"
	}
	if req.Reason == "" {
		req.Reason = "未记录"
	}
	if req.Title == "" {
		req.Title = firstRunes(req.Question, 40)
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}
	if req.ReasonTags == nil {
		req.ReasonTags = []string{}
	}

	// 读取已有错题，计算新 ID
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return models.ErrorProblem{}, err
	}

	nextID := 1
	for _, item := range errors {
		if item.ID >= nextID {
			nextID = item.ID + 1
		}
	}

	// 构造新错题
	now := time.Now()
	item := models.ErrorProblem{
		ID:          nextID,
		Subject:     req.Subject,
		Title:       req.Title,
		Question:    req.Question,
		Wrong:       req.Wrong,
		Correct:     req.Correct,
		Reason:      req.Reason,
		Tags:        req.Tags,
		ReasonTags:  req.ReasonTags,
		Created:     now.Format("2006-01-02 15:04:05"),
		ReviewCount: 0,
		LastReview:  nil, // 还没复习过
		ReviewStage: 0,
		NextReview:  now.Format("2006-01-02"),
	}

	errors = append(errors, item)
	if err := store.SaveJSON("errors.json", errors); err != nil {
		return models.ErrorProblem{}, err
	}
	return item, nil
}

// firstRunes 取字符串的前 max 个字符（按 rune 而非 byte）
func firstRunes(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max])
}
```

**这段代码在干什么？**

`CreateError` 是 service 层创建错题的核心函数。它的职责是：**接收前端传来的请求数据，经过验证和默认值处理，构造成一条完整的 `ErrorProblem` 记录，追加到 JSON 文件中保存**。

整个函数的设计体现了 service 层的标准模式：**验证输入 → 处理默认值 → 读取数据 → 构造新记录 → 写回文件**。

**逐层拆解：**

1. **输入清理**：`strings.TrimSpace` 去掉首尾空格，防止用户不小心输入了空白字符。
2. **业务验证**：科目必须已经存在（调用 `SubjectExists`），题目不能为空。这两个验证是业务规则，不属于 handler 层的"格式检查"。
3. **默认值处理**：如果用户没填错误答案、正确答案、原因，统一填"未记录"；标题为空时，自动取题目前 40 个字符作为标题——`firstRunes` 用 `[]rune` 而非字节切片，确保中文一个字就是一个字符。
4. **空切片处理**：`Tags` 和 `ReasonTags` 为 `nil` 时设为 `[]string{}`，这样 JSON 序列化输出 `[]` 而不是 `null`，前端更友好。
5. **ID 计算**：遍历已有错题，找到最大 ID 后 +1。这是一种简单的自增 ID 策略，适合 JSON 文件存储。
6. **时间处理**：`time.Now().Format("2006-01-02 15:04:05")` 是 Go 独有的时间格式化方式——用固定的参考时间 `Mon Jan 2 15:04:05 MST 2006`（即 1 2 3 4 5 6 7）作为模板。`LastReview` 是 `*string` 类型，用 `nil` 表示"还没复习过"，这是 Go 中表示可选字段的惯用做法。

**为什么返回 `(models.ErrorProblem, error)`？**  
调用方（handler）需要知道创建出来的错题 ID，以便返回给前端。如果只返回 `error`，handler 就不知道新建记录的 ID 是多少了。

**逐行解释：**

| 代码 | 什么意思 |
|------|---------|
| `models.ErrorProblem{}` | 创建一个空的 ErrorProblem 结构体 |
| `time.Now().Format("2006-01-02 15:04:05")` | Go 时间格式化必须用 `2006-01-02 15:04:05` 这个固定的参考时间，不是 `YYYY-MM-DD` |
| `LastReview: nil` | `*string` 类型的零值是 `nil`，表示未复习 |
| `[]rune(text)` | 将字符串转为 Unicode 字符切片，中文字符占 3 个字节，用 rune 才能正确计数 |
| `return models.ErrorProblem{}, err` | 返回空结构体 + 错误，调用方检查 error |

> **关于 Go 的时间格式：** Go 用了一个巧妙的记忆法——`Mon Jan 2 15:04:05 MST 2006`，就是 `1 2 3 4 5 6 7`。`2006`=年，`01`=月，`02`=日，`15`=时(24h)，`04`=分，`05`=秒。

### Step 5.2：添加查询功能

在同一个 `service/error_service.go` 文件末尾追加：

```go
// GetAllErrors 查询错题，支持按科目/关键词/标签筛选
func GetAllErrors(subject, keyword, tag, reasonTag string) ([]models.ErrorProblem, error) {
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return nil, err
	}
	// nil 切片转空切片
	if errors == nil {
		return []models.ErrorProblem{}, nil
	}

	result := []models.ErrorProblem{}
	for _, item := range errors {
		// 补全缺失的默认值
		normalizeReviewFields(&item)

		// 逐项筛选
		if subject != "" && subject != "全部" && item.Subject != subject {
			continue
		}
		if keyword != "" && !matchesKeyword(item, keyword) {
			continue
		}
		if tag != "" && !listContainsFold(item.Tags, tag) {
			continue
		}
		if reasonTag != "" && !listContainsFold(item.ReasonTags, reasonTag) {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

// normalizeReviewFields 补全字段默认值
/*
为什么读取时还要补默认值？

CreateError 只能保证"新代码创建的新错题"字段完整，但 errors.json 可能来自旧版本、手动编辑或备份导入。
这些数据可能没有 tags、reason_tags、next_review。查询接口如果原样返回，前端就可能拿到 null 或空字符串，
导致标签渲染、日期排序、每日推送判断出问题。

这里的原则是：写入时尽量严格，读取时要宽容。只要能推导出合理默认值，就在读出来后补齐。 */
func normalizeReviewFields(item *models.ErrorProblem) {
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if item.ReasonTags == nil {
		item.ReasonTags = []string{}
	}
	if item.NextReview == "" {
		if len(item.Created) >= 10 {
			item.NextReview = item.Created[:10]
		} else {
			item.NextReview = time.Now().Format("2006-01-02")
		}
	}
}

// matchesKeyword 检查错题是否匹配关键词
func matchesKeyword(item models.ErrorProblem, keyword string) bool {
	keyword = strings.ToLower(keyword)
	if strings.Contains(strings.ToLower(item.Question), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(item.Title), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(item.Reason), keyword) {
		return true
	}
	return listContainsFold(item.Tags, keyword) || listContainsFold(item.ReasonTags, keyword)
}

// listContainsFold 检查字符串切片是否包含关键词（忽略大小写）
func listContainsFold(list []string, keyword string) bool {
	keyword = strings.ToLower(keyword)
	for _, item := range list {
		if strings.Contains(strings.ToLower(item), keyword) {
			return true
		}
	}
	return false
}
```

**这段代码在干什么？**

这一组函数实现了错题的**查询和筛选**功能。`GetAllErrors` 是入口，它从 JSON 文件加载所有错题，然后根据四个可选参数逐条过滤，返回符合条件的结果。

**关键设计思路：**

1. **"过滤器链"模式**：`GetAllErrors` 的主体是一个 `for` 循环，里面用四个 `if ... continue` 串联起不同的筛选条件。每个条件独立判断，不满足就 `continue` 跳过。这种写法比嵌套 `if` 清晰得多，新增筛选条件时只需加一段 `if`，不影响其他逻辑。

2. **nil 切片转空切片**：`if errors == nil { return []models.ErrorProblem{}, nil }` 确保即使 JSON 文件为空或不存在，也返回空数组 `[]` 而非 `null`。这是 API 兼容性约束——前端期望 `res.errors` 永远是一个数组。

3. **`normalizeReviewFields` 的指针用法**：传 `&item`（地址），函数直接修改原变量，不需要返回值。这是 Go 中常见的"就地修改"模式。它做的事情是补全历史数据中可能缺失的字段（比如旧数据没有 `NextReview`），让旧数据兼容新逻辑。

4. **为什么 `NextReview` 要这样补：**

   ```go
   if item.NextReview == "" {
       if len(item.Created) >= 10 {
           item.NextReview = item.Created[:10]
       } else {
           item.NextReview = time.Now().Format("2006-01-02")
       }
   }
   ```

   这段不是随便挑一个日期，而是在给旧数据找一个最合理的复习起点。`NextReview` 为空通常说明这条错题来自旧版本、手动编辑或导入数据；如果 `Created` 是正常的 `"2026-06-03 12:52:22"`，前 10 个字符正好是创建日期 `"2026-06-03"`，用它作为下次复习日期，含义是"这道旧题至少从创建日开始就应该进入复习队列"。如果 `Created` 连 10 个字符都没有，直接 `item.Created[:10]` 会 panic，所以先判断长度。最后用今天日期兜底，是为了保证前端和每日推送永远有一个合法的 `YYYY-MM-DD` 可排序、可比较，而不是拿到空字符串后出现错乱。

5. **大小写不敏感的匹配**：`matchesKeyword` 和 `listContainsFold` 都用 `strings.ToLower` 统一转小写后再比较。这是文本搜索的基本操作——用户搜"导数"和"導數"虽然中文没区别，但搜"Math"和"math"就有区别了。

6. **为什么 `matchesKeyword` 检查多个字段？** 用户搜一个关键词时，可能出现在题目中、标题中、原因中、或者标签里。同时在所有字段中查找，搜索体验更好。

**逐行解释：**

| 代码 | 什么意思 |
|------|---------|
| `normalizeReviewFields(&item)` | 传 `&item`（地址），函数直接修改 item，不需要返回值 |
| `item.Created[:10]` | 切片操作，取前 10 个字符（`"2006-01-02"` 正好 10 个字符） |
| `strings.ToLower(keyword)` | 转小写，实现大小写不敏感的搜索 |
| `continue` | 跳过当前循环项，进入下一次循环 |

### Step 5.3：写 handlers/errors.go（第一部分）

创建 `handlers/errors.go`：

```go
package handlers

import (
	"net/http"

	"study-tracker-go/models"
	"study-tracker-go/service"

	"github.com/gin-gonic/gin"
)

// CreateError 处理 POST /api/errors
func CreateError(c *gin.Context) {
	var req models.AddErrorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}

	item, err := service.CreateError(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": item.ID, "message": "添加成功"})
}

// GetErrors 处理 GET /api/errors
func GetErrors(c *gin.Context) {
	errors, err := service.GetAllErrors(
		c.Query("subject"),   // ?subject=数学
		c.Query("keyword"),   // ?keyword=导数
		c.Query("tag"),       // ?tag=选择题
		c.Query("reason_tag"),// ?reason_tag=概念混淆
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	// 注意：前端读 res.errors 和 res.total
	c.JSON(http.StatusOK, gin.H{"errors": errors, "total": len(errors)})
}
```

**这段代码在干什么？**

这是 handler 层的两个函数——`CreateError` 和 `GetErrors`。handler 层的职责非常单一：**接收 HTTP 请求 → 解析参数 → 调用 service → 返回 HTTP 响应**。它不做任何业务逻辑，只是"传话筒"。

**两个函数的共同模式（Gin handler 的标准写法）：**

1. **解析请求**：`c.ShouldBindJSON(&req)` 把请求体 JSON 反序列化到 Go 结构体；`c.Query("subject")` 从 URL `?subject=xxx` 中取值。
2. **调用 service**：把解析好的参数传给 service 层函数，拿到结果和 error。
3. **处理错误**：如果 service 返回 error，用 `c.JSON` 返回错误响应，格式是 `{"detail": "..."}`——这是 API 兼容性约束，前端优先读 `detail` 字段来显示错误信息。
4. **返回成功响应**：同样用 `c.JSON`，HTTP 状态码 200，body 是 Gin 的 `H`（`map[string]any` 的快捷别名）。

**`CreateError` 的特殊之处：**  
返回的不是完整的错题对象，而是 `{"id": N, "message": "添加成功"}`。这是因为前端创建成功后只需要知道新记录的 ID 就够了（用来跳转或刷新列表），返回完整对象反而浪费带宽。

**`GetErrors` 的特殊之处：**  
返回格式必须是 `{"errors": [...], "total": N}`。注意：`total` 是筛选后的数量，不是总数量。前端用 `res.errors` 渲染列表，用 `res.total` 显示"共 N 条"。如果返回裸数组 `[...]`，前端会直接报错——这是最重要的 API 兼容性约束之一。

**逐行解释：**

| 代码 | 什么意思 |
|------|---------|
| `c.Query("subject")` | 取 URL 查询参数 `?subject=xxx` |
| `gin.H{"errors": errors, "total": len(errors)}` | 返回格式必须是 `{"errors": [...], "total": N}`，前端读 `res.errors` 和 `res.total` |

### Step 5.4：注册路由

在 `main.go` 的 `main()` 函数中，科目接口后面加上：

```go
	r.POST("/api/errors", handlers.CreateError)
	r.GET("/api/errors", handlers.GetErrors)
```

**这段代码在干什么？**

这两行在 Gin 路由器上注册了两个端点：

- `POST /api/errors`：收到 POST 请求时，Gin 调用 `handlers.CreateError` 处理。
- `GET /api/errors`：收到 GET 请求时，Gin 调用 `handlers.GetErrors` 处理。

Gin 的路由注册非常直观：`r.方法("路径", 处理函数)`。这里的 `handlers.CreateError` 和 `handlers.GetErrors` 是函数的引用（不是调用结果），Gin 会在请求到来时自动调用它们，并把 `*gin.Context` 传进去。

注意：这两行要和之前注册的科目路由放在一起，通常所有 `r.POST/r.GET/...` 集中在 `main()` 函数里、`r.Run()` 之前。

### ✅ 验证

```powershell
go run .
```

**curl 验证：**

```powershell
# 1. 查看所有错题
curl http://127.0.0.1:8000/api/errors

# 2. 按科目筛选
curl "http://127.0.0.1:8000/api/errors?subject=数学"

# 3. 添加一条错题
curl -X POST http://127.0.0.1:8000/api/errors -H "Content-Type: application/json" -d "{\"subject\":\"数学\",\"question\":\"1+1=?\",\"wrong\":\"3\",\"correct\":\"2\",\"reason\":\"粗心\"}"

# 4. 添加无效科目的错题（应报错）
curl -X POST http://127.0.0.1:8000/api/errors -H "Content-Type: application/json" -d "{\"subject\":\"不存在的科目\",\"question\":\"test\"}"
```

**浏览器验证：** 刷新 `http://127.0.0.1:8000`，进入**错题列表页面**：

- ✅ 错题列表显示出来了（不再是空白/加载错误）
- ✅ 科目筛选下拉框有数据了（因为科目 API 已通）
- ✅ 按科目筛选功能可用

> 🎯 **里程碑 2：前端错题列表通了！** 现在你能在页面上浏览所有错题了。

---

## Part 6：错题管理——更新和删除

### 目标

补全错题的更新和删除功能。

**更新功能的关键不是"把字段覆盖掉"，而是"只覆盖前端真的传了的字段"。**

编辑错题时，前端可能只改标题，只发送：

```json
{"title":"新标题"}
```

如果后端用普通 `string` 字段接收，没传的字段会变成空字符串，后端分不清"用户想清空"和"前端根本没传"。所以 `UpdateErrorRequest` 里字段都用指针：`nil` 表示没传，不更新；非 `nil` 表示传了，即使传的是空字符串，也代表用户有意修改。这是更新接口最重要的设计点。

删除功能看起来简单，但也有一个边界：ID 不存在时不能假装成功。对前端来说，`DELETE /api/errors/999` 删除一个不存在的资源，应该得到 404，这样页面才能给出正确提示。

### Step 6.1：追加更新和删除到 service/error_service.go

在 `service/error_service.go` 末尾追加：

```go
// UpdateError 更新一条错题
func UpdateError(id int, req models.UpdateErrorRequest) error {
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return err
	}

	for i := range errors {
		if errors[i].ID != id {
			continue
		}

		// 只更新前端传了的字段（非 nil 才更新）
		if req.Subject != nil {
			if !SubjectExists(*req.Subject) {
				return fmt.Errorf("无效科目")
			}
			errors[i].Subject = *req.Subject
		}
		if req.Title != nil {
			errors[i].Title = *req.Title
		}
		if req.Question != nil {
			if strings.TrimSpace(*req.Question) == "" {
				return fmt.Errorf("题目不能为空")
			}
			errors[i].Question = *req.Question
		}
		if req.Wrong != nil {
			errors[i].Wrong = *req.Wrong
		}
		if req.Correct != nil {
			errors[i].Correct = *req.Correct
		}
		if req.Reason != nil {
			errors[i].Reason = *req.Reason
		}
		if req.Tags != nil {
			errors[i].Tags = *req.Tags
		}
		if req.ReasonTags != nil {
			errors[i].ReasonTags = *req.ReasonTags
		}

		return store.SaveJSON("errors.json", errors)
	}

	return fmt.Errorf("未找到错题 #%d", id)
}

// DeleteError 删除一条错题
func DeleteError(id int) error {
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return err
	}

	found := false
	remaining := []models.ErrorProblem{}
	for _, item := range errors {
		if item.ID == id {
			found = true
			continue
		}
		remaining = append(remaining, item)
	}

	if !found {
		return fmt.Errorf("未找到错题 #%d", id)
	}

	return store.SaveJSON("errors.json", remaining)
}
```

**这段代码在干什么？**

这两个函数实现了错题的**更新和删除**，都属于 service 层的标准写操作：**加载数据 → 找到目标 → 修改/移除 → 写回文件**。

**`UpdateError` 的"部分更新"设计：**

这是整个函数最值得注意的设计点。`UpdateErrorRequest` 的所有字段都是 `*string` / `*[]string` 指针类型——这不是随意选的，而是为了区分"前端没传这个字段"和"前端传了空值"两种不同的情况：

- 前端只传了 `{"title": "新标题"}`，那么只有 `req.Title` 不是 `nil`，其他字段都是 `nil`。代码中 `if req.Subject != nil` 为 false，直接跳过，科目保持不变。
- 如果字段是普通 `string` 而非 `*string`，那么没传的字段就是 `""`（零值），代码就无法区分"没传"和"传了空字符串"。

这种用指针实现"可选字段"的模式在 Go 的 JSON API 中非常常见，特别是 PATCH/PUT 部分更新场景。

**更新逻辑要点：**
- 用 `for i := range errors` 而非 `for _, item := range`，因为需要**通过索引修改切片中的元素**（`errors[i].Subject = ...`）。如果用 `item`，只是修改了副本，原切片不变。
- `*req.Subject` 中的 `*` 是解引用操作符，取出指针指向的值。
- 如果更新了科目，必须再次校验科目是否存在（`SubjectExists`）。

**`DeleteError` 的"过滤器式"删除：**

不是用 `append(errors[:i], errors[i+1:]...)` 这种切片操作来删除，而是创建一个新的 `remaining` 切片，把不删除的元素逐个 append 进去。这种写法更安全、更易读，不容易出现索引越界的问题。

`found` 标记位确保如果 ID 不存在，能返回明确的错误信息而不是静默成功。

**逐行解释：**

| 代码 | 什么意思 |
|------|---------|
| `for i := range errors` | 用索引遍历（而非 `for _, item := range`），因为要修改切片中的元素 |
| `*req.Subject` | `*` 解引用指针，获取指针指向的 string 值 |
| `fmt.Errorf("未找到错题 #%d", id)` | `%d` 会被替换为 id 的值 |

### Step 6.2：追加 handler

在 `handlers/errors.go` 末尾追加：

```go
import (
	// ... 已有 import 保持不变，只需在 import 块中加一个
	"strconv"
)
```

**这段代码在干什么？**

这一小段是在现有的 `import` 声明块中新增一个包：`strconv`。这是 Go 标准库中的字符串转换包，提供了 `Atoi`（字符串转整数）和 `Itoa`（整数转字符串）两个核心函数。

handler 需要从 URL 路径中提取 ID（如 `/api/errors/3` 中的 `3`），但 `c.Param("id")` 返回的是字符串。要把它传给 service 层的 `UpdateError(id int, ...)`，必须先转成 `int`，这就是 `strconv.Atoi` 的用途。反过来，构造响应消息时需要把整数 ID 拼进字符串，就得用 `strconv.Itoa`。

名称记忆法：**A**scii **to** **I**nteger / **I**nteger **to** **A**scii。

然后在文件末尾追加：

```go
// UpdateError 处理 PUT /api/errors/:id
func UpdateError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "ID格式错误"})
		return
	}

	var req models.UpdateErrorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}

	if err := service.UpdateError(id, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "错题 #" + strconv.Itoa(id) + " 已更新"})
}

// DeleteError 处理 DELETE /api/errors/:id
func DeleteError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "ID格式错误"})
		return
	}

	if err := service.DeleteError(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "错题 #" + strconv.Itoa(id) + " 已删除"})
}
```

**这段代码在干什么？**

这是 handler 层的更新和删除处理函数。它们和 Part 5 的 handler 函数结构一致，但引入了一个新的技术点：**URL 路径参数**。

**路径参数的提取与转换：**

`c.Param("id")` 从 URL `/api/errors/:id` 中提取 `:id` 部分。例如请求 `/api/errors/3`，`c.Param("id")` 返回字符串 `"3"`。但 service 层的 `UpdateError` 和 `DeleteError` 需要 `int` 类型的 ID，所以必须用 `strconv.Atoi` 转换。

**错误处理的两个层次：**

- `strconv.Atoi` 失败（比如 `/api/errors/abc`）→ 返回 400 Bad Request，提示"ID格式错误"。这是"请求本身有问题"，handler 直接拦截，不会传到 service。
- service 返回 error（比如 ID 不存在）→ 同样返回错误响应，但 `DeleteError` 用 404，`UpdateError` 用 400——因为"找不到要更新的资源"和"找不到要删除的资源"在 HTTP 语义上略有不同：删除不存在的资源通常返回 404。

**响应消息的构造：**

`"错题 #" + strconv.Itoa(id) + " 已更新"` 是字符串拼接。Go 中直接用 `+` 连接字符串即可（对于简单场景）。更复杂的可以用 `fmt.Sprintf`，但这里两段拼接 `+` 最简洁。

**逐行解释：**

| 代码 | 什么意思 |
|------|---------|
| `strconv.Atoi(c.Param("id"))` | URL 参数都是字符串，`Atoi`（ASCII to Integer）转成整数 |
| `strconv.Itoa(id)` | 整数转字符串，Itoa = Integer to ASCII |

### Step 6.3：注册路由

在 `main.go` 中加两行：

```go
	r.PUT("/api/errors/:id", handlers.UpdateError)
	r.DELETE("/api/errors/:id", handlers.DeleteError)
```

**这段代码在干什么？**

这两行注册了更新和删除的路由。路径中的 `:id` 是 Gin 的**路径参数语法**——冒号表示这是一个动态段，可以匹配任意值。

例如 `PUT /api/errors/1` 匹配第一个路由，`c.Param("id")` 返回 `"1"`；`DELETE /api/errors/29` 匹配第二个路由，`c.Param("id")` 返回 `"29"`。

这是 RESTful API 的标准写法：用 HTTP 方法（PUT/DELETE）表示操作类型，用路径参数表示资源标识。

### ✅ 验证

```powershell
go run .
```

**curl 验证：**

```powershell
# 1. 更新一条错题的标题
curl -X PUT http://127.0.0.1:8000/api/errors/1 -H "Content-Type: application/json" -d "{\"title\":\"新标题\"}"

# 2. 删除测试错题（ID 按实际调整）
curl -X DELETE http://127.0.0.1:8000/api/errors/29
```

**浏览器验证：** 刷新 `http://127.0.0.1:8000`，进入错题列表：

- ✅ 点击某条错题进入详情 → **编辑功能可用**（修改字段后保存）
- ✅ **删除功能可用**（删一条错题后列表自动更新）

> 🎯 **里程碑 3：错题 CRUD 全部完成！** 前端能做完整的增删改查了。

---

## Part 7：复习功能 + 标签

### 目标

实现复习标记（艾宾浩斯间隔）和标签聚合。

**这一 Part 给错题增加两个"派生能力"。**

前面的 CRUD 只是管理错题本身；这里开始让数据产生复习计划和标签视图：

| 能力 | 依赖字段 | 前端用它做什么 |
|------|----------|----------------|
| 标记复习 | `ReviewCount`、`LastReview`、`ReviewStage`、`NextReview` | 点击"已复习"后刷新复习次数和下次复习日期 |
| 标签聚合 | `Tags`、`ReasonTags` | 生成筛选项、标签列表、错因分类 |

这两个功能都不应该放在 handler 里。handler 只知道 HTTP 请求；"复习一次后下次什么时候复习"、"标签怎么去重排序"是业务规则，应该放在 service 层。

### Step 7.1：追加到 service/error_service.go

在文件末尾追加：

```go
// 艾宾浩斯复习间隔（天数）
var reviewIntervals = []int{0, 1, 2, 4, 7, 15, 30, 60}

// ReviewError 标记复习一条错题
func ReviewError(id int) (models.ErrorProblem, error) {
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return models.ErrorProblem{}, err
	}

	for i := range errors {
		if errors[i].ID != id {
			continue
		}

		nowText := time.Now().Format("2006-01-02 15:04:05")
		errors[i].ReviewCount++
		errors[i].LastReview = &nowText // &取地址，因为 LastReview 是 *string
		errors[i].ReviewStage = errors[i].ReviewCount
		errors[i].NextReview = nextReviewDate(errors[i].ReviewCount)

		if err := store.SaveJSON("errors.json", errors); err != nil {
			return models.ErrorProblem{}, err
		}
		return errors[i], nil
	}

	return models.ErrorProblem{}, fmt.Errorf("未找到错题 #%d", id)
}

func nextReviewDate(reviewCount int) string {
	index := reviewCount
	if index < 0 {
		index = 0
	}
	if index >= len(reviewIntervals) {
		index = len(reviewIntervals) - 1
	}
	return time.Now().AddDate(0, 0, reviewIntervals[index]).Format("2006-01-02")
}
```

**这段代码在干什么？**

`ReviewError` 是"艾宾浩斯间隔复习"的核心 service 层函数。它的职责很明确：**找到指定 ID 的错题，把它的复习状态推进一个阶段，然后保存回 JSON 文件**。

这里涉及几个 Go 的实用知识点：

- **包级变量（package-level variable）**：`reviewIntervals` 定义在所有函数外面，整个 service 包共享。它是一个 `[]int` 切片，存储艾宾浩斯复习间隔天数——第 1 次复习当天（0天），第 2 次隔 1 天，第 3 次隔 2 天……最多到第 8 次及以后固定 60 天。用包级变量比在函数里硬编码数字更灵活——以后想调整间隔只需改这一个地方。

- **指针字段的赋值**：`errors[i].LastReview = &nowText`。为什么是 `&nowText` 而不是直接 `nowText`？因为 `LastReview` 的类型是 `*string`（字符串指针）。Go 里不能把 `string` 直接赋给 `*string`——必须用 `&` 取地址，告诉编译器"这个字段存的是 `nowText` 所在的内存地址"。如果 `LastReview` 为 `nil`，JSON 序列化时这个字段输出 `null`；赋过值后输出实际时间字符串。

- **用索引遍历切片**：`for i := range errors` 只拿到索引 `i`，没有拿值。这是因为后面需要 `errors[i].ReviewCount++` 直接修改切片里的原始数据——如果用 `for _, item := range errors`，`item` 只是一个**副本**，改了它原始切片不变。

- **`continue` 扁平化控制流**：`if errors[i].ID != id { continue }` 是 Go 里常见的"过滤"写法——不满足条件的跳过，主逻辑不要嵌套在 `if` 块里面。这样代码从左到右一条线读下来，而不是 if 套 if。

- **返回值惯例**：`(models.ErrorProblem, error)` 返回两个值——成功时第一个有意义、第二个是 `nil`；失败时第一个是零值（`models.ErrorProblem{}` 所有字段默认值）、第二个是具体错误。这是 Go 标准库一贯的多返回值风格。

`nextReviewDate` 是一个**辅助函数（helper）**，只被 `ReviewError` 调用。注意它做了边界保护：`index < 0` 设为 0，`index >= len(reviewIntervals)` 设为最后一个位置——防止数组越界（panic: index out of range）。`reviewCount` 可能因为数据异常变成一个超大或负数，加了保护程序就不会崩。`time.Now().AddDate(0, 0, days)` 在当前时间上加 N 天，三个参数依次是年、月、日——不用自己算跨月、跨年、闰年，Go 标准库都处理好了。

**逐行解释：**

| 代码 | 什么意思 |
|------|---------|
| `reviewIntervals` | 复习间隔：第 1 次当天复习，第 2 次隔 1 天，第 3 次隔 2 天…第 8 次及以后隔 60 天 |
| `errors[i].LastReview = &nowText` | `&` 取变量的地址，把 `*string` 指向 `nowText` |
| `time.Now().AddDate(0, 0, days)` | 在当前时间上加 N 天（三个参数：年、月、日） |

### Step 7.2：添加标签聚合

在 `service/error_service.go` 末尾追加：

```go
import "sort" // 加到文件开头的 import 块中
```

**这段代码在干什么？**

很简单的一行——在 `service/error_service.go` 文件开头的 `import` 块里加上 `"sort"`。`sort` 是 Go 标准库里的排序包，后面 `GetAllTags` 函数用 `sort.Strings(tags)` 对标签按字母排序，就需要这个导入。Go 规定：凡是使用的包都必须 import，没用的包 import 了会编译报错（`imported and not used`）。

```go
// GetAllTags 获取所有标签（从错题中提取，去重排序）
func GetAllTags() ([]string, error) {
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	for _, item := range errors {
		for _, tag := range item.Tags {
			seen[tag] = true
		}
		for _, tag := range item.ReasonTags {
			seen[tag] = true
		}
	}

	tags := []string{}
	for tag := range seen {
		tags = append(tags, tag)
	}
	sort.Strings(tags) // 排序，前端显示更稳定
	return tags, nil
}
```

**这段代码在干什么？**

`GetAllTags` 是一个 service 层函数，负责从所有错题中提取标签并去重、排序。前端用它来填充标签筛选下拉框。

核心思路是**两个来源，一个去重**：标签可能出现在 `Tags` 字段（题目标签，如"三角函数"），也可能出现在 `ReasonTags` 字段（原因标签，如"计算失误"）。用 `map[string]bool{}` 做去重——map 的键是唯一的，所以同一个标签无论出现几次，`seen[tag] = true` 只会存一次。

实现步骤：

1. **用 map 收集**：双重 `for range` 遍历所有错题，逐一收集 `Tags` 和 `ReasonTags`。`seen[tag] = true` 的值 `true` 本身没意义，只是占位——我们只需要键去重。
2. **map 转切片**：`for tag := range seen` 遍历 map 的所有键，逐个 `append` 到切片 `tags`。这一步之后标签已经去重了，但顺序不确定（map 的遍历顺序是随机的）。
3. **排序**：`sort.Strings(tags)` 按字母升序排列。排序的目的是让前端每次请求拿到的标签顺序一致，下拉框不会跳来跳去。

**返回值含义：** `([]string, error)`——返回去重排序后的标签列表和可能的错误。如果 JSON 文件读失败就返回 `nil, err`。注意 `return tags, nil` 里 `tags` 是 `[]string{}`（用字面量初始化的），不会是 `nil`，所以前端收到的是 `[]` 而不是 `null`——这和 API 兼容性约束"空切片用 `[]string{}` 而非 `nil`"一致。

**逐行解释：**

| 代码 | 什么意思 |
|------|---------|
| `map[string]bool{}` | 用 map 做去重，键是标签名，值恒为 true |
| `for tag := range seen` | 遍历 map 的所有键 |
| `sort.Strings(tags)` | 按字母排序 |

### Step 7.3：追加 handler

在 `handlers/errors.go` 末尾追加：

```go
// ReviewError 处理 PUT /api/errors/:id/review
func ReviewError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "ID格式错误"})
		return
	}

	item, err := service.ReviewError(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "错题 #" + strconv.Itoa(id) + " 已标记复习",
		"next_review":  item.NextReview,
		"review_count": item.ReviewCount,
	})
}

// GetTags 处理 GET /api/tags
func GetTags(c *gin.Context) {
	tags, err := service.GetAllTags()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}
```

**这段代码在干什么？**

这是 handler 层（HTTP 层）的两个新函数，负责接收 HTTP 请求、调用 service 层、返回 JSON 响应。

**`ReviewError`（PUT /api/errors/:id/review）：**
1. `c.Param("id")` 从 URL 路径中提取 `:id` 部分——Gin 框架自动解析路由参数。
2. `strconv.Atoi()` 把字符串转成 `int`。注意 Go 里没有异常机制，`Atoi` 通过返回的 `err` 告诉你转换是否成功。如果 ID 不是数字（比如 `/api/errors/abc/review`），返回 400。
3. 调 `service.ReviewError(id)` 执行业务逻辑。
4. 响应格式不是返回整个错题对象，而是返回 `message`、`next_review`、`review_count` 三个字段——这是前端需要的精简格式，减少传输数据量。

**`GetTags`（GET /api/tags）：**
1. 不需要任何参数，直接调 `service.GetAllTags()`。
2. 响应用 `{"tags": tags}` 包裹——和列表用 `{"errors": [...]}` 包裹一样，前端期望的是一个对象而非裸数组。如果裸返回 `["数学","英语"]`，前端 `response.data.tags` 会拿不到数据。

两个函数都遵循 handler 层的标准模式：**解析请求 → 调 service → 错误处理 → 返回 JSON**。每个 handler 都只做 HTTP 相关的事情，业务逻辑全在 service 层——这就是三层架构的分工。

### Step 7.4：注册路由

```go
	r.PUT("/api/errors/:id/review", handlers.ReviewError)
	r.GET("/api/tags", handlers.GetTags)
```

**这段代码在干什么？**

两行路由注册，放在 `main.go` 或路由设置函数里。`r.PUT` 注册一个 PUT 方法的路由，`r.GET` 注册 GET 方法的路由。Gin 的路由组 `r` 已经在前面初始化好了，这里只是追加两条新规则。

**路径参数** `:id` 是 Gin 的动态路由语法——冒号后面的 `id` 会变成参数名，handler 里用 `c.Param("id")` 就能取到。比如请求 `PUT /api/errors/5/review`，`c.Param("id")` 返回 `"5"`。

### ✅ 验证

```powershell
go run .
```

**curl 验证：**

```powershell
# 1. 复习错题 #1
curl -X PUT http://127.0.0.1:8000/api/errors/1/review
# 返回 next_review 日期和 review_count

# 2. 获取所有标签
curl http://127.0.0.1:8000/api/tags
```

**浏览器验证：** 刷新 `http://127.0.0.1:8000`

- ✅ 错题详情页：点击"复习"按钮 → 复习次数增加，下次复习日期更新
- ✅ 标签筛选功能可用（标签下拉框有数据了）
- ✅ 错题卡片上的标签正常显示

> 🎯 **里程碑 4：复习和标签通了！** 核心的艾宾浩斯复习功能可以用了。

---

### Step 7.5：当前完整文件版，防止 import 写乱

前面为了讲清楚流程，把 `service/error_service.go` 和 `handlers/errors.go` 分成几步写。真正敲代码时，如果你担心 import 漏掉，可以直接用下面两个完整版本覆盖文件。

#### 完整版 `service/error_service.go`

```go
package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"study-tracker-go/models"
	"study-tracker-go/store"
)

var reviewIntervals = []int{0, 1, 2, 4, 7, 15, 30, 60}

func CreateError(req models.AddErrorRequest) (models.ErrorProblem, error) {
	req.Subject = strings.TrimSpace(req.Subject)
	req.Question = strings.TrimSpace(req.Question)
	if !SubjectExists(req.Subject) {
		return models.ErrorProblem{}, fmt.Errorf("无效科目")
	}
	if req.Question == "" {
		return models.ErrorProblem{}, fmt.Errorf("题目不能为空")
	}
	if req.Wrong == "" {
		req.Wrong = "未记录"
	}
	if req.Correct == "" {
		req.Correct = "未记录"
	}
	if req.Reason == "" {
		req.Reason = "未记录"
	}
	if req.Title == "" {
		req.Title = firstRunes(req.Question, 40)
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}
	if req.ReasonTags == nil {
		req.ReasonTags = []string{}
	}

	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return models.ErrorProblem{}, err
	}
	nextID := 1
	for _, item := range errors {
		if item.ID >= nextID {
			nextID = item.ID + 1
		}
	}

	now := time.Now()
	item := models.ErrorProblem{
		ID:          nextID,
		Subject:     req.Subject,
		Title:       req.Title,
		Question:    req.Question,
		Wrong:       req.Wrong,
		Correct:     req.Correct,
		Reason:      req.Reason,
		Tags:        req.Tags,
		ReasonTags:  req.ReasonTags,
		Created:     now.Format("2006-01-02 15:04:05"),
		ReviewCount: 0,
		LastReview:  nil,
		ReviewStage: 0,
		NextReview:  now.Format("2006-01-02"),
	}

	errors = append(errors, item)
	if err := store.SaveJSON("errors.json", errors); err != nil {
		return models.ErrorProblem{}, err
	}
	return item, nil
}

func GetAllErrors(subject, keyword, tag, reasonTag string) ([]models.ErrorProblem, error) {
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return nil, err
	}
	if errors == nil {
		return []models.ErrorProblem{}, nil
	}
	result := []models.ErrorProblem{}
	for _, item := range errors {
		normalizeReviewFields(&item)
		if subject != "" && subject != "全部" && item.Subject != subject {
			continue
		}
		if keyword != "" && !matchesKeyword(item, keyword) {
			continue
		}
		if tag != "" && !listContainsFold(item.Tags, tag) {
			continue
		}
		if reasonTag != "" && !listContainsFold(item.ReasonTags, reasonTag) {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func UpdateError(id int, req models.UpdateErrorRequest) error {
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return err
	}
	for i := range errors {
		if errors[i].ID != id {
			continue
		}
		if req.Subject != nil {
			if !SubjectExists(*req.Subject) {
				return fmt.Errorf("无效科目")
			}
			errors[i].Subject = *req.Subject
		}
		if req.Title != nil {
			errors[i].Title = *req.Title
		}
		if req.Question != nil {
			if strings.TrimSpace(*req.Question) == "" {
				return fmt.Errorf("题目不能为空")
			}
			errors[i].Question = *req.Question
		}
		if req.Wrong != nil {
			errors[i].Wrong = *req.Wrong
		}
		if req.Correct != nil {
			errors[i].Correct = *req.Correct
		}
		if req.Reason != nil {
			errors[i].Reason = *req.Reason
		}
		if req.Tags != nil {
			errors[i].Tags = *req.Tags
		}
		if req.ReasonTags != nil {
			errors[i].ReasonTags = *req.ReasonTags
		}
		return store.SaveJSON("errors.json", errors)
	}
	return fmt.Errorf("未找到错题 #%d", id)
}

func DeleteError(id int) error {
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return err
	}
	found := false
	remaining := []models.ErrorProblem{}
	for _, item := range errors {
		if item.ID == id {
			found = true
			continue
		}
		remaining = append(remaining, item)
	}
	if !found {
		return fmt.Errorf("未找到错题 #%d", id)
	}
	return store.SaveJSON("errors.json", remaining)
}

func ReviewError(id int) (models.ErrorProblem, error) {
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return models.ErrorProblem{}, err
	}
	for i := range errors {
		if errors[i].ID != id {
			continue
		}
		nowText := time.Now().Format("2006-01-02 15:04:05")
		errors[i].ReviewCount++
		errors[i].LastReview = &nowText
		errors[i].ReviewStage = errors[i].ReviewCount
		errors[i].NextReview = nextReviewDate(errors[i].ReviewCount)
		if err := store.SaveJSON("errors.json", errors); err != nil {
			return models.ErrorProblem{}, err
		}
		return errors[i], nil
	}
	return models.ErrorProblem{}, fmt.Errorf("未找到错题 #%d", id)
}

func GetAllTags() ([]string, error) {
	var errors []models.ErrorProblem
	if err := store.LoadJSON("errors.json", &errors); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, item := range errors {
		for _, tag := range item.Tags {
			seen[tag] = true
		}
		for _, tag := range item.ReasonTags {
			seen[tag] = true
		}
	}
	tags := []string{}
	for tag := range seen {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags, nil
}

func firstRunes(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max])
}

func normalizeReviewFields(item *models.ErrorProblem) {
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if item.ReasonTags == nil {
		item.ReasonTags = []string{}
	}
	if item.NextReview == "" {
		if len(item.Created) >= 10 {
			item.NextReview = item.Created[:10]
		} else {
			item.NextReview = time.Now().Format("2006-01-02")
		}
	}
}

func nextReviewDate(reviewCount int) string {
	index := reviewCount
	if index < 0 {
		index = 0
	}
	if index >= len(reviewIntervals) {
		index = len(reviewIntervals) - 1
	}
	return time.Now().AddDate(0, 0, reviewIntervals[index]).Format("2006-01-02")
}

func matchesKeyword(item models.ErrorProblem, keyword string) bool {
	keyword = strings.ToLower(keyword)
	if strings.Contains(strings.ToLower(item.Question), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(item.Title), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(item.Reason), keyword) {
		return true
	}
	return listContainsFold(item.Tags, keyword) || listContainsFold(item.ReasonTags, keyword)
}

func listContainsFold(list []string, keyword string) bool {
	keyword = strings.ToLower(keyword)
	for _, item := range list {
		if strings.Contains(strings.ToLower(item), keyword) {
			return true
		}
	}
	return false
}
```

**这段代码在干什么？**

这是 `service/error_service.go` 的**完整版**——把之前分散在各步骤里写的函数整合到一份文件里，方便你一次性覆盖。包含的辅助函数有：

- `firstRunes`：截取前 N 个字符（按 rune 而非字节，支持中文），用于自动生成标题。
- `normalizeReviewFields`：保证 `Tags`、`ReasonTags` 不为 `nil`，`NextReview` 不为空——防止前端拿到 `null`。
- `nextReviewDate`：根据复习次数计算下次复习日期。
- `matchesKeyword` / `listContainsFold`：不区分大小写的模糊搜索。

如果你之前逐步追加时 import 写乱了，直接用这份完整版覆盖文件就能编译通过。

#### 完整版 `handlers/errors.go`

```go
package handlers

import (
	"net/http"
	"strconv"

	"study-tracker-go/models"
	"study-tracker-go/service"

	"github.com/gin-gonic/gin"
)

func CreateError(c *gin.Context) {
	var req models.AddErrorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}
	item, err := service.CreateError(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": item.ID, "message": "添加成功"})
}

func GetErrors(c *gin.Context) {
	errors, err := service.GetAllErrors(
		c.Query("subject"),
		c.Query("keyword"),
		c.Query("tag"),
		c.Query("reason_tag"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"errors": errors, "total": len(errors)})
}

func UpdateError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "ID格式错误"})
		return
	}
	var req models.UpdateErrorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}
	if err := service.UpdateError(id, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "错题 #" + strconv.Itoa(id) + " 已更新"})
}

func DeleteError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "ID格式错误"})
		return
	}
	if err := service.DeleteError(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "错题 #" + strconv.Itoa(id) + " 已删除"})
}

func ReviewError(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "ID格式错误"})
		return
	}
	item, err := service.ReviewError(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":      "错题 #" + strconv.Itoa(id) + " 已标记复习",
		"next_review":  item.NextReview,
		"review_count": item.ReviewCount,
	})
}

func GetTags(c *gin.Context) {
	tags, err := service.GetAllTags()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}
```

**这段代码在干什么？**

这是 `handlers/errors.go` 的**完整版**，整合了所有 CRUD handler 加上复习和标签 handler。同样是一份"可直接覆盖"的参考版本。所有 handler 都遵循同一个流程：读参数 → 调 service → 处理错误 → 返回 JSON。如果之前手写时有遗漏，直接用这份覆盖即可。

## Part 8：每日推送

### 目标

实现每日推送接口，返回今日到期错题、逾期数量、知识点提示。

**每日推送不是一个新数据表，而是"读取现有错题后临时计算出来的视图"。**

它依赖前面几个功能的字段约定：

| 字段/数据 | 来自哪里 | 每日推送怎么用 |
|-----------|----------|----------------|
| `NextReview` | 创建错题、复习错题、旧数据补全 | 判断今天是否到期、是否逾期 |
| `ReviewCount` | 复习功能 | 统计已经复习过多少题 |
| `subjects.json` | 科目功能 | 决定要给哪些科目推知识点 |
| `knowledge.json` | 可选知识库文件 | 有就用用户自定义，没有就用内置默认知识点 |

所以这一步最重要的是容错：即使某条旧错题缺 `next_review`，即使 `knowledge.json` 不存在，首页也应该正常显示，而不是空白或报错。

### Step 8.1：写 service/daily_service.go

创建 `service/daily_service.go`：

```go
package service

import (
	"math/rand"
	"sort"
	"time"

	"study-tracker-go/models"
	"study-tracker-go/store"
)

// 知识点库（硬编码默认值，如果 knowledge.json 不存在就用这个）
var defaultKnowledge = map[string][]string{
	"数学": {"等价无穷小：x→0时，sinx~x，tanx~x", "洛必达法则适合 0/0 或 ∞/∞ 型"},
	"英语": {"ambiguous 表示模棱两可的", "fundamental 表示根本的、基础的"},
	"物理": {"牛顿第二定律：F = ma", "欧姆定律：I = U / R"},
	"化学": {"阿伏伽德罗常数：6.02 × 10²³", "勒夏特列原理：平衡会减弱改变"},
	"生物": {"ATP 是细胞的能量通货", "DNA 碱基配对：A-T，C-G"},
	"语文": {"常见修辞：比喻、拟人、排比、对偶", "论证方法：举例、道理、对比、比喻"},
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// GetDailyPush 生成每日推送数据
func GetDailyPush() (models.DailyPushResult, error) {
	errors, err := GetAllErrors("", "", "", "")
	if err != nil {
		return models.DailyPushResult{}, err
	}
	subjects, err := GetAllSubjects()
	if err != nil {
		return models.DailyPushResult{}, err
	}

	today := time.Now().Format("2006-01-02")
	due := []models.ErrorProblem{}
	overdue := 0
	reviewed := 0

	for _, item := range errors {
		if item.ReviewCount > 0 {
			reviewed++
		}
		next := item.NextReview
		if next == "" {
			next = today
		}
		if next <= today {
			due = append(due, item)
		}
		if next < today {
			overdue++
		}
	}

	// 按到期日排序
	sort.Slice(due, func(i, j int) bool {
		if due[i].NextReview == due[j].NextReview {
			return due[i].ID < due[j].ID
		}
		return due[i].NextReview < due[j].NextReview
	})

	// 每个科目随机选一条知识点
	knowledgeBase := getKnowledgeBase()
	knowledge := map[string]string{}
	for _, subject := range subjects {
		tips := knowledgeBase[subject]
		if len(tips) > 0 {
			knowledge[subject] = tips[rand.Intn(len(tips))]
		}
	}

	// 建议文案
	advice := "今天没有到期错题，可以新增整理或轻量回看近期内容"
	if len(due) > 0 {
		advice = "今天有到期错题，建议先清空复习队列"
	}
	if overdue > 0 {
		advice = "今天有逾期错题，建议优先处理最早到期的题目"
	}

	return models.DailyPushResult{
		Date:         today,
		TotalErrors:  len(errors),
		Reviewed:     reviewed,
		DueCount:     len(due),
		OverdueCount: overdue,
		Knowledge:    knowledge,
		WeakErrors:   due,
		Advice:       advice,
	}, nil
}

func getKnowledgeBase() map[string][]string {
	var kb map[string][]string
	if err := store.LoadJSON("knowledge.json", &kb); err != nil {
		return defaultKnowledge
	}
	if kb == nil {
		return defaultKnowledge
	}
	return kb
}
```

**这段代码在干什么？**

`GetDailyPush` 是每日推送的**核心业务逻辑**，属于 service 层。它的职责是：把错题数据、复习进度、知识点库整合成一份"今日简报"，前端仪表盘直接渲染这份数据。

**整体流程：**

1. **拉取所有错题和科目** — 调用已有的 `GetAllErrors` 和 `GetAllSubjects`，复用现有 service 函数，不重复造轮子。
2. **计算到期/逾期** — 遍历每条错题的 `NextReview` 字段，和今天日期做字符串比较。Go 的 `"2006-01-02"` 格式有个巧妙之处：日期字符串可以直接用 `<=`、`<` 比较，因为它是从大到小排列的（年-月-日），字典序等于时间序。
3. **排序到期错题** — `sort.Slice` 是 Go 标准库提供的切片排序函数，按 `NextReview` 升序排列，同一天按 ID 升序，让前端展示有规律。
4. **随机抽取知识点** — 从 `knowledge.json`（或内置的默认知识库）中，每个科目随机选一条小贴士。用 `rand.Intn` 实现随机，`init()` 函数里用时间戳设置了随机种子，避免每次重启都抽到同一条。
5. **生成建议文案** — 根据是否有到期/逾期错题，用简单的 if-else 逻辑生成一句话建议。这是业务层的"人性化"输出，前端不用再判断。

**Go 语言知识点：**

| 概念 | 说明 |
|------|------|
| `var defaultKnowledge = map[string][]string{...}` | **包级变量**，在 `init()` 之前初始化。这是一个硬编码的默认知识点库，当 `knowledge.json` 不存在时作为兜底。 |
| `init()` 函数 | Go 的特殊函数，包被导入时自动执行，比 `main()` 更早。这里用来设置随机种子。 |
| `rand.Seed(time.Now().UnixNano())` | 用当前时间的纳秒数初始化随机数生成器，确保每次运行结果不同。注意：Go 1.20+ 已经自动初始化，但显式调用也无害。 |
| `time.Now().Format("2006-01-02")` | Go 语言独一无二的日期格式化方式——用固定参考时间 `2006-01-02 15:04:05` 来表达格式。`"2006-01-02"` 就是"年-月-日"。 |
| `sort.Slice(due, func(i, j int) bool {...})` | 对切片原地排序，闭包里写比较逻辑。i 和 j 是元素下标，不是元素本身。 |
| `knowledge[subject] = tips[rand.Intn(len(tips))]` | `rand.Intn(n)` 返回 `[0, n)` 范围内的随机整数，正好做随机索引。 |
| `store.LoadJSON("knowledge.json", &kb)` | 尝试从文件加载知识点。如果文件不存在，`LoadJSON` 静默返回（这是 store 层的设计），kb 保持 nil，然后 fallback 到 `defaultKnowledge`。 |

**为什么把 `getKnowledgeBase` 抽成单独函数？** 因为加载知识库是独立逻辑，且需要"加载失败就 fallback"的容错处理。拆出去让 `GetDailyPush` 主函数保持简洁，也方便单独测试。

**几个容易误解的判断：**

| 代码 | 为什么这样写 |
|------|--------------|
| `if next == "" { next = today }` | 这是第二道兜底。正常情况下 `GetAllErrors` 已经用 `normalizeReviewFields` 补过 `NextReview`，但每日推送是首页关键功能，再兜一次可以防止异常数据让首页崩掉。 |
| `if next <= today { due = append(due, item) }` | `due` 表示"今天需要处理的题"，包括今天到期和已经逾期。日期格式是 `YYYY-MM-DD`，字符串比较顺序等于日期顺序。 |
| `if next < today { overdue++ }` | `overdue` 只统计已经逾期的题，不包含今天刚到期的题。这样前端可以区分"今天该复习"和"已经拖延"。 |
| 同一天按 `ID` 排序 | 两题同一天到期时，用 ID 保持稳定顺序，刷新页面不会乱跳。 |
| `defaultKnowledge` | 第一次运行或用户没配置知识库时，首页仍然有内容。默认值不是核心数据，只是体验兜底。 |

**为什么能直接比较日期字符串？**

只有 `YYYY-MM-DD` 这种从大到小排列的格式才可以这么比。比如：

```text
"2026-06-02" < "2026-06-19"  // true
"2026-07-01" > "2026-06-19"  // true
```

如果写成 `"06/19/2026"` 或 `"2026-6-9"` 就不安全了，所以前面所有日期都统一用 `time.Now().Format("2006-01-02")`。

### Step 8.2：写 handlers/daily.go

创建 `handlers/daily.go`：

```go
package handlers

import (
	"net/http"

	"study-tracker-go/service"

	"github.com/gin-gonic/gin"
)

// GetDailyPush 处理 GET /api/daily-push
func GetDailyPush(c *gin.Context) {
	result, err := service.GetDailyPush()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
```

**这段代码在干什么？**

`GetDailyPush` handler 是 Part 8 中最短的一个 handler——只有 6 行有效代码。这恰好展示了 **handler 层的标准写法**：handler 不包含任何业务逻辑，只是一个"接线员"，负责三件事：

1. **调用 service** — `service.GetDailyPush()` 拿到结果。
2. **处理错误** — 返回 `{"detail": ...}` 格式的错误 JSON（保持和前端约定的 API 兼容性）。
3. **返回成功响应** — 直接把 service 返回的 `models.DailyPushResult` 结构体传给 `c.JSON()`。Gin 会自动把 Go 结构体序列化为 JSON，结构体字段的 `json:"..."` 标签决定输出的 key 名。

**为什么 handler 这么薄？** 这是三层架构的核心思想：**handler 不碰数据、不写逻辑、不做判断**。所有业务逻辑都在 service 里，handler 只负责 HTTP 层面的"收"和"发"。这样做的好处是：如果以后想加一个 CLI 版本的每日推送命令，直接调 `service.GetDailyPush()` 就行，完全不依赖 HTTP。

**Go 语言知识点：**

| 概念 | 说明 |
|------|------|
| `c *gin.Context` | Gin 框架的上下文对象，封装了 `http.ResponseWriter` 和 `*http.Request`，提供了方便的 JSON 序列化方法。每个 HTTP 请求都会创建一个新的 `*gin.Context`。 |
| `c.JSON(statusCode, obj)` | 设置 `Content-Type: application/json`，把 obj 序列化为 JSON 写入响应体，并设置 HTTP 状态码。 |

### Step 8.3：注册路由

```go
	r.GET("/api/daily-push", handlers.GetDailyPush)
```

**这段代码在干什么？**

这是一行路由注册代码。在 Gin 框架中，`r.GET(path, handler)` 的意思是把一个 HTTP GET 请求路径绑定到对应的处理函数上。

当浏览器（或前端）访问 `GET /api/daily-push` 时，Gin 会自动调用 `handlers.GetDailyPush` 函数来处理这个请求。路由注册通常放在 `main.go` 的 `main()` 函数中；如果你后来把路由整理成单独函数，也要保证这些注册在 `r.Run()` 之前完成。

**为什么只有一行？** 因为 service 和 handler 已经写好了，路由注册就是把 URL 和处理函数"连接"起来，是最薄的一层。

### ✅ 验证

```powershell
go run .
```

**curl 验证：**

```powershell
curl http://127.0.0.1:8000/api/daily-push
```

**浏览器验证：** 刷新 `http://127.0.0.1:8000`

- ✅ **首页/仪表盘**：每日推送数据正常显示（到期错题数、逾期数、每日知识点、建议文案）

> 🎯 **里程碑 5：首页仪表盘通了！** 这是前端最核心的页面。

---

## Part 9：设置接口

### 目标

实现 Token 和用户名的读写。数据存在 `config.json`。

**设置接口的重点是安全和兼容。**

Token 属于敏感信息，前端只需要知道"有没有配置"和"大概是哪一个 token"，不应该拿到完整明文。用户名则是普通展示信息，可以直接返回。两类设置放在同一个 `config.json`，但接口分开，是为了贴合前端设置页的两个独立区域：Token 区域可以单独保存/清空，用户名区域可以单独保存。

这里还有一个细节：保存 token 时，空字符串表示"不修改旧 token"，不是"清空 token"。真正清空要走 `DELETE /api/settings/token`。这样做可以避免前端表单没填 token 时误把旧 token 覆盖成空。

### Step 9.1：写 service/settings_service.go

创建 `service/settings_service.go`：

```go
package service

import (
	"strings"

	"study-tracker-go/models"
	"study-tracker-go/store"
)

// GetTokenInfo 获取 Token 信息（已脱敏）
func GetTokenInfo() (masked string, configured bool, username string, err error) {
	config, err := loadConfig()
	if err != nil {
		return "", false, "", err
	}

	token := strings.TrimSpace(config.MineruToken)
	if token != "" {
		// 脱敏显示：前8位 + *** + 后4位
		if len(token) > 12 {
			masked = token[:8] + "***" + token[len(token)-4:]
		} else {
			masked = "***"
		}
	}
	return masked, token != "", config.Username, nil
}

// SetToken 设置 Token
func SetToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	config, err := loadConfig()
	if err != nil {
		return err
	}
	config.MineruToken = token
	return store.SaveJSON("config.json", config)
}

// ClearToken 清除 Token
func ClearToken() error {
	config, err := loadConfig()
	if err != nil {
		return err
	}
	config.MineruToken = ""
	return store.SaveJSON("config.json", config)
}

// SetUsername 设置用户名
func SetUsername(name string) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}
	config.Username = strings.TrimSpace(name)
	return store.SaveJSON("config.json", config)
}

func loadConfig() (models.Config, error) {
	var config models.Config
	if err := store.LoadJSON("config.json", &config); err != nil {
		return models.Config{}, err
	}
	return config, nil
}
```

**这段代码在干什么？**

这个文件实现了**设置功能的 service 层**，负责 Token 和用户名的读写。数据存储在 `config.json` 中，通过 store 层的 `LoadJSON` / `SaveJSON` 进行持久化。

**各个函数的职责：**

| 函数 | 职责 | 关键逻辑 |
|------|------|----------|
| `GetTokenInfo()` | 获取 Token 信息（脱敏后） | 一次返回三个值：脱敏串、是否已配置、用户名 |
| `SetToken(token)` | 设置 Token | 去空格 → 空就跳过 → 写入 config.json |
| `ClearToken()` | 清除 Token | 把字段置空字符串后保存 |
| `SetUsername(name)` | 设置用户名 | 去空格后保存 |
| `loadConfig()` | 加载配置（私有函数） | 所有函数共用，避免重复写加载逻辑 |

**Go 语言知识点：**

| 概念 | 说明 |
|------|------|
| **多返回值** | `GetTokenInfo` 返回 `(masked string, configured bool, username string, err error)`——四个值！Go 的多返回值让"数据 + 状态 + 错误"一次传回，不需要包装成结构体。 |
| `loadConfig` 小写开头 | Go 用首字母大小写控制可见性。小写 `loadConfig` 是包内私有函数，外部包（如 handler）不能直接调用，必须通过导出的 `GetTokenInfo`、`SetToken` 等接口访问。这就是封装。 |
| Token 脱敏 | 长度 > 12 时显示"前8位 + \*\*\* + 后4位"，短 Token 直接显示 `***`。这是一个安全实践：即使别人看到页面，也不会泄露完整 Token。 |
| 空 Token 不保存 | `SetToken` 在 token 为空时直接 `return nil`，避免把空字符串写入配置文件。这是一个防御性设计。 |

**为什么每个函数都要调 `loadConfig()` 而不是缓存？** 因为配置文件可能被外部修改（手动编辑、其他进程写入），每次读写都重新加载能保证数据一致性。代价很小（JSON 文件通常只有几十字节），但避免了"缓存失效"的各种边界情况。

### Step 9.2：写 handlers/settings.go

创建 `handlers/settings.go`：

```go
package handlers

import (
	"net/http"

	"study-tracker-go/service"

	"github.com/gin-gonic/gin"
)

func GetToken(c *gin.Context) {
	token, configured, username, err := service.GetTokenInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"configured": configured,
		"username":   username,
	})
}

func SetToken(c *gin.Context) {
	var body struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}
	if err := service.SetToken(body.Token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Token saved"})
}

func DeleteToken(c *gin.Context) {
	if err := service.ClearToken(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Token cleared"})
}

func SetUsername(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请求格式错误"})
		return
	}
	if err := service.SetUsername(body.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Username saved"})
}
```

**这段代码在干什么？**

这是设置功能的 **handler 层**，包含四个 HTTP 处理函数。每个 handler 都遵循相同的模式：**解析请求 → 调用 service → 返回 JSON**。

**四个 handler 一览：**

| Handler | HTTP 方法 | 功能 | 特点 |
|---------|-----------|------|------|
| `GetToken` | GET | 查看 Token 状态 | 返回脱敏 Token + 用户名，不暴露原始 Token |
| `SetToken` | PUT | 设置 Token | 用匿名结构体 `struct { Token string }` 接收 JSON body |
| `DeleteToken` | DELETE | 清除 Token | 无请求体，直接调 service |
| `SetUsername` | PUT | 设置用户名 | 同样用匿名结构体接收 `{"name": "..."}` |

**Go 语言知识点：**

| 概念 | 说明 |
|------|------|
| **匿名结构体** | `var body struct { Token string \`json:"token"\` }` —— 在函数内部临时定义一个结构体类型，只在这个函数里用。不需要在 models 包里定义，减少了跨文件的类型污染。 |
| **RESTful 风格** | 同一个路径 `/api/settings/token`，用 GET/PUT/DELETE 三种 HTTP 方法区分操作。这是 RESTful API 的标准设计：URL 代表资源，HTTP 方法代表操作。 |
| `c.ShouldBindJSON(&body)` | Gin 的方法，读取请求体的 JSON 并反序列化到结构体。失败时自动返回 400，但这里还是手动检查了 `err` 来统一错误格式。 |

**注意：** 这里没有用 `GET /api/settings` 返回全部设置，而是拆成了 token 和 username 两个独立端点。原因是前端设置页面有两个独立的表单区域，分别加载自己的数据，分开请求更灵活。

### Step 9.3：注册路由

```go
	r.GET("/api/settings/token", handlers.GetToken)
	r.PUT("/api/settings/token", handlers.SetToken)
	r.DELETE("/api/settings/token", handlers.DeleteToken)
	r.PUT("/api/settings/username", handlers.SetUsername)
```

**这段代码在干什么？**

四行路由注册，把设置页面的四个 API 端点绑定到对应的 handler。注意这里用到了 **RESTful 风格的多方法路由**：

- `GET /api/settings/token` — 查 Token
- `PUT /api/settings/token` — 设 Token
- `DELETE /api/settings/token` — 删 Token
- `PUT /api/settings/username` — 设用户名

同一个路径 `/api/settings/token` 因为 HTTP 方法不同，Gin 会路由到不同的 handler。这是 RESTful API 设计的核心思想：URL 标识资源，HTTP 方法标识操作。

### ✅ 验证

```powershell
go run .
```

**curl 验证：**

```powershell
curl http://127.0.0.1:8000/api/settings/token
curl -X PUT http://127.0.0.1:8000/api/settings/username -H "Content-Type: application/json" -d "{\"name\":\"Knock\"}"
```

**浏览器验证：** 刷新 `http://127.0.0.1:8000`，进入**设置页面**：

- ✅ Token 配置状态正常显示（显示脱敏后的 token）
- ✅ 用户名设置功能可用

> 🎯 **里程碑 6：设置页面通了！**

---

## Part 10：备份导出导入

### 目标

实现数据备份的导出（下载 zip）和导入（上传 zip）。

**备份功能的核心不是 zip，而是"安全地接受外部文件"。**

导出方向比较简单：把可信的本地 JSON 文件打成 zip 给用户下载。导入方向更危险：用户上传的 zip 可能缺文件、文件格式错、包含多余文件、文件巨大，甚至带路径攻击。所以导入流程必须先完整校验，再覆盖本地数据。

这一章的设计原则：

| 原则 | 在代码里的体现 |
|------|----------------|
| 默认拒绝 | 只接受 `backupFiles` 白名单里的文件 |
| 先验证后写入 | 先把所有可导入文件解析到 `parsed`，确认无误后才 `SaveJSON` |
| 覆盖前留后路 | 写入新数据前先创建 `pre-import` 快照 |
| 不信任 zip 路径 | 用 `filepath.Base(file.Name)` 只取文件名，避免 zip 内部路径影响本地路径 |

### Step 10.1：写 handlers/backup.go

创建 `handlers/backup.go`：

```go
package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"study-tracker-go/store"

	"github.com/gin-gonic/gin"
)

// 允许备份/导入的文件白名单
var backupFiles = map[string]bool{
	"errors.json":    true,
	"subjects.json":  true,
	"config.json":    true,
	"knowledge.json": true,
}

const backupMaxFileSize = 10 * 1024 * 1024 // 单个文件最大 10MB

// ExportBackup 导出备份 GET /api/backup/export
func ExportBackup(c *gin.Context) {
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)

	names := sortedBackupNames()
	for _, name := range names {
		path := store.Path(name)
		if _, err := os.Stat(path); err != nil {
			continue // 文件不存在就跳过
		}
		if err := addFileToZip(zipWriter, path, name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
			zipWriter.Close()
			return
		}
	}

	if err := zipWriter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}

	filename := fmt.Sprintf("study-tracker-backup-%s.zip", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "application/zip", buffer.Bytes())
}

// ImportBackup 导入备份 POST /api/backup/import
// 注意：前端发的是原始 zip 二进制，不是 multipart/form-data
func ImportBackup(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "备份文件不能为空"})
		return
	}

	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "请上传有效的 zip 备份文件"})
		return
	}

	parsed := map[string]interface{}{}
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		name := filepath.Base(file.Name)
		if !backupFiles[name] {
			if filepath.Ext(name) == ".json" {
				c.JSON(http.StatusBadRequest, gin.H{"detail": "备份包包含不支持的数据文件：" + file.Name})
				return
			}
			continue
		}
		if file.UncompressedSize64 > backupMaxFileSize {
			c.JSON(http.StatusBadRequest, gin.H{"detail": name + " 文件过大"})
			return
		}

		data, err := readZipFile(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
			return
		}

		var value interface{}
		if err := json.Unmarshal(data, &value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": name + " 不是有效 JSON 文件"})
			return
		}
		if err := validateBackupData(name, value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
			return
		}
		parsed[name] = value
	}

	if len(parsed) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "备份包中没有可恢复的数据文件"})
		return
	}

	// 导入前先备份当前数据
	snapshot, err := saveCurrentBackupSnapshot("pre-import")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
		return
	}

	imported := []string{}
	for name, value := range parsed {
		if err := store.SaveJSON(name, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
			return
		}
		imported = append(imported, name)
	}
	sort.Strings(imported)

	c.JSON(http.StatusOK, gin.H{
		"message":  "备份导入成功",
		"files":    imported,
		"snapshot": filepath.Base(snapshot),
	})
}

// --- 以下是辅助函数 ---

func sortedBackupNames() []string {
	names := []string{}
	for name := range backupFiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func addFileToZip(zipWriter *zip.Writer, path string, name string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := zipWriter.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	return err
}

func readZipFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func validateBackupData(name string, data interface{}) error {
	switch name {
	case "errors.json":
		list, ok := data.([]interface{})
		if !ok {
			return fmt.Errorf("errors.json 数据结构不正确")
		}
		for _, item := range list {
			if _, ok := item.(map[string]interface{}); !ok {
				return fmt.Errorf("errors.json 数据结构不正确")
			}
		}
	case "subjects.json":
		list, ok := data.([]interface{})
		if !ok {
			return fmt.Errorf("subjects.json 数据结构不正确")
		}
		for _, item := range list {
			if _, ok := item.(string); !ok {
				return fmt.Errorf("subjects.json 数据结构不正确")
			}
		}
	case "config.json", "knowledge.json":
		if _, ok := data.(map[string]interface{}); !ok {
			return fmt.Errorf("%s 数据结构不正确", name)
		}
	}
	return nil
}

func saveCurrentBackupSnapshot(prefix string) (string, error) {
	backupDir := filepath.Join(store.DataDir(), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}

	snapshot := filepath.Join(backupDir, fmt.Sprintf("%s-%s.zip", prefix, time.Now().Format("20060102-150405")))
	file, err := os.Create(snapshot)
	if err != nil {
		return "", err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	for _, name := range sortedBackupNames() {
		path := store.Path(name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if err := addFileToZip(zipWriter, path, name); err != nil {
			zipWriter.Close()
			return "", err
		}
	}
	return snapshot, zipWriter.Close()
}
```

**这段代码在干什么？**

这是整个项目中**最复杂的 handler 文件**，实现了数据备份的两个核心功能：导出（下载 zip）和导入（上传 zip）。两个函数都直接放在 handler 层（没有对应的 service 层），因为备份操作主要是文件 I/O，不涉及复杂的业务规则。

---

**ExportBackup — 导出备份**

整体流程：**收集文件列表 → 逐个打包进 zip → 写入 HTTP 响应（浏览器自动下载）**

关键步骤：

1. **用 `bytes.Buffer` 做内存缓冲** — 不写临时文件，直接在内存中构建 zip。`bytes.Buffer` 实现了 `io.Writer` 接口，可以传给 `zip.NewWriter`。
2. **遍历白名单文件** — `sortedBackupNames()` 按字母序返回 `["config.json", "errors.json", "knowledge.json", "subjects.json"]`，排序让 zip 内容可预测。
3. **跳过不存在的文件** — `os.Stat(path)` 检查文件是否存在，不存在就 `continue`。因为用户可能没有创建 `knowledge.json`。
4. **`addFileToZip` 辅助函数** — 打开源文件 → 在 zip 中创建条目 → `io.Copy` 复制内容。`io.Copy` 是 Go 中最常用的数据传输方式，直接对接两个 `io.Reader`/`io.Writer`。
5. **设置响应头** — `Content-Disposition: attachment; filename="..."` 告诉浏览器"下载"而非"展示"。文件名包含时间戳，比如 `study-tracker-backup-20260118-143025.zip`。
6. **`c.Data()` 发送二进制** — `c.JSON()` 只能发 JSON，`c.Data()` 可以发任意二进制数据，这里 contentType 是 `application/zip`。

---

**ImportBackup — 导入备份**

这个函数是整个项目中**安全考虑最多**的函数，因为接收的是不可信的外部数据。整体流程：**读原始 body → 解压 zip → 逐文件校验 → 备份当前数据 → 写入新数据**

关键的安全检查（按顺序）：

| 步骤 | 检查 | 为什么 |
|------|------|--------|
| 1 | `len(body) == 0` | 拒绝空请求，防止后续空指针 |
| 2 | `zip.NewReader` 失败 | 不是有效的 zip 文件，拒绝 |
| 3 | `backupFiles[name]` 白名单 | **只允许导入白名单文件**，防止路径穿越攻击（zip slip）。即使 zip 里放了 `../../../etc/passwd`，`filepath.Base()` 也只会取文件名部分，然后白名单检查会拒绝 |
| 4 | `file.UncompressedSize64 > 10MB` | 防止 zip 炸弹——一个恶意构造的 zip 可以解压出几百 GB 的数据 |
| 5 | `json.Unmarshal` 失败 | 文件内容不是合法 JSON，拒绝 |
| 6 | `validateBackupData` 结构校验 | 即使 JSON 合法，也要检查数据结构是否符合预期（errors.json 应该是对象数组，subjects.json 应该是字符串数组等） |
| 7 | 导入前 `saveCurrentBackupSnapshot` | **先备份再覆盖**！导入前自动把当前数据打成 zip 放到 `data/backups/pre-import-*.zip`，这样即使导入出错也能恢复 |

---

**Go 语言知识点详解：**

| 概念 | 说明 |
|------|------|
| `bytes.Buffer` | 一个实现了 `io.Reader` 和 `io.Writer` 的**内存缓冲区**。写入的内容存在内存中，最后通过 `.Bytes()` 取出来。适合构建中小型二进制数据，避免创建临时文件。 |
| `zip.NewWriter(&buffer)` | 创建一个 zip 写入器，写入目标是一个 `io.Writer`（这里是内存 buffer）。所有 `zipWriter.Create()` 和 `io.Copy()` 的数据最终都会进入这个 buffer。 |
| `zipWriter.Close()` | **必须调用！** 它会写入 zip 的中央目录（central directory），不关闭的 zip 文件是损坏的。注意 `ExportBackup` 中多处调用了 `Close()`——错误路径和成功路径都需要关闭，否则内存泄漏。 |
| `io.Copy(dst, src)` | Go 中最通用的数据复制函数。从 `src`（`io.Reader`）读取所有数据，写入 `dst`（`io.Writer`）。内部使用 32KB 缓冲区，高效且不占额外内存。 |
| `c.Data(status, contentType, data)` | Gin 的通用响应方法。和 `c.JSON()` 不同，`c.Data()` 不序列化，直接把字节切片写入响应体。适用于下载文件、图片等二进制数据。 |
| `io.ReadAll(c.Request.Body)` | 读取 HTTP 请求体的**全部原始字节**。前端用 `fetch(url, { method: 'POST', body: blob })` 发送 zip 文件的原始二进制，不是 `multipart/form-data`，所以不能用 `FormFile()`。这是 API 兼容性的关键点。 |
| `bytes.NewReader(body)` | 把字节切片 `[]byte` 包装成 `io.Reader`。`zip.NewReader` 需要一个 `io.ReaderAt`，而 `bytes.Reader` 实现了这个接口。 |
| `filepath.Base(file.Name)` | 提取路径的最后一段（文件名）。zip 文件中的条目可能包含路径如 `a/b/c/errors.json`，`filepath.Base()` 能防止路径穿越攻击。 |
| `data.([]interface{})` | **类型断言**（type assertion）。`json.Unmarshal` 到 `interface{}` 后，实际的 Go 类型是 `[]interface{}`（数组）或 `map[string]interface{}`（对象）。用 `.(type)` 语法检查实际类型。如果断言失败，`ok` 为 `false`，不会 panic。 |
| `filepath.Join(store.DataDir(), "backups")` | 跨平台安全地拼接文件路径。Windows 上用 `\`，Linux 上用 `/`，不要手动拼字符串。 |

---

**为什么导入前要先备份？** 这是数据安全的"后悔药"。导入本质上是覆盖操作——把用户的现有数据替换成备份文件里的数据。如果备份文件有问题（比如只包含部分数据、格式错误导致只导入了一半），用户就没有办法恢复了。导入前自动创建一个 pre-import 快照，放在 `data/backups/` 目录下，用户随时可以找回之前的数据。

**设计思路：white-list vs black-list。** 安全设计的一个基本原则是"默认拒绝"。这里用的是白名单 `backupFiles`——只允许已知的四个 JSON 文件，任何不在白名单中的文件（包括将来可能新增的敏感文件）都会被自动拒绝。如果用黑名单（"禁止 xxx"），新增敏感文件时很容易忘记更新黑名单，造成安全漏洞。

**为什么不直接边读边写？**

导入代码先把 zip 里的 JSON 全部读出来放进 `parsed`，等所有文件都通过校验后再写入 `data/`。这样可以避免"前两个文件已经覆盖，第三个文件校验失败"的半导入状态。半导入比完全失败更麻烦，因为数据之间可能不一致，比如 `errors.json` 导入了新科目名，但 `subjects.json` 没导入成功。

**为什么不能调用 `ExportBackup(c)` 做导入前快照？**

`ExportBackup(c)` 的职责是给浏览器写 HTTP 下载响应，而导入请求已经有自己的响应了。一个 HTTP 请求只能返回一次结果，所以导入内部必须用 `saveCurrentBackupSnapshot` 这种只写磁盘、不碰 HTTP 响应的辅助函数。这个例子也说明了为什么业务辅助函数不要和 handler 强绑定：handler 是对外响应，辅助函数才适合内部复用。

### Step 10.2：注册路由

```go
	r.GET("/api/backup/export", handlers.ExportBackup)
	r.POST("/api/backup/import", handlers.ImportBackup)
```

**这段代码在干什么？**

两行路由注册，把备份的导出和导入端点绑定到 handler。这里用了一个有意思的细节：

- `GET /api/backup/export` — 导出用 GET，因为浏览器直接访问 URL 就能下载，不需要请求体。
- `POST /api/backup/import` — 导入用 POST，因为需要上传文件（zip 二进制 body），GET 请求不能带 body。

**前端怎么用这个接口？** 导出时，前端直接 `window.open('/api/backup/export')` 或设置 `<a>` 标签的 href，浏览器收到 `Content-Disposition: attachment` 头后自动触发下载。导入时，前端用 `fetch` 把用户选择的 zip 文件作为 body 发送：

```javascript
// 前端导出：让浏览器自动下载
window.location.href = '/api/backup/export';

// 前端导入：发送原始 zip 二进制
const blob = new Blob([fileContent], { type: 'application/zip' });
await fetch('/api/backup/import', { method: 'POST', body: blob });
```

### ✅ 验证

```powershell
go run .
```

**curl 验证：** 浏览器访问 `http://127.0.0.1:8000/api/backup/export`，应该自动下载一个 zip 文件。

**浏览器验证：** 进入**设置/备份页面**：

- ✅ 点击"导出备份" → 浏览器下载 zip
- ✅ 选择一个备份 zip 文件导入 → 数据恢复成功

> 🎯 **里程碑 7：备份功能通了！**

---

## Part 11：OCR 识别

### 目标

对接 MinerU API，上传图片返回 Markdown 识别结果。

**OCR 是本项目里最不像普通 CRUD 的功能。**

普通接口一般是"前端请求 → 后端处理 → 立刻返回结果"。OCR 不一样，图片识别需要第三方服务排队和计算，所以流程变成：读取图片 → 读取 MinerU Token → 申请上传地址 → 上传图片 → 轮询任务状态 → 下载结果 zip → 提取 Markdown → 把 zip 内图片转成前端能直接显示的 base64 → 返回 Markdown。

这也是为什么 OCR 代码拆成很多小函数。不是为了显得复杂，而是每一步都可能失败：Token 没配、上传地址没拿到、上传失败、识别超时、zip 下载失败、Markdown 不存在、图片路径替换失败。拆开后每个错误都能定位到具体阶段。

### Step 11.1：写 service/ocr_service.go

创建 `service/ocr_service.go`（代码较长，直接复制即可）：

```go
package service

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"study-tracker-go/models"
	"study-tracker-go/store"
)

const mineruBaseURL = "https://mineru.net/api/v4"

// OCRImageBytes 对图片字节数据进行 OCR 识别，返回 Markdown
func OCRImageBytes(imageBytes []byte, fileName string) (string, error) {
	token, err := getMinerUToken()
	if err != nil {
		return "", err
	}

	batchID, uploadURL, err := createMinerUBatch(token, fileName)
	if err != nil {
		return "", err
	}

	// 上传图片到 MinerU 返回的预签名 URL
	req, err := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(imageBytes))
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("上传图片失败：HTTP %d", resp.StatusCode)
	}

	// 轮询等待识别完成
	zipURL, err := pollMinerUResult(token, batchID)
	if err != nil {
		return "", err
	}

	return downloadAndExtractMarkdown(zipURL)
}

// getMinerUToken 获取 MinerU Token（优先 config.json，其次环境变量）
func getMinerUToken() (string, error) {
	var config models.Config
	_ = store.LoadJSON("config.json", &config)
	token := strings.TrimSpace(config.MineruToken)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("MINERU_TOKEN"))
	}
	if token == "" {
		return "", fmt.Errorf("MinerU token not configured")
	}
	return token, nil
}

func createMinerUBatch(token string, fileName string) (batchID string, uploadURL string, err error) {
	body := map[string]interface{}{
		"files":          []map[string]string{{"name": fileName}},
		"model_version":  "vlm",
		"enable_formula": true,
		"enable_table":   false,
		"language":       "ch",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest(http.MethodPost, mineruBaseURL+"/file-urls/batch", bytes.NewReader(jsonBody))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("创建 MinerU 批次失败：HTTP %d", resp.StatusCode)
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			BatchID  string   `json:"batch_id"`
			FileURLs []string `json:"file_urls"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", "", err
	}
	if result.Code != 0 {
		return "", "", fmt.Errorf("MinerU error: %s", result.Msg)
	}
	if result.Data.BatchID == "" || len(result.Data.FileURLs) == 0 {
		return "", "", fmt.Errorf("MinerU 没有返回上传地址")
	}
	return result.Data.BatchID, result.Data.FileURLs[0], nil
}

func pollMinerUResult(token string, batchID string) (string, error) {
	deadline := time.Now().Add(5 * time.Minute)
	var lastErr error

	for time.Now().Before(deadline) {
		time.Sleep(3 * time.Second)

		zipURL, state, taskID, err := queryBatchResult(token, batchID)
		if err != nil {
			lastErr = err
			continue
		}
		if zipURL != "" {
			return zipURL, nil
		}
		if state == "failed" {
			return "", fmt.Errorf("MinerU OCR 失败")
		}
		if taskID != "" {
			zipURL, state, err := queryTaskResult(token, taskID)
			if err != nil {
				lastErr = err
				continue
			}
			if zipURL != "" {
				return zipURL, nil
			}
			if state == "failed" {
				return "", fmt.Errorf("MinerU OCR 失败")
			}
		}
	}

	if lastErr != nil {
		return "", fmt.Errorf("MinerU OCR 超时，最后一次错误：%w", lastErr)
	}
	return "", fmt.Errorf("MinerU OCR 超时")
}

func queryBatchResult(token string, batchID string) (zipURL string, state string, taskID string, err error) {
	req, err := http.NewRequest(http.MethodGet, mineruBaseURL+"/extract-results/batch/"+batchID, nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("查询 MinerU 批次失败：HTTP %d", resp.StatusCode)
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			ExtractResult []struct {
				State      string `json:"state"`
				FullZipURL string `json:"full_zip_url"`
				TaskID     string `json:"task_id"`
			} `json:"extract_result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", "", err
	}
	if result.Code != 0 {
		return "", "", "", fmt.Errorf("MinerU error: %s", result.Msg)
	}
	if len(result.Data.ExtractResult) == 0 {
		return "", "", "", nil
	}
	item := result.Data.ExtractResult[0]
	if item.State == "done" {
		return item.FullZipURL, item.State, item.TaskID, nil
	}
	return "", item.State, item.TaskID, nil
}

func queryTaskResult(token string, taskID string) (zipURL string, state string, err error) {
	req, err := http.NewRequest(http.MethodGet, mineruBaseURL+"/extract/task/"+taskID, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("查询 MinerU 任务失败：HTTP %d", resp.StatusCode)
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			State      string `json:"state"`
			FullZipURL string `json:"full_zip_url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}
	if result.Code != 0 {
		return "", "", fmt.Errorf("MinerU error: %s", result.Msg)
	}
	if result.Data.State == "done" {
		return result.Data.FullZipURL, result.Data.State, nil
	}
	return "", result.Data.State, nil
}

func downloadAndExtractMarkdown(zipURL string) (string, error) {
	resp, err := http.Get(zipURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载 OCR 结果失败：HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}

	imageMap := map[string]string{}
	markdown := ""

	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, "full.md") {
			content, err := readOCRZipFile(file)
			if err != nil {
				return "", err
			}
			markdown = string(content)
			continue
		}
		if strings.Contains(file.Name, "images/") && !file.FileInfo().IsDir() {
			content, err := readOCRZipFile(file)
			if err != nil {
				continue
			}
			ext := strings.ToLower(filepath.Ext(file.Name))
			mime := "image/png"
			if ext == ".jpg" || ext == ".jpeg" {
				mime = "image/jpeg"
			}
			imageMap[file.Name] = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(content)
		}
	}

	if markdown == "" {
		return "", fmt.Errorf("OCR 结果中没有 full.md")
	}

	return replaceMarkdownImages(markdown, imageMap), nil
}

func readOCRZipFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func replaceMarkdownImages(markdown string, imageMap map[string]string) string {
	re := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	return re.ReplaceAllStringFunc(markdown, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		src := parts[2]
		if dataURI, ok := imageMap[src]; ok {
			return `<img src="` + dataURI + `" width="400">`
		}
		base := filepath.Base(src)
		for name, dataURI := range imageMap {
			if filepath.Base(name) == base {
				return `<img src="` + dataURI + `" width="400">`
			}
		}
		return match
	})
}
```
**这段代码在干什么？**

`ocr_service.go` 是整个项目里最复杂的 service 文件，它负责对接 MinerU（一个第三方 OCR 云服务），把用户上传的题目图片转成 Markdown 文本。它的核心函数 `OCRImageBytes` 串联了一个**多步骤异步工作流**：

```
申请上传地址 → 上传图片 → 轮询等待识别完成 → 下载结果 zip → 提取 Markdown
```

**为什么不能像普通接口一样"发请求、收响应"就完事？**

OCR 识别一张图片通常需要几秒到几十秒，MinerU 的 v4 API 采用的是**异步模式**：你先申请一个"批次"拿到预签名上传地址，上传完图片后服务端在后台处理，你必须反复查询"好了没有"，好了再下载结果 zip。所以这个文件里的函数分工很明确：

| 函数 | 角色 |
|------|------|
| `OCRImageBytes` | 总调度，把下面所有步骤串起来 |
| `getMinerUToken` | 从配置文件或环境变量读取 MinerU Token |
| `createMinerUBatch` | 向 MinerU 申请批次，拿到上传 URL |
| `pollMinerUResult` | 轮询等待，最多等 5 分钟 |
| `queryBatchResult` / `queryTaskResult` | 单次查询识别状态 |
| `downloadAndExtractMarkdown` | 下载结果 zip，提取 `full.md` |
| `replaceMarkdownImages` | 把 zip 里的本地图片路径替换成 base64 data URI |

**关键 Go 知识点：**

- **匿名结构体解析 JSON**：`createMinerUBatch` 里没有单独定义响应结构体，而是用 `var result struct{...}` 临时声明。因为 MinerU 的响应格式只在这一个函数里用到，单独定义反而啰嗦。
- **`bytes.NewReader`**：上传图片时，图片在内存里是 `[]byte`，但 HTTP 请求需要 `io.Reader`，`bytes.NewReader` 把字节切片包装成可读流。
- **轮询 + 超时**：`pollMinerUResult` 用 `time.Now().Add(5 * time.Minute)` 设置截止时间，每 3 秒查一次。如果超时还没出结果就报错，防止程序无限等待。它还会用 `lastErr` 记录最后一次查询失败原因，避免真实问题（比如 token 失效或 MinerU 返回异常）被笼统的“超时”掩盖。
- **`defer resp.Body.Close()`**：每个 HTTP 响应都必须关闭 Body，否则 HTTP 连接不会释放回连接池，长时间运行会耗尽系统资源。
- **base64 内嵌图片**：`downloadAndExtractMarkdown` 把 zip 里的图片读到内存，用 `base64.StdEncoding.EncodeToString` 转成 data URI，然后把 Markdown 里的 `![...](images/x.png)` 替换成 `<img src="data:image/png;base64,...">`。这样做的原因是前端无法访问 OCR 服务器 zip 内的图片路径，嵌成 data URI 后 Markdown 可以独立显示。
- **正则替换**：`replaceMarkdownImages` 用 `regexp.MustCompile` 编译 Markdown 图片语法 `![...](...)` 的正则，然后用 `ReplaceAllStringFunc` 对每个匹配做自定义替换。`filepath.Base` 用于文件名模糊匹配——MinerU 返回的 Markdown 里图片路径的写法可能和 zip 内实际文件名不完全一致，用 `filepath.Base` 做后缀匹配能提高命中率。

**错误处理模式：**
这个文件的错误处理严格遵循 Go 惯例——每个可能出错的步骤都检查 `if err != nil`，立即返回。HTTP 调用还要额外检查 `resp.StatusCode`，因为 HTTP 200 不代表业务成功（MinerU 的 JSON 响应里可能返回 `{"code": 1, "msg": "参数错误"}`）。轮询时没有立刻返回所有查询错误，是为了给第三方服务短暂抖动留重试机会；但代码会记录 `lastErr`，超时时把最后一次真实错误带出来，方便排查。函数返回值使用了**命名返回值**（如 `func createMinerUBatch(...) (batchID string, uploadURL string, err error)`），让调用方一眼就知道返回的三个值分别是什么。

**Token 为什么先读 `config.json`，再读环境变量？**

`config.json` 是给普通用户和设置页用的，保存后重启程序也还在；环境变量 `MINERU_TOKEN` 更适合开发调试或临时部署，不想把 token 写进数据文件时可以使用。优先读配置文件，是为了让设置页的保存结果立即生效；配置里没有时再读环境变量，是为了保留开发便利性。

**为什么要同时检查 HTTP 状态码和 MinerU 返回的 `code/state`？**

这两个层级含义不同：

| 检查 | 说明 |
|------|------|
| `resp.StatusCode` | 网络和 HTTP 层是否成功，例如上传地址是否接受了 PUT |
| `result.Code` / `state` | MinerU 业务层是否成功，例如参数是否正确、OCR 是否完成或失败 |

只检查其中一个都不够。HTTP 200 只能说明请求送到了服务端，不代表 OCR 任务成功；业务 `code == 0` 也需要先能正确读到响应体。




**流程解释：**

```
图片字节 → 创建 MinerU 批次 → 获取预签名上传 URL → PUT 上传图片
→ 轮询 batch 结果（每 3 秒查一次，最长 5 分钟）
→ 下载结果 zip → 提取 full.md → 将图片替换为 base64 data URI
→ 返回 Markdown
```

### Step 11.2：写 handlers/ocr.go

创建 `handlers/ocr.go`：

```go
package handlers

import (
	"io"
	"net/http"

	"study-tracker-go/service"

	"github.com/gin-gonic/gin"
)

// OCRImage 处理 POST /api/ocr
// 注意：前端发的是原始图片 Blob，不是 multipart/form-data
func OCRImage(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "No file uploaded"})
		return
	}

	markdown, err := service.OCRImageBytes(body, "ocr_upload.png")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"markdown": markdown})
}
```


**这段代码在干什么？**

`handlers/ocr.go` 是 OCR 功能的 HTTP 入口，处理 `POST /api/ocr` 请求。这个 handler 非常薄，只做三件事：读请求体、调 service、返回 JSON。

**为什么用 `io.ReadAll(c.Request.Body)` 而不是 `c.FormFile("file")`？**

这是前后端兼容性的关键约束：前端 Vue 代码发送 OCR 图片时用的是 `fetch(url, { body: imageBlob })`，直接把图片的二进制数据作为请求体发送，不是 `multipart/form-data` 表单格式。所以后端必须用 `io.ReadAll(c.Request.Body)` 读取原始字节流，`FormFile` 无法解析这种请求体。

**返回格式：** 返回 `{"markdown": "..."}` 是因为前端读取 `res.markdown` 来填充题目文本框。错误时返回 `{"detail": "错误原因"}` 保持与 FastAPI 版本一致的错误格式。

**handler 层的设计原则：**
这个 handler 体现了三层架构中 handler 层的核心职责：只做 HTTP 翻译（请求体解析 + 响应格式化），不包含任何业务逻辑。真正的 OCR 流程全部在 `service.OCRImageBytes` 里处理。



### Step 11.3：注册路由

```go
	r.POST("/api/ocr", handlers.OCRImage)
```


**这段代码在干什么？**

注册 OCR 路由，把 `POST /api/ocr` 请求交给 `handlers.OCRImage` 处理。路由注册本身很简单，但要注意一个设计习惯：**不要设计过宽的参数路由**，比如 `r.POST("/api/:id", ...)` 这种会和很多具体接口抢同一层路径。当前教程里的 `/api/errors/:id` 不会匹配 `/api/ocr`，因为它们路径层级不同；Gin 也会先在路由树里查找已注册的具体路径，找不到时才进入 `NoRoute`。所以这里更重要的不是“靠书写顺序躲开参数”，而是把具体接口路径设计清楚，并把所有 API 路由集中写在 `r.Run()` 之前，方便检查。



### ✅ 验证

OCR 需要有效的 MinerU Token，先设置：

```powershell
curl -X PUT http://127.0.0.1:8000/api/settings/token -H "Content-Type: application/json" -d "{\"token\":\"你的token\"}"
```

或用环境变量：`$env:MINERU_TOKEN="你的token"`，然后 `go run .`。

**浏览器验证：** 进入**添加错题页面** → 点击 OCR 上传图片按钮 → 选择一张题目图片 → 等待识别 → Markdown 结果自动填入题目框。

> 🎯 **里程碑 8：OCR 识别通了！**

---

## Part 12：版本和更新接口

### 目标

实现版本查询和更新检查的桩接口（先不做真正的自动更新，但接口必须存在，否则前端设置页会报错）。

**这一 Part 的重点是"接口兼容"，不是"真的自动更新"。**

迁移后端时经常会遇到这种情况：前端已经写好了某些调用，即使第一阶段后端暂时不实现完整功能，也必须返回前端能理解的 JSON。版本和更新接口就是典型例子。设置页会调用这些接口，如果 Go 后端直接返回 404，用户看到的不是"暂不支持自动更新"，而是前端报错甚至页面异常。

所以这里先做桩接口：路径、HTTP 方法、字段名都和前端约定一致，但业务含义明确告诉前端"当前没有更新、暂不支持自动替换"。这是一种渐进迁移策略，先保证页面不坏，再逐步实现复杂功能。

### Step 12.1：写 service/update_service.go

创建 `service/update_service.go`：

```go
package service

import (
	"encoding/json"
	"os"
	"time"
)

type VersionInfo struct {
	Version   string `json:"version"`
	Repo      string `json:"repo"`
	AssetName string `json:"asset_name"`
	AppExe    string `json:"app_exe"`
}

func GetVersionResponse() map[string]interface{} {
	info := loadVersionInfo()
	return map[string]interface{}{
		"version":         info.Version,
		"repo":            info.Repo,
		"asset_name":      info.AssetName,
		"app_exe":         info.AppExe,
		"can_auto_update": false,
	}
}

func CheckUpdate(force bool) map[string]interface{} {
	info := loadVersionInfo()
	return map[string]interface{}{
		"ok":              true,
		"message":         "Go 版暂未启用自动更新检查",
		"current_version": info.Version,
		"latest_version":  info.Version,
		"tag_name":        "",
		"has_update":      false,
		"repo":            info.Repo,
		"asset_name":      info.AssetName,
		"asset_found":     false,
		"asset_size":      0,
		"published_at":    "",
		"html_url":        "",
		"notes":           "",
		"can_auto_update": false,
		"checked_at":      time.Now().Format("2006-01-02 15:04:05"),
	}
}

func ApplyUpdate() map[string]interface{} {
	return map[string]interface{}{
		"message":         "Go 版暂不支持自动替换，请手动更新程序文件",
		"can_auto_update": false,
	}
}

func loadVersionInfo() VersionInfo {
	info := VersionInfo{
		Version:   "0.0.0-dev",
		Repo:      "Zilvren/Learning-Assitant",
		AssetName: "Tracker.zip",
		AppExe:    "Tracker.exe",
	}

	data, err := os.ReadFile("version.json")
	if err != nil {
		return info
	}
	_ = json.Unmarshal(data, &info)
	if info.Version == "" {
		info.Version = "0.0.0-dev"
	}
	if info.Repo == "" {
		info.Repo = "Zilvren/Learning-Assitant"
	}
	if info.AssetName == "" {
		info.AssetName = "Tracker.zip"
	}
	if info.AppExe == "" {
		info.AppExe = "Tracker.exe"
	}
	return info
}
```


**这段代码在干什么？**

`update_service.go` 是版本和更新检查的 service 层。它目前是**桩实现**（stub）——接口存在、返回格式正确，但暂时不真的去 GitHub 检查新版本。

**为什么要先做桩？**

原 Vue 前端的设置页会主动调用 `/api/version` 和 `/api/update/check`。如果这些接口不存在（返回 404），前端 JavaScript 会报错，整个设置页可能白屏。所以 Go 迁移第一阶段先把这些接口搭起来，返回"暂无更新"的兼容数据，让前端正常工作。真正的自动更新逻辑（调 GitHub API、下载 Release、替换 exe）是第二阶段的任务。

**三个对外函数的职责：**

| 函数 | 对应接口 | 前端什么时候调 |
|------|---------|-------------|
| `GetVersionResponse` | `GET /api/version` | 打开设置页时 |
| `CheckUpdate` | `GET /api/update/check?force=true` | 点击"检查更新"按钮 |
| `ApplyUpdate` | `POST /api/update/apply` | 点击"立即更新"按钮 |

**`loadVersionInfo` 的默认值模式：**
读取 `version.json` 文件，如果文件不存在或字段为空，就用硬编码的默认值（`0.0.0-dev`）。这是一个常见的**防御性编程**模式：程序不能因为缺少配置文件就崩溃。函数先初始化一个带默认值的 `VersionInfo` 结构体，然后尝试读取 JSON 文件覆盖，覆盖后再检查每个字段是否为空，空则回退到默认值。

**`map[string]interface{}` 返回类型：**
你可能会注意到这几个函数返回的是 `map[string]interface{}` 而不是具体的结构体。这是因为版本检查的返回字段非常多（13 个），而且后续会频繁加减字段，用 `map` 比定义结构体更灵活，直接构造 JSON 也更直观。



#### 这个文件里的函数怎么理解

| 函数 | 解释 |
|------|------|
| `GetVersionResponse` | 给 `/api/version` 用，返回版本信息和是否支持自动更新 |
| `CheckUpdate` | 给 `/api/update/check` 用，先返回兼容字段，暂不做真正联网检查 |
| `ApplyUpdate` | 给 `/api/update/apply` 用，先返回“暂不支持自动替换” |
| `loadVersionInfo` | 读取 `version.json`，没有就用默认值 |

傻瓜式理解：

1. 前端设置页先问“你是什么版本”。
2. 如果用户点检查更新，再问“有没有新版本”。
3. 如果用户点立即更新，再问“你现在能不能自动替换”。
4. 现在 Go 版先把这些问题都答出来，但先不真的下载更新。

### Step 12.2：写 handlers/update.go

创建 `handlers/update.go`：

```go
package handlers

import (
	"study-tracker-go/service"

	"github.com/gin-gonic/gin"
)

func GetVersion(c *gin.Context) {
	c.JSON(200, service.GetVersionResponse())
}

func CheckUpdate(c *gin.Context) {
	force := c.Query("force") == "true"
	c.JSON(200, service.CheckUpdate(force))
}

func ApplyUpdate(c *gin.Context) {
	c.JSON(200, service.ApplyUpdate())
}
```


**这段代码在干什么？**

`handlers/update.go` 是版本更新相关的三个 HTTP handler，每个都非常简单——只做参数提取和返回 JSON，不包含任何业务逻辑。

**值得注意的细节：**

- `CheckUpdate` 里 `c.Query("force") == "true"` 从 URL 查询参数里取 `force` 的值。Gin 的 `c.Query` 返回的是字符串，所以要和字符串 `"true"` 比较。前端发的是 `?force=true`，如果 force 参数不存在，`c.Query("force")` 返回空字符串，条件自然为 false。
- 三个 handler 都直接返回 HTTP 200，因为当前都是桩实现，不存在真正的"失败"情况（service 层总是返回成功）。
- 这种"极薄 handler"是三层架构的典型特征：handler 不思考，只做翻译（HTTP 请求翻译成 service 调用，service 结果翻译成 HTTP 响应）。



### Step 12.3：注册路由

```go
	r.GET("/api/version", handlers.GetVersion)
	r.GET("/api/update/check", handlers.CheckUpdate)
	r.POST("/api/update/apply", handlers.ApplyUpdate)
```


**这段代码在干什么？**

注册三个版本/更新相关的路由。注意 `CheckUpdate` 用的是 `GET` 方法（因为它是查询操作），而 `ApplyUpdate` 用的是 `POST`（因为它会触发状态变更行为）。这是 RESTful API 的惯例：读操作用 GET，写操作用 POST。虽然当前这些接口都是桩实现，但路由方法的选择已经为未来的真实实现做好了铺垫。



### ✅ 验证

```powershell
go run .
```

**curl 验证：**

```powershell
curl http://127.0.0.1:8000/api/version
```

**浏览器验证：** 进入**设置 → 关于/更新页面** → 版本号正常显示，不再报错。

> 🎯 **里程碑 9：版本接口通了！所有 API 全部完成。**

---

## Part 13：测试页面 + 最终验证

### 目标

创建独立 API 测试页面，做一次完整的全功能验证。

**为什么还要一个独立测试页面？**

Vue 前端页面出了问题时，你很难第一眼判断是前端组件错了、请求路径错了、还是后端接口错了。`test.html` 的作用就是把 Vue 暂时拿开，直接用最简单的 `fetch` 调接口。如果测试页面能成功，而 Vue 页面失败，问题大概率在前端；如果测试页面也失败，问题就在后端路由、handler、service 或数据文件。

这个页面不是正式产品功能，而是迁移阶段的排错工具。它越简单越好，因为它的价值就在于减少干扰因素。

### Step 13.1：创建 test.html

在 `server-go/` 目录下创建 `test.html`（独立 API 测试页面，不依赖 Vue 前端）：

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <title>Go API 测试</title>
  <style>
    body { font-family: sans-serif; max-width: 800px; margin: 20px auto; }
    h2 { border-bottom: 1px solid #ddd; padding-bottom: 4px; }
    input { margin: 4px; padding: 4px; }
    button { margin: 4px; padding: 6px 12px; cursor: pointer; }
    pre { background: #f5f5f5; padding: 12px; border-radius: 4px; max-height: 400px; overflow: auto; }
  </style>
</head>
<body>
  <h2>科目管理</h2>
  <input id="subjectName" placeholder="科目名">
  <button onclick="addSubject()">添加科目</button>
  <button onclick="deleteSubject()">删除科目</button>
  <button onclick="getSubjects()">刷新科目</button>

  <h2>错题管理</h2>
  <input id="errSubject" placeholder="科目">
  <input id="errQuestion" placeholder="题目">
  <input id="errWrong" placeholder="错误答案">
  <input id="errCorrect" placeholder="正确答案">
  <button onclick="addError()">添加错题</button>
  <button onclick="getErrors()">查看错题</button>

  <input id="errorId" placeholder="错题ID">
  <button onclick="reviewError()">标记复习</button>
  <button onclick="deleteError()">删除错题</button>

  <h2>其他接口</h2>
  <button onclick="getTags()">标签</button>
  <button onclick="getDailyPush()">每日推送</button>
  <button onclick="getVersion()">版本</button>
  <button onclick="exportBackup()">导出备份</button>

  <h3>返回结果</h3>
  <pre id="result">点击按钮查看结果...</pre>

  <script>
    function show(data) {
      document.getElementById('result').textContent = JSON.stringify(data, null, 2)
    }
    async function request(path, options) {
      const res = await fetch(path, options)
      const type = res.headers.get('content-type') || ''
      if (type.includes('application/json')) return res.json()
      return { status: res.status, text: await res.text() }
    }
    async function getSubjects() { show(await request('/api/subjects')) }
    async function addSubject() {
      const name = document.getElementById('subjectName').value
      show(await request('/api/subjects', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name })
      }))
    }
    async function deleteSubject() {
      const name = document.getElementById('subjectName').value
      show(await request('/api/subjects/' + encodeURIComponent(name), { method: 'DELETE' }))
    }
    async function addError() {
      show(await request('/api/errors', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          subject: document.getElementById('errSubject').value,
          question: document.getElementById('errQuestion').value,
          wrong: document.getElementById('errWrong').value,
          correct: document.getElementById('errCorrect').value
        })
      }))
    }
    async function getErrors() { show(await request('/api/errors')) }
    async function reviewError() {
      const id = document.getElementById('errorId').value
      show(await request('/api/errors/' + id + '/review', { method: 'PUT' }))
    }
    async function deleteError() {
      const id = document.getElementById('errorId').value
      show(await request('/api/errors/' + id, { method: 'DELETE' }))
    }
    async function getTags() { show(await request('/api/tags')) }
    async function getDailyPush() { show(await request('/api/daily-push')) }
    async function getVersion() { show(await request('/api/version')) }
    function exportBackup() { window.open('/api/backup/export') }
  </script>
</body>
</html>
```


**这段代码在干什么？**

`test.html` 是一个**独立 API 测试页面**，完全不依赖 Vue 前端。它的作用是让你在浏览器里逐个验证所有后端接口是否正常工作，相当于一个可视化的 Postman。

**设计思路：**

- 纯静态 HTML，不需要编译，通过 Gin 的 `StaticFile` 路由即可访问。
- 页面上每个按钮对应一个后端 API，点击后调用 `fetch()` 发送请求，返回的 JSON 格式化显示在页面底部的 `<pre>` 区域。
- `show(data)` 函数用 `JSON.stringify(data, null, 2)` 把结果美化输出（缩进 2 空格），方便肉眼检查字段是否完整。
- `request(path, options)` 是一个通用封装函数：它先检查响应 `Content-Type`，如果是 JSON 就解析成对象，如果是二进制（比如备份下载）就显示状态码和文本摘要。

**涵盖的测试接口：**
科目增删查、错题增删查、标记复习、标签、每日推送、版本、备份导出。OCR 接口没放在这个页面里，因为 OCR 需要文件上传（`<input type="file">`），测试起来稍复杂。

**为什么要做测试页面？**
三层架构迁移涉及十几个接口，一个个用 curl 测试效率低且容易漏。一个集中的测试页面可以一键验证所有接口，新功能加完立刻就能看到结果。



### Step 13.2：注册测试页面路由

在 `main.go` 的 `main()` 函数中，建议和其他明确路由放在一起，通常写在 `r.NoRoute(serveFrontend)` **之前**：

```go
	r.StaticFile("/test", "./test.html")
```


**这段代码在干什么？**

用 Gin 的 `r.StaticFile("/test", "./test.html")` 注册 `/test` 路由。访问 `http://127.0.0.1:8000/test` 时，Gin 直接读取 `test.html` 文件内容并返回给浏览器。

**`StaticFile` vs `Static`：**
- `StaticFile(route, filepath)` 映射单个文件，适合测试页、favicon 这种场景。
- `Static(route, dir)` 映射整个目录，适合前端静态资源（CSS、JS、图片等）。

**为什么建议放在 `r.NoRoute(...)` 之前？**
这里是代码组织习惯，不是 Gin 的硬性顺序规则。Gin 会先在路由树里查找 `GET /test` 这种明确注册的路径，找不到匹配路由时才执行 `NoRoute`。所以 `/test` 不会因为代码写在 `NoRoute` 后面就天然失效。

仍然建议把 `/test` 放在 `NoRoute` 前，是因为这份教程把“明确 API/测试路由”集中放在前面，把“前端兜底路由”放在最后，读代码时更容易看出：先处理后端接口和测试页，剩下的页面路径再交给 Vue 前端。



### Step 13.3：最终全功能验证

```powershell
gofmt -w .
go build .
go run .
```

**1. 独立测试页面：** 打开 `http://127.0.0.1:8000/test`，逐个按钮测试所有 API。

**2. Vue 前端全功能验证：** 打开 `http://127.0.0.1:8000`，逐页检查：

| 页面/功能 | 检查项 | 期望结果 |
|-----------|--------|---------|
| 🏠 首页/仪表盘 | 到期错题数、逾期数、知识点、建议 | 数据正常显示，不空白 |
| 📋 错题列表 | 列表加载、科目筛选、关键词搜索 | 列表有数据，筛选生效 |
| ➕ 添加错题 | 填写表单、OCR 上传（需Token） | 添加成功后列表更新 |
| ✏️ 错题详情 | 编辑字段、点击复习、删除 | 编辑保存/复习计数/删除成功 |
| 🏷️ 标签 | 标签列表、按标签筛选错题 | 标签显示、筛选可用 |
| 📚 科目管理 | 添加/删除科目 | 操作后列表更新 |
| ⚙️ 设置 | Token 状态、用户名设置 | 状态正常显示 |
| 💾 备份 | 导出下载 zip、导入恢复 | 导出成功/导入成功 |
| 🔄 版本/更新 | 版本号显示 | 显示版本信息，不报错 |

全部通过后——**恭喜，迁移完成！** 🎉

---

## 常见错误速查

| 错误信息 | 可能原因 | 解决办法 |
|----------|---------|---------|
| `main redeclared` | 目录中有两个 `func main()` | 检查是否有多余的 `.go` 文件（如 `Gintest.go`），删除或改名 |
| `pattern frontend/dist: no matching files found` | `embed.go` 要求的目录不存在 | 确认已复制 `frontend/dist` |
| `cannot find package` | import 路径写错了 | 检查模块名是 `study-tracker-go`，不是 `server-go` |
| 前端显示"请求失败" | 后端返回格式不对 | 确认用 `{"detail":"..."}` 格式，列表用 `{"errors":[...],"total":N}` |
| 备份导入失败 | 前端发的是 raw body，后端用了 `FormFile` | 确认用 `io.ReadAll(c.Request.Body)` |
| `undefined: handlers.XXX` | main.go 没 import handlers 包 | 确认 import 中有 `"study-tracker-go/handlers"` |
| 大括号语法错误 | Go 不允许 `{` 换行 | `func main() {` 必须在同一行 |

---

## 函数逐个傻瓜式解释

这一节专门补“每个函数到底在干什么”。你可以把它当成查字典：写到哪个函数看不懂，就回来查这一节。

### main.go

#### `main()`

| 项目 | 解释 |
|------|------|
| 所在层 | 程序入口 |
| 谁调用它 | Go 程序启动时自动调用 |
| 参数 | 没有 |
| 返回值 | 没有 |
| 作用 | 创建 Gin 服务器、注册所有路由、启动 8000 端口 |

傻瓜式流程：

1. `r := gin.Default()`：创建一个 HTTP 服务器。
2. `r.GET(...)` / `r.POST(...)`：告诉服务器“这个地址来了请求，就交给哪个函数处理”。
3. `r.StaticFile("/test", "./test.html")`：让浏览器可以访问测试页。
4. `r.NoRoute(serveFrontend)`：如果不是 API，就交给 Vue 前端。
5. `r.Run("127.0.0.1:8000")`：真正开始监听浏览器请求。

为什么要这样写：  
`main()` 像总开关，它不应该写太多业务逻辑，只负责“把路由接起来”。

#### `serveFrontend(c *gin.Context)`

| 项目 | 解释 |
|------|------|
| 所在层 | HTTP 静态文件层 |
| 谁调用它 | Gin 找不到其他路由时自动调用 |
| 参数 | `c *gin.Context`，当前请求上下文 |
| 返回值 | 没有，直接给浏览器写响应 |
| 作用 | 返回 Vue 前端文件；找不到具体文件时返回 `index.html` |

傻瓜式流程：

1. 判断路径是不是 `/api` 开头。
2. 如果是 `/api`，说明这是接口请求，但没有匹配到路由，返回 404。
3. 如果不是 `/api`，就当成前端页面或静态资源。
4. 先尝试从 `frontend/dist` 找真实文件，比如 `assets/index.js`。
5. 找不到就返回 `frontend/dist/index.html`，让 Vue 路由自己处理。

为什么需要它：  
Vue 是单页应用。浏览器刷新 `/settings` 时，后端磁盘里没有 `settings` 文件，所以必须回退到 `index.html`。

---

### models/models.go

#### `ErrorProblem`

| 项目 | 解释 |
|------|------|
| 类型 | 结构体，不是函数 |
| 作用 | 描述一条错题长什么样 |
| 谁使用它 | `service/error_service.go`、`handlers/errors.go`、JSON 文件 |

重点字段：

| 字段 | 意思 |
|------|------|
| `ID int` | 错题编号 |
| `Subject string` | 科目 |
| `Question string` | 题目内容 |
| `Wrong/Correct/Reason` | 错答、正解、错因 |
| `Tags/ReasonTags` | 题目标签、错因标签 |
| `ReviewCount` | 已复习次数 |
| `LastReview *string` | 上次复习时间，`nil` 表示没复习过 |
| `NextReview` | 下次该复习的日期 |

为什么字段首字母大写：  
Go 里首字母大写表示“别的 package 可以访问”。如果写成 `id int`，JSON 序列化和其他包都不好用。

#### `AddErrorRequest`

| 项目 | 解释 |
|------|------|
| 类型 | 请求体结构体 |
| 谁使用它 | `handlers.CreateError` |
| 作用 | 接收前端新增错题时发来的 JSON |

前端发送：

```json
{"subject":"数学","question":"1+1=?","wrong":"3","correct":"2"}
```

Gin 会把它变成：

```go
models.AddErrorRequest{Subject: "数学", Question: "1+1=?"}
```

#### `UpdateErrorRequest`

| 项目 | 解释 |
|------|------|
| 类型 | 请求体结构体 |
| 谁使用它 | `handlers.UpdateError`、`service.UpdateError` |
| 作用 | 接收前端编辑错题时传来的字段 |

为什么字段是 `*string`、`*[]string`：  
因为更新时要区分两件事：

```text
没传 title：不要改标题
传了 title: ""：把标题改成空字符串
```

不用指针就分不清这两种情况。

#### `DailyPushResult`

| 项目 | 解释 |
|------|------|
| 类型 | 返回体结构体 |
| 谁使用它 | `service.GetDailyPush` |
| 作用 | 规定 `/api/daily-push` 返回给前端的数据格式 |

为什么要单独定义：  
每日推送返回的字段很多，用结构体比临时 `map` 更清楚，也更不容易拼错字段名。

---

### store/json_store.go

#### `SetDataDir(dir string)`

| 项目 | 解释 |
|------|------|
| 所在层 | store 数据层 |
| 谁调用它 | 一般测试代码会调用，正式运行可不调用 |
| 参数 | `dir string`，新的数据目录 |
| 返回值 | 没有 |
| 作用 | 修改 JSON 文件保存目录 |

初学阶段可以先记住：默认数据目录是 `data`，这个函数暂时用不到。

#### `DataDir() string`

| 项目 | 解释 |
|------|------|
| 所在层 | store 数据层 |
| 谁调用它 | 备份函数 `saveCurrentBackupSnapshot` |
| 参数 | 没有 |
| 返回值 | 数据目录路径，比如 `"data"` |
| 作用 | 确保数据目录存在，并把目录名返回出去 |

为什么里面有 `os.MkdirAll`：  
如果 `data/` 不存在，备份时创建文件会失败。所以先创建目录。

#### `Path(filename string) string`

| 项目 | 解释 |
|------|------|
| 所在层 | store 数据层 |
| 谁调用它 | 备份导出函数 |
| 参数 | 文件名，比如 `"errors.json"` |
| 返回值 | 完整路径，比如 `data/errors.json` |
| 作用 | 统一拼接数据文件路径 |

为什么不用字符串相加：  
Windows 用 `\`，Linux/macOS 用 `/`。`filepath.Join` 会自动处理不同系统的路径分隔符。

#### `LoadJSON(filename string, target interface{}) error`

| 项目 | 解释 |
|------|------|
| 所在层 | store 数据层 |
| 谁调用它 | 各个 service |
| 参数 | 文件名 + 要写入的变量地址 |
| 返回值 | `error`，成功就是 `nil` |
| 作用 | 从 `data/*.json` 读取数据到 Go 变量 |

调用例子：

```go
var errors []models.ErrorProblem
err := store.LoadJSON("errors.json", &errors)
```

为什么传 `&errors`：  
`&errors` 是变量地址。函数拿到地址后，才能把 JSON 内容“填回”这个变量。

如果文件不存在为什么不报错：  
第一次运行时可能还没有 `errors.json`。这不是程序错误，所以直接返回 `nil`。

#### `SaveJSON(filename string, data interface{}) error`

| 项目 | 解释 |
|------|------|
| 所在层 | store 数据层 |
| 谁调用它 | 各个 service 和备份导入 |
| 参数 | 文件名 + 要保存的数据 |
| 返回值 | `error` |
| 作用 | 把 Go 变量保存成格式化 JSON 文件 |

傻瓜式流程：

1. 确保 `data/` 目录存在。
2. 用 `json.MarshalIndent` 把 Go 数据转成漂亮的 JSON。
3. 用 `os.WriteFile` 写入磁盘。

---

### service/subject_service.go

#### `GetAllSubjects() ([]string, error)`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.GetSubjects`、`SubjectExists`、`GetDailyPush` |
| 参数 | 没有 |
| 返回值 | 科目列表 + 错误 |
| 作用 | 读取 `subjects.json` 并保证返回的是空数组而不是 `nil` |

为什么要把 `nil` 变成 `[]string{}`：  
前端更喜欢收到 `[]`，不喜欢收到 `null`。`[]` 表示“列表为空”，更好处理。

#### `SubjectExists(name string) bool`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `CreateError`、`UpdateError` |
| 参数 | 科目名 |
| 返回值 | `true` 表示存在，`false` 表示不存在 |
| 作用 | 判断用户传来的科目是不是合法科目 |

为什么新增错题前要检查科目：  
如果不检查，用户可以新增一个科目列表里不存在的错题，前端筛选时就会乱。

#### `AddSubject(name string) ([]string, error)`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.AddSubject` |
| 参数 | 新科目名 |
| 返回值 | 更新后的科目列表 + 错误 |
| 作用 | 校验科目名、检查重复、保存新列表 |

傻瓜式流程：

1. 去掉科目前后空格。
2. 如果空字符串，返回错误。
3. 读取旧科目列表。
4. 检查是否重复。
5. 追加新科目。
6. 保存回 `subjects.json`。
7. 把更新后的列表返回给 handler。

#### `DeleteSubject(name string) ([]string, error)`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.DeleteSubject` |
| 参数 | 要删除的科目名 |
| 返回值 | 更新后的科目列表 + 错误 |
| 作用 | 从科目列表中移除指定科目 |

为什么要返回更新后的列表：  
原 FastAPI 接口就是这样返回的，前端可以直接用返回值刷新侧边栏。

---

### handlers/subjects.go

#### `GetSubjects(c *gin.Context)`

| 项目 | 解释 |
|------|------|
| 所在层 | handler HTTP 层 |
| 谁调用它 | 浏览器请求 `GET /api/subjects` |
| 参数 | Gin 请求上下文 |
| 返回值 | 没有，直接写 JSON 响应 |
| 作用 | 把科目列表返回给前端 |

傻瓜式流程：

1. 调 `service.GetAllSubjects()` 拿科目列表。
2. 如果出错，返回 `500` 和 `{"detail": "..."}`
3. 如果成功，返回 `200` 和 `{"subjects": [...]}`

#### `AddSubject(c *gin.Context)`

| 项目 | 解释 |
|------|------|
| 所在层 | handler HTTP 层 |
| 谁调用它 | 浏览器请求 `POST /api/subjects` |
| 参数 | Gin 请求上下文 |
| 返回值 | 没有，直接写 JSON 响应 |
| 作用 | 解析请求体，调用 service 添加科目 |

为什么 handler 里只做解析和返回：  
检查重复、保存文件是业务逻辑，放在 service。handler 只负责 HTTP。

#### `DeleteSubject(c *gin.Context)`

| 项目 | 解释 |
|------|------|
| 所在层 | handler HTTP 层 |
| 谁调用它 | 浏览器请求 `DELETE /api/subjects/:name` |
| 参数 | Gin 请求上下文 |
| 返回值 | 没有 |
| 作用 | 从 URL 取科目名，调用 service 删除 |

`c.Param("name")` 的意思：  
如果路由是 `/api/subjects/:name`，请求 `/api/subjects/数学`，那么 `c.Param("name")` 就是 `"数学"`。

---

### service/error_service.go：创建和查询

#### `CreateError(req models.AddErrorRequest) (models.ErrorProblem, error)`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.CreateError` |
| 参数 | 新增错题请求体 |
| 返回值 | 新错题 + 错误 |
| 作用 | 创建一条完整错题并保存到 `errors.json` |

傻瓜式流程：

1. 清理科目和题目前后空格。
2. 检查科目是否存在。
3. 检查题目是否为空。
4. 给 `wrong/correct/reason` 设置默认值 `"未记录"`。
5. 如果没有标题，就截取题目前 40 个字符当标题。
6. 如果标签是 `nil`，改成空切片。
7. 读取已有错题。
8. 扫描最大 ID，新 ID = 最大 ID + 1。
9. 创建 `models.ErrorProblem`。
10. 保存回 `errors.json`。
11. 返回新错题。

为什么 ID 用最大 ID + 1：  
删除错题后不复用旧 ID，历史记录更稳定，前端也不容易混乱。

#### `firstRunes(text string, max int) string`

| 项目 | 解释 |
|------|------|
| 所在层 | service 辅助函数 |
| 谁调用它 | `CreateError` |
| 参数 | 原字符串 + 最大字符数 |
| 返回值 | 截断后的字符串 |
| 作用 | 按“字符”截断标题，避免中文被截坏 |

为什么不用 `text[:40]`：  
中文一个字通常占 3 个字节，直接按字节切可能切坏。`[]rune` 是按字符处理。

#### `GetAllErrors(subject, keyword, tag, reasonTag string) ([]models.ErrorProblem, error)`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.GetErrors`、`GetDailyPush` |
| 参数 | 筛选条件 |
| 返回值 | 筛选后的错题列表 + 错误 |
| 作用 | 读取错题并按前端条件筛选 |

筛选逻辑：

```text
subject 不为空：只要这个科目的错题
keyword 不为空：题目、标题、错因、标签里包含关键词才要
tag 不为空：题目标签匹配才要
reasonTag 不为空：错因标签匹配才要
```

#### `normalizeReviewFields(item *models.ErrorProblem)`

| 项目 | 解释 |
|------|------|
| 所在层 | service 辅助函数 |
| 谁调用它 | `GetAllErrors` |
| 参数 | 错题指针 |
| 返回值 | 没有，直接修改传入的错题 |
| 作用 | 补齐旧数据里可能缺失的复习字段 |

为什么参数是指针：  
传指针后，函数改的是原来的 `item`。如果不传指针，只是改副本。

为什么要补 `NextReview`：  
旧版本、手动编辑或备份导入的数据可能没有 `next_review`。如果为空，优先从 `Created` 取前 10 位日期，让旧题从创建日进入复习队列；取之前先判断 `len(item.Created) >= 10`，避免短字符串切片 panic；如果连创建日期都不可信，就用今天日期兜底，保证前端排序和每日推送总能拿到合法 `YYYY-MM-DD`。

#### `matchesKeyword(item models.ErrorProblem, keyword string) bool`

| 项目 | 解释 |
|------|------|
| 所在层 | service 辅助函数 |
| 谁调用它 | `GetAllErrors` |
| 参数 | 一条错题 + 关键词 |
| 返回值 | 是否匹配 |
| 作用 | 判断关键词是否出现在题目、标题、错因或标签里 |

为什么要单独拆出来：  
关键词匹配条件比较长，拆成函数后 `GetAllErrors` 更容易读。

#### `listContainsFold(list []string, keyword string) bool`

| 项目 | 解释 |
|------|------|
| 所在层 | service 辅助函数 |
| 谁调用它 | `matchesKeyword`、`GetAllErrors` |
| 参数 | 字符串列表 + 关键词 |
| 返回值 | 是否包含 |
| 作用 | 判断标签列表里有没有包含关键词的标签 |

`Fold` 在这里表示忽略大小写。比如 `ABC` 和 `abc` 会当成一样。

---

### handlers/errors.go：创建和查询

#### `CreateError(c *gin.Context)`

| 项目 | 解释 |
|------|------|
| 所在层 | handler HTTP 层 |
| 谁调用它 | 浏览器请求 `POST /api/errors` |
| 参数 | Gin 请求上下文 |
| 返回值 | 没有 |
| 作用 | 接收前端 JSON，创建错题，返回新 ID |

傻瓜式流程：

1. 用 `ShouldBindJSON` 把请求体解析到 `AddErrorRequest`。
2. 调 `service.CreateError(req)`。
3. service 报错就返回 400。
4. 成功就返回 `{"id": 新ID, "message": "添加成功"}`。

为什么不直接返回完整错题：  
原 FastAPI 版返回的是 `id/message`，为了兼容前端，Go 版也这样返回。

#### `GetErrors(c *gin.Context)`

| 项目 | 解释 |
|------|------|
| 所在层 | handler HTTP 层 |
| 谁调用它 | 浏览器请求 `GET /api/errors?...` |
| 参数 | Gin 请求上下文 |
| 返回值 | 没有 |
| 作用 | 读取 URL 查询参数，返回错题列表 |

为什么返回 `{"errors": errors, "total": len(errors)}`：  
前端 `api.getErrors()` 读取的是 `res.errors` 和 `res.total`。直接返回数组会让前端失效。

---

### service/error_service.go：更新和删除

#### `UpdateError(id int, req models.UpdateErrorRequest) error`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.UpdateError` |
| 参数 | 错题 ID + 更新字段 |
| 返回值 | 错误 |
| 作用 | 找到指定错题，只更新前端传来的字段 |

傻瓜式流程：

1. 读取 `errors.json`。
2. 用 `for i := range errors` 找到 ID 对应的错题。
3. 每个字段都判断是否为 `nil`。
4. 非 `nil` 表示前端传了这个字段，就更新。
5. 如果更新科目，要检查科目是否合法。
6. 如果更新题目，要检查题目不能为空。
7. 保存整个错题列表。
8. 找不到 ID 就返回错误。

为什么用 `for i := range errors`：  
因为要修改切片里的原始元素。`for _, item := range errors` 拿到的是副本，改了也不会写回原切片。

#### `DeleteError(id int) error`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.DeleteError` |
| 参数 | 错题 ID |
| 返回值 | 错误 |
| 作用 | 删除指定 ID 的错题 |

删除方式：  
Go 里常见做法不是“原地删除”，而是创建一个新列表 `remaining`，把不删除的错题放进去，最后保存新列表。

---

### handlers/errors.go：更新和删除

#### `UpdateError(c *gin.Context)`

| 项目 | 解释 |
|------|------|
| 所在层 | handler HTTP 层 |
| 谁调用它 | 浏览器请求 `PUT /api/errors/:id` |
| 参数 | Gin 请求上下文 |
| 返回值 | 没有 |
| 作用 | 解析 URL 里的 ID 和 JSON 请求体，然后调用 service |

`strconv.Atoi` 的作用：  
URL 参数永远是字符串。`Atoi` 把 `"12"` 变成整数 `12`。

#### `DeleteError(c *gin.Context)`

| 项目 | 解释 |
|------|------|
| 所在层 | handler HTTP 层 |
| 谁调用它 | 浏览器请求 `DELETE /api/errors/:id` |
| 参数 | Gin 请求上下文 |
| 返回值 | 没有 |
| 作用 | 解析 ID，调用 service 删除错题 |

为什么删除失败返回 404：  
如果 ID 不存在，对 HTTP 来说就是“资源不存在”，所以用 `StatusNotFound`。

---

### service/error_service.go：复习和标签

#### `ReviewError(id int) (models.ErrorProblem, error)`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.ReviewError` |
| 参数 | 错题 ID |
| 返回值 | 更新后的错题 + 错误 |
| 作用 | 标记一题已复习，并计算下次复习日期 |

傻瓜式流程：

1. 读取所有错题。
2. 找到 ID 对应的错题。
3. `ReviewCount++`，复习次数加 1。
4. 记录 `LastReview` 为当前时间。
5. 更新 `ReviewStage`。
6. 调 `nextReviewDate` 计算下次复习日期。
7. 保存 JSON。
8. 返回更新后的错题。

为什么返回更新后的错题：  
handler 要从里面取 `next_review` 和 `review_count` 返回给前端。

#### `nextReviewDate(reviewCount int) string`

| 项目 | 解释 |
|------|------|
| 所在层 | service 辅助函数 |
| 谁调用它 | `ReviewError` |
| 参数 | 当前复习次数 |
| 返回值 | 下次复习日期字符串 |
| 作用 | 根据艾宾浩斯间隔计算日期 |

例子：

```text
reviewCount = 1 → 1 天后
reviewCount = 2 → 2 天后
reviewCount = 3 → 4 天后
```

如果次数超过数组长度，就一直用最后一个间隔，避免数组越界。

#### `GetAllTags() ([]string, error)`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.GetTags` |
| 参数 | 没有 |
| 返回值 | 标签列表 + 错误 |
| 作用 | 汇总所有错题的题目标签和错因标签，并去重排序 |

为什么用 `map[string]bool` 去重：  
map 的 key 不能重复。把标签当 key 放进去，天然就去重了。

---

### handlers/errors.go：复习和标签

#### `ReviewError(c *gin.Context)`

| 项目 | 解释 |
|------|------|
| 所在层 | handler HTTP 层 |
| 谁调用它 | 浏览器请求 `PUT /api/errors/:id/review` |
| 参数 | Gin 请求上下文 |
| 返回值 | 没有 |
| 作用 | 标记错题复习，并返回下次复习日期 |

返回格式必须包含：

```json
{"next_review": "2026-06-18", "review_count": 1}
```

前端会用这些字段刷新界面。

#### `GetTags(c *gin.Context)`

| 项目 | 解释 |
|------|------|
| 所在层 | handler HTTP 层 |
| 谁调用它 | 浏览器请求 `GET /api/tags` |
| 参数 | Gin 请求上下文 |
| 返回值 | 没有 |
| 作用 | 返回 `{"tags": [...]}` |

为什么不是直接返回数组：  
原前端读取的是 `res.tags`，所以必须包一层对象。

---

### service/daily_service.go

#### `GetDailyPush() (models.DailyPushResult, error)`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.GetDailyPush` |
| 参数 | 没有 |
| 返回值 | 每日推送结果 + 错误 |
| 作用 | 计算首页每日复习提醒数据 |

傻瓜式流程：

1. 调 `GetAllErrors` 读取全部错题。
2. 调 `GetAllSubjects` 读取全部科目。
3. 计算今天日期。
4. 遍历错题，找出今天到期和已经逾期的题。
5. 统计已经复习过的数量。
6. 按下次复习日期和 ID 排序。
7. 读取知识点库。
8. 给每个科目随机挑一个知识点。
9. 根据是否有到期/逾期错题生成建议。
10. 组装成 `DailyPushResult` 返回。

为什么 `next <= today` 可以比较日期：  
日期格式是 `YYYY-MM-DD`，字符串顺序和日期顺序一致，所以可以直接比较。

为什么 `next == ""` 又兜底成今天：  
正常情况下 `GetAllErrors` 已经调用 `normalizeReviewFields` 补过日期，但首页每日推送是关键入口，再兜一次可以防止异常数据让首页空白。空日期按今天处理，含义是"这题应该现在进入复习队列"。

到期和逾期的区别：  
`next <= today` 表示今天需要处理，包含今天到期和以前逾期；`next < today` 才算逾期。这样前端可以分别显示"今日到期数"和"逾期数"。

#### `getKnowledgeBase() map[string][]string`

| 项目 | 解释 |
|------|------|
| 所在层 | service 辅助函数 |
| 谁调用它 | `GetDailyPush` |
| 参数 | 没有 |
| 返回值 | 科目到知识点列表的映射 |
| 作用 | 优先读取 `knowledge.json`，没有就用默认知识点 |

为什么返回默认值：  
第一次运行可能没有 `knowledge.json`，但每日推送仍然应该能正常显示。

为什么随机知识点按科目生成：  
每日推送是首页简报，不只是错题列表。按科目给一条知识点，可以让没有到期错题的科目也有轻量复习提示；如果某个科目没有知识点，就跳过，不影响其他科目。

---

### handlers/daily.go

#### `GetDailyPush(c *gin.Context)`

| 项目 | 解释 |
|------|------|
| 所在层 | handler HTTP 层 |
| 谁调用它 | 浏览器请求 `GET /api/daily-push` |
| 参数 | Gin 请求上下文 |
| 返回值 | 没有 |
| 作用 | 调 service 计算每日推送，然后返回给前端 |

这个 handler 很薄，只有三件事：

1. 调 service。
2. 错误转成 HTTP 500。
3. 成功返回 JSON。

---

### service/settings_service.go

#### `GetTokenInfo() (masked string, configured bool, username string, err error)`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.GetToken` |
| 参数 | 没有 |
| 返回值 | 脱敏 token、是否已配置、用户名、错误 |
| 作用 | 读取设置页需要展示的信息 |

为什么 token 要脱敏：  
Token 是隐私信息，返回给前端时只显示一部分，避免泄露。

脱敏为什么还要返回 `configured`：  
前端不能靠 token 文本判断是否已配置，因为短 token 会显示成 `"***"`，空 token 也可能被 UI 显示成占位符。`configured` 是明确的布尔值，前端用它控制"已配置/未配置"状态。

#### `SetToken(token string) error`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.SetToken` |
| 参数 | 新 token |
| 返回值 | 错误 |
| 作用 | 保存 MinerU Token 到 `config.json` |

如果 token 是空字符串，函数直接返回成功，表示“不修改旧 token”。

为什么空 token 不覆盖旧值：  
设置页可能只想保存用户名，或者 token 输入框为空但并不代表用户要清空。把空字符串当成"不修改"可以避免误删；真正清空 token 走 `DELETE /api/settings/token`，语义更明确。

#### `ClearToken() error`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.DeleteToken` |
| 参数 | 没有 |
| 返回值 | 错误 |
| 作用 | 清空 `config.json` 里的 MinerU Token |

#### `SetUsername(name string) error`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.SetUsername` |
| 参数 | 用户名 |
| 返回值 | 错误 |
| 作用 | 保存用户名 |

#### `loadConfig() (models.Config, error)`

| 项目 | 解释 |
|------|------|
| 所在层 | service 辅助函数 |
| 谁调用它 | 设置相关 service 函数 |
| 参数 | 没有 |
| 返回值 | 配置结构体 + 错误 |
| 作用 | 统一读取 `config.json` |

为什么要拆出来：  
`GetTokenInfo`、`SetToken`、`ClearToken`、`SetUsername` 都要读配置，拆成一个函数避免重复代码。

为什么每次都读文件而不是缓存：  
`config.json` 很小，读写成本低；但配置可能被设置页、手动编辑或其他流程修改。每次读取可以避免缓存过期，保证界面看到的是最新配置。

---

### handlers/settings.go

#### `GetToken(c *gin.Context)`

处理 `GET /api/settings/token`。它调用 `service.GetTokenInfo()`，然后返回：

```json
{"token": "***", "configured": true, "username": "Knock"}
```

#### `SetToken(c *gin.Context)`

处理 `PUT /api/settings/token`。它从 JSON 请求体里取 `token`，交给 service 保存。

#### `DeleteToken(c *gin.Context)`

处理 `DELETE /api/settings/token`。它不需要请求体，只调用 service 清空 token。

#### `SetUsername(c *gin.Context)`

处理 `PUT /api/settings/username`。它从 JSON 请求体里取 `name`，保存到配置文件。

---

### handlers/backup.go

#### `ExportBackup(c *gin.Context)`

| 项目 | 解释 |
|------|------|
| 所在层 | handler HTTP 层 |
| 谁调用它 | 浏览器请求 `GET /api/backup/export` |
| 参数 | Gin 请求上下文 |
| 返回值 | 没有，直接返回 zip 文件 |
| 作用 | 把数据文件打包成 zip 下载 |

傻瓜式流程：

1. 创建一个内存缓冲区 `bytes.Buffer`。
2. 创建 zip 写入器 `zip.NewWriter(&buffer)`。
3. 获取允许备份的文件名列表。
4. 遍历每个文件。
5. 文件存在就加入 zip。
6. 关闭 zip 写入器。
7. 设置下载文件名响应头。
8. 用 `c.Data` 把 zip 字节返回给浏览器。

为什么用内存缓冲区：  
导出备份不一定要先生成临时文件，直接在内存里拼好 zip 返回更简单。

为什么导出只打包白名单文件：  
备份包应该只包含用户数据，不应该把可执行文件、临时文件、日志或未知文件一起打进去。白名单让备份内容稳定，也避免将来目录里出现敏感文件时被意外导出。

#### `ImportBackup(c *gin.Context)`

| 项目 | 解释 |
|------|------|
| 所在层 | handler HTTP 层 |
| 谁调用它 | 浏览器请求 `POST /api/backup/import` |
| 参数 | Gin 请求上下文 |
| 返回值 | 没有，直接返回 JSON |
| 作用 | 读取上传的 zip，校验后覆盖本地数据 |

傻瓜式流程：

1. 用 `io.ReadAll(c.Request.Body)` 读取前端上传的 zip 二进制。
2. 用 `zip.NewReader` 解析 zip。
3. 遍历 zip 里的文件。
4. 只接受白名单里的 JSON 文件。
5. 检查文件大小，避免超大文件。
6. 读取每个 JSON。
7. 用 `json.Unmarshal` 检查 JSON 格式。
8. 用 `validateBackupData` 检查数据结构。
9. 导入前先调用 `saveCurrentBackupSnapshot` 备份当前数据。
10. 把新数据保存到 `data/`。
11. 返回导入成功、导入了哪些文件、快照文件名。

为什么不能调用 `ExportBackup(c)` 做导入前备份：  
`ExportBackup(c)` 会直接给浏览器写下载响应。一个请求只能返回一次响应，所以导入内部必须用 `saveCurrentBackupSnapshot` 这种“只保存文件、不写 HTTP 响应”的函数。

为什么先全部校验再写入：  
如果边读边写，可能出现一半文件已经覆盖、后面文件校验失败的半导入状态。先把所有 JSON 解析和结构校验通过，再统一写入，可以降低数据不一致风险。

#### `sortedBackupNames() []string`

返回白名单文件名，并排序。  
排序不是必须，但能让导出的 zip 文件顺序稳定，方便检查。

#### `addFileToZip(zipWriter *zip.Writer, path string, name string) error`

| 项目 | 解释 |
|------|------|
| 谁调用它 | `ExportBackup`、`saveCurrentBackupSnapshot` |
| 参数 | zip 写入器、真实文件路径、zip 里的文件名 |
| 返回值 | 错误 |
| 作用 | 把一个磁盘文件复制进 zip |

傻瓜式流程：

1. `os.Open(path)` 打开磁盘文件。
2. `defer src.Close()` 函数结束时关闭文件。
3. `zipWriter.Create(name)` 在 zip 里创建一个文件条目。
4. `io.Copy(dst, src)` 把磁盘文件内容复制进 zip 条目。

#### `readZipFile(file *zip.File) ([]byte, error)`

从 zip 里的一个文件读取全部内容。  
它和 `os.ReadFile` 类似，只不过读的是 zip 内部的文件，不是普通磁盘文件。

#### `validateBackupData(name string, data interface{}) error`

| 项目 | 解释 |
|------|------|
| 谁调用它 | `ImportBackup` |
| 参数 | 文件名 + 解析后的 JSON 数据 |
| 返回值 | 错误 |
| 作用 | 防止坏备份覆盖正常数据 |

检查规则：

```text
errors.json 必须是数组，数组里每项必须是对象
subjects.json 必须是数组，数组里每项必须是字符串
config.json 必须是对象
knowledge.json 必须是对象
```

#### `saveCurrentBackupSnapshot(prefix string) (string, error)`

| 项目 | 解释 |
|------|------|
| 谁调用它 | `ImportBackup` |
| 参数 | 备份名前缀，比如 `"pre-import"` |
| 返回值 | 快照路径 + 错误 |
| 作用 | 导入前自动保存一份当前数据 |

为什么要有它：  
导入备份会覆盖当前数据。如果用户上传错文件，有导入前快照就还有回滚机会。

为什么快照放在 `data/backups/`：  
它仍然属于用户数据的一部分，放在数据目录下面方便一起管理，也不会混到项目源码或前端文件里。

---

### service/ocr_service.go

#### `OCRImageBytes(imageBytes []byte, fileName string) (string, error)`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.OCRImage` |
| 参数 | 图片字节 + 文件名 |
| 返回值 | Markdown 文本 + 错误 |
| 作用 | 串起完整 OCR 流程 |

傻瓜式流程：

1. 读取 MinerU Token。
2. 创建 MinerU 批次，拿到上传地址。
3. 把图片 PUT 到上传地址。
4. 轮询 MinerU，等待识别完成。
5. 下载结果 zip。
6. 提取 Markdown。
7. 返回 Markdown 给 handler。

它是 OCR 的“总调度函数”，真正细节拆到下面的小函数里。

#### `getMinerUToken() (string, error)`

先从 `data/config.json` 读取 `mineru_token`。  
如果配置文件没有，再读环境变量 `MINERU_TOKEN`。  
两个都没有就返回错误。

为什么支持环境变量：  
开发调试时可以不写入配置文件，直接临时设置 token。

为什么配置文件优先于环境变量：  
设置页保存 token 后，用户预期立即生效；如果环境变量优先，可能出现设置页显示已保存但实际还在用旧环境变量的困惑。配置为空时再读环境变量，兼顾普通使用和开发调试。

#### `createMinerUBatch(token string, fileName string) (batchID string, uploadURL string, err error)`

| 项目 | 解释 |
|------|------|
| 谁调用它 | `OCRImageBytes` |
| 参数 | MinerU Token + 文件名 |
| 返回值 | 批次 ID、上传 URL、错误 |
| 作用 | 向 MinerU 申请一个 OCR 任务和上传地址 |

为什么不是直接把图片发给 OCR 接口：  
MinerU v4 流程是先申请上传 URL，再把文件上传到这个 URL，最后轮询结果。

#### `pollMinerUResult(token string, batchID string) (string, error)`

| 项目 | 解释 |
|------|------|
| 谁调用它 | `OCRImageBytes` |
| 参数 | Token + 批次 ID |
| 返回值 | 结果 zip 下载地址 + 错误 |
| 作用 | 每 3 秒查询一次 MinerU，最多等 5 分钟 |

为什么要轮询：  
OCR 不是瞬间完成的。提交图片后，服务端需要时间识别，所以要反复查询状态。

为什么要设置 5 分钟超时：  
没有超时就可能无限等待，占住请求和前端 loading。5 分钟给复杂图片足够时间，同时能在服务异常时明确返回错误。

为什么要记录 `lastErr`：  
轮询过程中可能遇到临时网络抖动，所以单次查询失败不一定马上终止；但如果一直失败到超时，就不能只告诉用户“超时”。把最后一次错误一起返回，可以看出到底是 token 失效、HTTP 状态异常，还是 MinerU 返回了业务错误。

#### `queryBatchResult(token string, batchID string) (...)`

查询批次结果。  
如果已经完成，返回 `full_zip_url`。  
如果还没完成，可能返回 `task_id`，后面可以用任务 ID 查得更准。

#### `queryTaskResult(token string, taskID string) (...)`

用单个任务 ID 查询 OCR 状态。  
它是 `queryBatchResult` 的补充查询方式。

#### `downloadAndExtractMarkdown(zipURL string) (string, error)`

| 项目 | 解释 |
|------|------|
| 谁调用它 | `OCRImageBytes` |
| 参数 | MinerU 返回的 zip 下载地址 |
| 返回值 | Markdown 文本 + 错误 |
| 作用 | 下载结果 zip，提取 `full.md`，处理图片引用 |

傻瓜式流程：

1. `http.Get(zipURL)` 下载 zip。
2. `io.ReadAll(resp.Body)` 读到内存。
3. `zip.NewReader` 解析 zip。
4. 找到 `full.md`。
5. 找到 `images/` 下的图片。
6. 把图片转成 base64 data URI。
7. 调 `replaceMarkdownImages` 替换 Markdown 里的图片路径。

为什么要把图片转 base64：  
MinerU 的 Markdown 可能引用 zip 里的本地图片路径。前端无法直接访问 zip 内部图片，所以要嵌成 `data:image/...;base64,...`。

为什么不能保留 `images/x.png` 路径：  
这些图片只存在于下载到后端内存里的 zip 文件中，并不是浏览器可访问的静态文件。转成 data URI 后，Markdown 文本本身就携带图片内容，前端不需要额外文件服务也能显示。

#### `readOCRZipFile(file *zip.File) ([]byte, error)`

读取 OCR 结果 zip 里的某个文件。  
它和备份章节的 `readZipFile` 思路一样。

#### `replaceMarkdownImages(markdown string, imageMap map[string]string) string`

| 项目 | 解释 |
|------|------|
| 谁调用它 | `downloadAndExtractMarkdown` |
| 参数 | Markdown 文本 + 图片路径到 base64 的映射 |
| 返回值 | 替换后的 Markdown |
| 作用 | 把 `![...](images/x.png)` 替换成 `<img src="data:image/...">` |

为什么用正则：  
Markdown 图片语法有固定格式：`![说明](路径)`。正则可以把里面的路径抓出来。

---

### handlers/ocr.go

#### `OCRImage(c *gin.Context)`

| 项目 | 解释 |
|------|------|
| 所在层 | handler HTTP 层 |
| 谁调用它 | 浏览器请求 `POST /api/ocr` |
| 参数 | Gin 请求上下文 |
| 返回值 | 没有 |
| 作用 | 读取前端上传的图片字节，调用 OCR service |

关键点：  
前端发的是原始 Blob，不是表单。所以这里必须用：

```go
io.ReadAll(c.Request.Body)
```

不能用：

```go
c.FormFile("file")
```

返回格式必须是：

```json
{"markdown": "..."}
```

因为前端读取的是 `res.markdown`。

---

### service/update_service.go

#### `GetVersionResponse() map[string]interface{}`

返回当前版本信息。  
前端设置页打开时会调用 `/api/version`，如果这个接口不存在，设置页就会报错。

#### `CheckUpdate(force bool) map[string]interface{}`

| 项目 | 解释 |
|------|------|
| 所在层 | service 业务层 |
| 谁调用它 | `handlers.CheckUpdate` |
| 参数 | `force`，是否强制检查 |
| 返回值 | 更新检查结果 |
| 作用 | 第一阶段先返回“没有更新”，保持前端不报错 |

为什么现在不真的检查 GitHub：  
自动更新涉及下载 Release、校验 zip、启动 Updater.exe、退出当前程序，复杂度高。Go 迁移第一阶段先保证接口兼容。

#### `ApplyUpdate() map[string]interface{}`

返回“Go 版暂不支持自动替换”。  
这个函数是为了让前端点击更新按钮时得到正常 JSON，而不是 404。

#### `loadVersionInfo() VersionInfo`

读取 `version.json`。  
如果文件不存在，就返回默认版本 `0.0.0-dev`。

---

### handlers/update.go

#### `GetVersion(c *gin.Context)`

处理 `GET /api/version`。  
它只做一件事：把 `service.GetVersionResponse()` 返回给前端。

#### `CheckUpdate(c *gin.Context)`

处理 `GET /api/update/check?force=true`。  
它读取 `force` 查询参数，然后交给 service。

#### `ApplyUpdate(c *gin.Context)`

处理 `POST /api/update/apply`。  
当前只返回“暂不支持自动替换”，但接口存在，前端不会炸。

---

## 写代码时最重要的调用链

你可以把整个后端理解成这些链条：

```text
浏览器 GET /api/subjects
→ handlers.GetSubjects
→ service.GetAllSubjects
→ store.LoadJSON("subjects.json")
→ 返回 {"subjects": [...]}
```

```text
浏览器 POST /api/errors
→ handlers.CreateError
→ service.CreateError
→ store.LoadJSON("errors.json")
→ store.SaveJSON("errors.json")
→ 返回 {"id": 1, "message": "添加成功"}
```

```text
浏览器 POST /api/ocr
→ handlers.OCRImage
→ service.OCRImageBytes
→ MinerU API
→ 返回 {"markdown": "..."}
```

你写代码时只要记住：

```text
handler 负责 HTTP
service 负责业务逻辑
store 负责文件读写
models 负责数据长什么样
```

就不容易乱。

## 下一步

后端迁移完成后，你可以：

1. **写单元测试**：给 service 层加 `_test.go` 文件
2. **替换 JSON 存储为 SQLite**：只需改 store 层，handler 和 service 不用动
3. **添加认证**：在 Gin 中间件中实现 JWT 验证
4. **Docker 部署**：把编译好的 exe + data 目录打包成镜像
