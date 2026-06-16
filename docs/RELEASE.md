# 打包发布说明

本项目发布包是 Windows 下的 `Tracker.zip`。自动更新依赖 GitHub Releases 中的 `Tracker.zip` 资源名，不要改名上传。

## 本地打包

```powershell
.\scripts\build-release.ps1
```

指定版本：

```powershell
.\scripts\build-release.ps1 -Version 2026.06.16-1223
```

脚本会：

1. 安装并构建前端。
2. 使用 PyInstaller 构建 `Tracker.exe`。
3. 使用 PyInstaller 构建 `Updater.exe`。
4. 写入发布包内的 `version.json`。
5. 生成 `dist/release/Tracker.zip`。

发布包内容：

```text
Tracker.exe
Updater.exe
version.json
README.md
README-release.txt
```

## 上传 GitHub Release

需要先登录 GitHub CLI：

```powershell
gh auth login
```

打包并上传：

```powershell
.\scripts\build-release.ps1 -Version 2026.06.16-1223 -Upload
```

脚本会创建或更新：

```text
tag: v2026.06.16-1223
asset: Tracker.zip
```

自动更新检查的是 latest release，并在其中查找 `Tracker.zip`。

## 使用 push.bat

本机 `push.bat` 会调用 `push.ps1`：

```powershell
.\push.bat
```

它会：

1. 生成当前时间版本号。
2. 写入根目录 `version.json`。
3. 提交并推送代码变更。
4. 调用 `scripts/build-release.ps1 -Upload`。

注意：`push.*` 当前被 `.gitignore` 忽略，这是本机辅助脚本，不会提交到仓库。

## 版本号规则

推荐版本号格式：

```text
2026.06.16-1223
```

发布脚本会把传入版本写入发布包内的：

```text
version.json
```

示例：

```json
{
  "version": "2026.06.16-1223",
  "repo": "Zilvren/Learning-Assitant",
  "asset_name": "Tracker.zip",
  "app_exe": "Tracker.exe"
}
```

自动更新会比较本地 `version.json` 和 GitHub latest release tag。只有远端版本大于本地版本时才提示更新。

## 自动更新测试

1. 用旧版本 `Tracker.zip` 解压到一个全新的测试目录。
2. 双击 `Tracker.exe`。
3. 打开“设置”，点击“检查更新”。
4. 发现新版本后点击“立即更新”。
5. 等待原网页自动刷新。

检查更新日志：

```text
data/updates/update.log
```

成功日志应包含：

```text
Payload version ...
Replaced version.json
Installed version ...
Update installed successfully
```

确认数据仍在：

```text
data/errors.json
data/subjects.json
```

## 常见发布问题

### 最新版本下载后版本号没变

检查三个位置：

- GitHub Release 中 `Tracker.zip` 内的 `version.json`。
- 程序目录下的 `version.json`。
- `data/updates/update.log`。

如果日志中没有 `Update installed successfully`，说明 updater 没有完成替换。

### 自动更新找不到更新包

确认 latest release 中资源名必须是：

```text
Tracker.zip
```

### 本地 zip 版本和解压目录版本不一样

`dist/release/Tracker.zip` 是压缩包本体，`dist/release/Tracker/` 可能是旧解压目录。测试时建议解压到一个全新目录。
