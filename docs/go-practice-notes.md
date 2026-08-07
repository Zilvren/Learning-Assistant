# Go 知识点笔记

记录自 OCR 后端代码学习过程中的知识点。

---

## 目录

1. [MaxBytesReader](#maxbytesreader)
2. [errors.As](#errorsas)
3. [OCRImageBytes 函数详解](#ocrimagebytes-函数详解)
4. [Blob](#blob)
5. [bytes.NewReader](#bytesnewreader)
6. [http.Client.Do](#httpclientdo)
7. [http.StatusCreated](#httpstatuscreated)
8. [json.NewDecoder vs json.Unmarshal](#jsonnewdecoder-vs-jsonunmarshal)
9. [io.ReadAll](#ioreadall)
10. [io.Reader](#ioreader)
11. [queryTaskResult & downloadAndExtractMarkdown](#querytaskresult--downloadandextractmarkdown)

---

## MaxBytesReader

**所在位置：** [api/handlers/ocr.go:18](api/handlers/ocr.go#L18)

```go
c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, ocrMaxUploadSize)
// ocrMaxUploadSize = 200 * 1024 * 1024 (200MB)
```

### 是什么

`http.MaxBytesReader` 是 Go 标准库 `net/http` 提供的函数，用来**限制 HTTP 请求体的最大读取字节数**，防止客户端发送超大请求导致服务器内存爆掉（DoS 攻击防护）。

### 参数

| 参数 | 含义 |
|------|------|
| `c.Writer` | `http.ResponseWriter`，超限时自动写错误响应并关闭连接 |
| `c.Request.Body` | 原始的请求体，被包装成有上限的 Reader |
| `ocrMaxUploadSize` | 上限值（200MB） |

### 工作原理

它把原来的 `c.Request.Body` 替换成一个带计数器的 wrapper。后续用 `io.ReadAll` 读取时，一旦累计字节数超过上限，`ReadAll` 就会返回 `*http.MaxBytesError` 类型的错误。

```go
// 超过限制后会返回 MaxBytesError
var sizeErr *http.MaxBytesError
if errors.As(err, &sizeErr) {
    c.JSON(http.StatusBadRequest, gin.H{"detail": "图片文件不能超过 200MB"})
    return
}
```

**一句话：上传限流阀，超过 200MB 直接掐断。**

---

## errors.As

**所在位置：** [api/handlers/ocr.go:22](api/handlers/ocr.go#L22)

```go
var sizeErr *http.MaxBytesError
if errors.As(err, &sizeErr) {
    // err 是 *http.MaxBytesError 类型
}
```

### 是什么

`errors.As` 是 Go 标准库 `errors` 包的函数，用于**检查一个 error 是否是某个特定类型**，并**把该类型的值提取出来**。

### 参数

| 参数 | 含义 |
|------|------|
| `err` | 要检查的错误 |
| `&sizeErr` | 指向目标类型变量的指针 |

### 与 errors.Is 的区别

```go
errors.Is(err, io.EOF)         // err 是不是 io.EOF 这个具体的值？
errors.As(err, &sizeErr)       // err 是不是 *http.MaxBytesError 这个类型？
```

| 函数 | 用途 | 类比 |
|------|------|------|
| `errors.Is` | 判断是否**某个具体的错误值** | "这个人是不是张三？" |
| `errors.As` | 判断是否**某个错误类型** | "这个人是不是人类？" |

**一句话：类型断言版的错误检查。**

---

## OCRImageBytes 函数详解

**所在位置：** [internal/service/ocr_service.go:33](internal/service/ocr_service.go#L33)

```go
func OCRImageBytes(ctx context.Context, imageBytes []byte, fileName string) (string, error)
```

### 功能

对图片字节数据进行 OCR 识别，返回 Markdown 文本。

### 完整流程（7步）

```
handler (文件上传)
    │
    ▼
OCRImageBytes ──→ ① 创建 task 记录 (pending)
    │
    ├─→ ② 获取 MinerU token
    │      失败 → markOCRFailed → return err
    │
    ├─→ ③ 创建 MinerU 批次 → batchID + uploadURL
    │      失败 → markOCRFailed → return err
    │
    ├─→ ④ 上传图片到预签名 URL
    │      失败 → markOCRFailed → return err
    │
    ├─→ ⑤ 轮询 5 分钟等结果 → zipURL
    │      失败 → markOCRFailed → return err
    │
    ├─→ ⑥ 下载 zip → 提取 full.md → 图片转 base64 → 替换图片链接
    │      失败 → markOCRFailed → return err
    │
    └─→ ⑦ 更新任务状态为 succeeded → 返回 markdown
```

### 关键设计

| 设计 | 原因 |
|------|------|
| **预签名上传** | 图片不经过自己的服务器中转，节省带宽和内存 |
| **每 3 秒轮询，最长 5 分钟** | MinerU 是异步处理，不能同步等 |
| **图片转 base64 内嵌** | 前端拿到 Markdown 就能直接显示，不用再发请求下载图片 |
| **`markOCRFailed` 统一处理** | 不管哪步失败都记到数据库，方便排查 |
| **部分 Update 错误被忽略（`_`）** | Update 失败不影响主流程 |

---

## Blob

### 是什么

Blob（Binary Large Object）是前端 JavaScript 的一种数据类型，用来表示**原始的二进制数据**（图片、文件等）。

### 为什么在 OCR 的代码里提到它

[api/handlers/ocr.go:16](api/handlers/ocr.go#L16) 的注释：

```go
// 注意：前端发的是原始图片 Blob，不是 multipart/form-data
```

### 前端两种传文件的方式

```javascript
// ❌ 传统方式（multipart/form-data）
const form = new FormData()
form.append('file', fileInput.files[0])
fetch('/api/ocr', { method: 'POST', body: form })

// ✅ 本项目的用法（直接发 Blob）
const blob = fileInput.files[0]  // File 继承自 Blob
fetch('/api/ocr', {
  method: 'POST',
  body: blob,         // 直接发二进制，没有 multipart 包装
  headers: { 'Content-Type': 'image/png' }
})
```

### 后端因此怎么做

没有用 `c.FormFile()` 或 `c.MultipartForm()`，而是：

```go
body, err := io.ReadAll(c.Request.Body)  // 直接读原始请求体
```

因为请求体里就是图片本身的二进制字节，没有 multipart 分层。

| 概念 | 说明 |
|------|------|
| **Blob** | JS 中的二进制数据容器（文件、图片、任意字节） |
| **在哪看到的** | `ocr.go` 注释，提醒别用 `FormFile` 去读 |
| **为什么重要** | 两种传法后端解析方式完全不同 |

---

## bytes.NewReader

**所在位置：** [internal/service/ocr_service.go:63](internal/service/ocr_service.go#L63)

```go
req, err := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(imageBytes))
```

### 是什么

`bytes.NewReader` 把 `[]byte`（字节切片）包装成一个 `io.Reader`，让字节切片可以用标准库的 `Read` 方法来读取。

### 为什么需要它

`http.NewRequest` 的第三个参数 `body` 要求是 `io.Reader` 类型，但我们手里是 `[]byte`。`bytes.NewReader` 就是做这个类型适配的：

```go
// 你有 []byte，但接口要 io.Reader
data := []byte{...}
reader := bytes.NewReader(data)  // []byte → io.Reader
```

### 同类函数

| 函数 | 作用 |
|------|------|
| `bytes.NewReader(b)` | `[]byte` → `io.Reader`（只读，可 Seek） |
| `bytes.NewBuffer(b)` | `[]byte` → `io.Reader` + `io.Writer`（可读写） |
| `strings.NewReader(s)` | `string` → `io.Reader` |

**一句话：把内存里的字节切片伪装成"文件流"，让需要 io.Reader 的地方能读取它。**

---

## http.Client.Do

**所在位置：** [internal/service/ocr_service.go:67](internal/service/ocr_service.go#L67)

```go
resp, err := ocrHTTPClient.Do(req)
```

### 是什么

`Client.Do` 是 `net/http` 包的方法，用于**执行一个 HTTP 请求并获取响应**。

### 本项目的 HTTP 客户端

```go
var ocrHTTPClient = &http.Client{Timeout: 60 * time.Second}
```

自定义客户端，设置了 **60 秒超时**。没使用 `http.DefaultClient`（没有超时限制，可能卡死）。

### 返回值

| 返回值 | 含义 |
|--------|------|
| `resp` | `*http.Response` — 状态码、响应头、响应体 |
| `err` | 网络层面错误（DNS 失败、连接拒绝、超时等） |

### 响应处理

```go
resp.Body.Close()                    // 关掉响应体，防连接泄漏
if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
    // 非 200/201 → 上传失败
}
```

**为什么只关 Body 不读？** PUT 到预签名 URL 的上传响应通常没有 body，只看状态码就行。

**一句话：`Do` = 把构建好的请求发出去，拿回服务器的响应。**

---

## http.StatusCreated

**所在位置：** [internal/service/ocr_service.go:73](internal/service/ocr_service.go#L73)

```go
if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
```

### 是什么

`http.StatusCreated` = **201**，表示"资源已成功创建"。

### 为什么同时检查 200 和 201

不同的云存储服务对预签名 PUT 上传的响应习惯不同：
- 有的返回 `200 OK`（"收到了，处理完了"）
- 有的返回 `201 Created`（"成功创建了新文件"）

**同时接受两者，更加健壮。**

### 常见 HTTP 状态码

| 常量 | 数值 | 含义 |
|------|------|------|
| `http.StatusOK` | 200 | 请求成功 |
| `http.StatusCreated` | 201 | 创建成功 |
| `http.StatusNoContent` | 204 | 成功但无返回内容 |
| `http.StatusBadRequest` | 400 | 请求参数错误 |
| `http.StatusUnauthorized` | 401 | 未认证 |
| `http.StatusNotFound` | 404 | 找不到资源 |
| `http.StatusInternalServerError` | 500 | 服务器内部错误 |
| `http.StatusServiceUnavailable` | 503 | 服务暂不可用 |

---

## json.NewDecoder vs json.Unmarshal

**所在位置：** [internal/service/ocr_service.go:242](internal/service/ocr_service.go#L242)

```go
json.NewDecoder(resp.Body).Decode(&result)
```

### 是什么

这行是**三步合写**：

```go
decoder := json.NewDecoder(resp.Body)  // ① 创建解码器
err := decoder.Decode(&result)         // ② 解析 JSON 到结构体
if err != nil { ... }                  // ③ 处理错误
```

### 两种 JSON 解析方式对比

```go
// 方式一：json.Unmarshal（适合 []byte）
data, _ := io.ReadAll(resp.Body)
json.Unmarshal(data, &result)

// 方式二：json.NewDecoder.Decode（适合 io.Reader 流）
json.NewDecoder(resp.Body).Decode(&result)
```

| | `json.Unmarshal` | `json.NewDecoder.Decode` |
|--|------------------|------------------------|
| 输入 | `[]byte` | `io.Reader`（流式） |
| 内存占用 | 需要完整字节切片 | 流式解析，可处理大 JSON |
| 适用场景 | 已有 `[]byte` 时 | 从文件/网络读入时 |
| 代码行数 | 3 行 | 1 行 |

### 本项目的两种用法

第150行用 `ReadAll` + `Unmarshal`（因为后面还要用 `data` 变量）：

```go
data, err := io.ReadAll(resp.Body)
json.Unmarshal(data, &result)
```

第242行用 `NewDecoder`（只需要解析 JSON，不需要原始字节）：

```go
json.NewDecoder(resp.Body).Decode(&result)
```

**推荐：从 HTTP 响应解析 JSON 时，优先用 `json.NewDecoder(resp.Body).Decode(&result)`，更简洁高效。**

---

## io.ReadAll

**所在位置：** [internal/service/ocr_service.go:150](internal/service/ocr_service.go#L150)

```go
data, err := io.ReadAll(resp.Body)
```

### 是什么

`io.ReadAll` 从 `io.Reader` 里把**所有数据读完**，返回 `[]byte`。

### 为什么需要它

网络流（`io.Reader`）是"用完即走"的，不能回头再读。如果不一次性读完存到变量里，后面的代码就没法用了。

### 类比

```
io.Reader = 自助餐的菜，边吃边消失，没法回头夹
io.ReadAll(reader) = 一次性把菜全夹到碗里，存起来慢慢吃
```

### 常见的 Reader 来源

| 类型 | 数据来源 |
|------|---------|
| `resp.Body` | HTTP 响应（网络） |
| `os.File` | 磁盘文件 |
| `bytes.NewReader(data)` | 内存中的 `[]byte` |
| `strings.NewReader(s)` | 字符串 |
| `os.Stdin` | 标准输入（键盘） |

---

## io.Reader

### 是什么

`io.Reader` 是 Go 标准库 `io` 包的一个**接口**，定义了"读取数据"这件事的通用抽象。

### 定义

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}
```

只要一个类型实现了 `Read` 方法，它就实现了 `io.Reader` 接口。

### 核心思想

**把"读数据"这个行为抽象成接口，不管底层数据源是什么，代码都能用同样的方式去读。**

### 常见的 io.Reader 实现

| 类型 | 读取来源 |
|------|---------|
| `*bytes.Reader` | 内存里的 `[]byte` |
| `*strings.Reader` | 内存里的字符串 |
| `resp.Body` | 网络连接 |
| `os.File` | 磁盘文件 |
| `*zip.File` | 压缩包里的文件 |

### 关键特性：流

**io.Reader 是流（stream）**，数据流过 Reader 就不能回头：

```go
data1, _ := io.ReadAll(reader)  // 读完了
data2, _ := io.ReadAll(reader)  // 什么也读不到了，EOF
```

就像看视频直播——你不能把进度条往回拖。

---

## queryTaskResult & downloadAndExtractMarkdown

**所在位置：** [internal/service/ocr_service.go:258](internal/service/ocr_service.go#L258)

### queryTaskResult — 查单个任务状态

```go
func queryTaskResult(token string, taskID string) (zipURL string, state string, err error)
```

#### 功能

调 MinerU API 查某个具体任务的识别状态（和 `queryBatchResult` 查批次不同，这个是查批次内的**某一个任务**）。

#### 返回值

| 返回值 | 含义 |
|--------|------|
| `zipURL` | 结果 zip 下载地址（仅 `state == "done"` 时有值） |
| `state` | 状态：`"done"` / `"processing"` / `"failed"` |
| `err` | 网络/解析错误 |

#### 流程

1. **GET** 请求 → `{baseURL}/extract/task/{taskID}`
2. 带 Bearer Token 认证
3. 解析 JSON，检查 `code == 0`
4. `state == "done"` → 返回 zip 下载链接
5. 否则返回 `""` + 当前 state（让调用方继续轮询）

---

### downloadAndExtractMarkdown — 下载并提取 Markdown

**所在位置：** [internal/service/ocr_service.go:293](internal/service/ocr_service.go#L293)

```go
func downloadAndExtractMarkdown(zipURL string) (string, error)
```

#### 功能

OCR 流程的**终点**：下载结果 zip，解析出 Markdown，把图片转 base64 内嵌。

#### 完整步骤

**Step 1 — 下载 zip 包（限流 200MB）：**

```go
data, err := io.ReadAll(io.LimitReader(resp.Body, ocrResultZipMaxSize+1))
```

用 `io.LimitReader` 限制下载大小不超过 200MB + 1 字节，超过即报错。**防爆机制。**

**Step 2 — 解压：**

```go
reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
```

**Step 3 — 遍历 zip 文件：**

| 文件类型 | 处理方式 | 大小限制 |
|----------|---------|---------|
| `full.md` | 读取内容，提取 Markdown | 5MB |
| `images/` 下的图片 | 转成 base64 data URI | 单张 12MB，总计 24MB |

图片编码示例：

```go
imageMap[file.Name] = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(content)
// → data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...
```

**Step 4 — 替换 Markdown 中的图片引用：**

```go
return replaceMarkdownImages(markdown, imageMap), nil
```

原文：
```markdown
![示意图](images/fig1.png)
```
替换后：
```markdown
![示意图](data:image/png;base64,iVBORw0KGgoAAA...)
```

#### 为什么要转 base64？

因为前端只需要一个 Markdown 字符串就能展示全部内容（文本 + 图片），不需要额外发请求下载图片。

#### 三层限流

```
       200MB                5MB              12MB         24MB
  resultZipMaxSize    markdownMaxSize    imageMaxSize    totalImageMaxSize
  ┌────────────┐     ┌─────────────┐   ┌──────────┐    ┌──────────────┐
  │ 整个 zip 包  │     │ full.md 文件  │   │ 单张图片   │    │ 所有图片总和   │
  └────────────┘     └─────────────┘   └──────────┘    └──────────────┘
```

**三层限流，防止 MinerU 返回恶意数据撑爆内存。**

---

## Go 基础语法速记

### `:=` vs `=`

| 符号 | 用途 |
|------|------|
| `:=` | 声明并赋值新变量（左侧至少有一个新变量） |
| `=` | 给已有变量赋值 |

```go
x := 10      // 新变量 x
x = 20       // 已有变量 x
x, y := 1, 2 // x 已有，y 新变量（可以用 :=）
```

### `_`（下划线 / 空标识符）

用于忽略不需要的返回值：

```go
taskID, _ := repos.OCRTasks.Create(...)  // 忽略 error
_ = repos.OCRTasks.Update(...)            // 忽略所有返回值
```

### `defer`

延迟执行，函数返回前一定会执行。常用于关闭资源：

```go
defer resp.Body.Close()
// 等价于在函数 return 前自动执行 resp.Body.Close()
```

### 错误处理模式

Go 没有 try-catch，错误通过返回值传递：

```go
value, err := someFunc()
if err != nil {
    // 处理错误
    return "", err
}
// 正常使用 value
```

### 结构体内嵌

```go
var result struct {
    Code int    `json:"code"`
    Data struct {
        State string `json:"state"`
    } `json:"data"`
}
```

这种写法叫**匿名结构体**，不用提前定义类型，一次性使用很方便。

### `:=` 在 if 里的用法

```go
if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
    return "", "", "", err
}
```

`err` 的作用域仅限于这个 `if-else` 块，不影响外层变量。
