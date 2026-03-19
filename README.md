# media-dl

命令行播客/视频下载工具，支持提取音频并转换 shownotes 图片为 markdown
专为 OpenClaw 开发

## 工作原理

```
┌──────────┐    ┌────────────┐    ┌──────────────────┐
│  输入 URL │───▶│ 平台识别   │───▶│  截取音频 (.m4a) │
└──────────┘    │ (油管/B站/  │    │  + 元数据        │
                │  小宇宙)    │    │  + Markdown 图片 │
                └────────────┘    └──────────────────┘
```

输入 URL → 自动识别平台 → yt-dlp 下载音频 → shownotes 图片转换为 markdown 格式

## 支持的平台

| 平台 | 类型 | 说明 |
|------|------|------|
| YouTube | 视频/音乐 | 提取音频 |
| Bilibili | 视频 | 提取音频 |
| 小宇宙FM | 播客 | 含 shownotes Markdown 图片 |

## 前置依赖

1. **Go 1.25+** - 编译工具
2. **yt-dlp** - 音视频下载核心
   - macOS: `brew install yt-dlp`
   - pip: `pip install yt-dlp`
## 安装

### 从源码构建

```bash
git clone https://github.com/slarkio/media-dl.git
cd media-dl
go build -o ~/.agents/tools/media-dl
```

### 通过 go install

```bash
go install github.com/slarkio/media-dl
```

## 使用

```bash
media-dl <url>                  # 下载音频 + shownotes
media-dl <url> -o ./out         # 指定输出目录
media-dl <url> -c ./cookies.txt  # 带 Cookie（VIP 内容）
media-dl <url> --json           # JSON 输出
media-dl <url> -v               # 调试模式
media-dl <url> -a               # 仅下载音频
media-dl <url> -s               # 仅下载 shownotes
```

## 命令行选项

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `-o, --output` | 输出目录 | 当前目录 |
| `-c, --cookie` | Cookie 文件路径 | 无 |
| `-j, --json` | JSON 格式输出 | false |
| `-v, --verbose` | 调试信息 | false |
| `-a, --audio-only` | 仅下载音频，跳过 shownotes | false |
| `-s, --shownotes-only` | 仅下载 shownotes，跳过音频 | false |

## 输出文件

下载完成后，当前目录会生成：

| 文件 | 说明 |
|------|------|
| `*.m4a` | 音频文件 |
| `shownotes.md` | 小宇宙播客专有，包含 Markdown 格式图片 |

## 示例

```bash
# 下载小宇宙播客
media-dl https://www.xiaoyuzhoufm.com/episode/abc123

# 下载 YouTube 视频
media-dl https://www.youtube.com/watch?v=xyz123

# 下载 Bilibili 视频
media-dl https://www.bilibili.com/video/BV123456
```

## 常见问题

### yt-dlp 未安装

如果运行时报错找不到 `yt-dlp`，请先安装：

- macOS: `brew install yt-dlp`
- Linux: `pip install yt-dlp`

### Cookie 文件路径必须为绝对路径

使用 `-c` 参数时，Cookie 文件路径必须为绝对路径，不能使用相对路径。例如：

- ❌ `-c ./cookies.txt`
- ✅ `-c /home/user/cookies.txt`

### 不支持的平台 URL

目前支持以下平台的 URL：

| 平台 | 支持的 URL 格式 |
|------|----------------|
| YouTube | `https://www.youtube.com/watch?v=xxx` |
| Bilibili | `https://www.bilibili.com/video/BVxxx` |
| 小宇宙 | `https://www.xiaoyuzhoufm.com/episode/xxx` |

### b23.tv 短链接

Bilibili 的 `b23.tv` 短链接会自动解析为完整的 Bilibili URL，无需用户手动处理。
