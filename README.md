# Dify 文件上传工作流封装服务

## 新增：仅上传URL文件到Dify接口

我们新增了一个专门用于将URL文件上传到Dify的接口，不调用工作流：

```
POST /dify/upload/url
```

请求头:
```
Authorization: Bearer <your_dify_api_key>
Content-Type: application/json
```

请求体示例:
```json
{
  "file_url": "https://example.com/path/to/file.pdf",
  "user": "username",
  "domain": "http://dify.example.com"
}
```

响应示例:
```json
{
  "code": 200,
  "message": "上传成功",
  "data": {
    "id": "文件ID",
    "name": "文件名",
    "size": 文件大小,
    "extension": "文件扩展名",
    "mime_type": "文件MIME类型",
    "created_by": "创建者ID",
    "created_at": 创建时间戳
  }
}
```

这个接口非常适合需要先上传文件然后再在其他请求中使用文件ID的场景。

## 异步请求与回调机制

本服务支持**异步请求**，适用于大文件或耗时任务。用户可在请求中指定回调地址，服务端处理完成后会自动回调结果。

### 异步请求用法

- 在请求体（JSON或form-data）中增加 `async` 字段（或form参数`callback_url`），格式如下：

```json
"async": {
  "callback_url": "https://your-callback-url.com/webhook", // 必填，回调地址
  "request_id": "可选自定义ID" // 可选，不填则自动生成唯一ID
}
```

- 服务端收到异步请求后，**立即返回**一个唯一ID（202 Accepted），并在后台处理任务。

```json
{
  "code": 202,
  "message": "请求已接受，正在异步处理",
  "data": {
    "request_id": "唯一请求ID",
    "status": "pending",
    "message": "请求已接收，正在处理中"
  }
}
```

- 处理完成后，服务端会以POST方式将完整结果回调到`callback_url`。
- 用户可通过 `/dify/async/:requestID` 查询处理进度。

### 异步回调数据格式

```json
{
  "request_id": "唯一请求ID",
  "result": { /* 处理结果，结构同同步接口返回 */ },
  "timestamp": 1710000000
}
```

### 查询异步请求状态

```
GET /dify/async/:requestID
```
返回：
```json
{
  "code": 200,
  "message": "成功",
  "data": {
    "request_id": "唯一请求ID",
    "status": "pending|processing|completed|failed",
    "message": "状态描述"
  }
}
```

### 流式响应与异步请求组合

本服务现已支持同时使用流式响应(streaming)模式和异步请求。当您设置`response_mode="streaming"`并同时提供`callback_url`时，系统会自动将请求转为blocking模式进行处理，确保能够正确解析响应并回调结果。

---

这是一个用Go语言实现的，封装了Dify平台文件上传和工作流API的服务。

## 功能特性

- 支持两种格式上传文件：form-data直接上传、URL文件下载上传
- 支持批量上传，最多可上传10个文件
- 支持单文件和文件列表两种工作流调用方式
- 支持streaming和blocking两种响应模式
- 支持异步请求和回调机制
- 支持仅上传文件获取ID，不调用工作流
- 完整的错误处理和响应
- 自动创建工作流所需的文件变量，无需在请求中提前定义

## 鉴权方式

所有API请求需要在请求头中添加`Authorization`头部，格式为：

```
Authorization: Bearer <your_dify_api_key>
```

其中`<your_dify_api_key>`是您在Dify平台获取的API密钥。

## API接口

### 单文件URL格式

```
POST /dify/fileSingle/workflow
```

请求头:
```
Authorization: Bearer <your_dify_api_key>
Content-Type: application/json
```

请求体示例:

```json
{
    "domain": "http://dify.example.com",
    "inputs": {
        "file": {
            "file_url": "https://example.com/sample.pdf",
            "file_value": "file_single"
        },
        "key1": "value1",
        "key2": "value2"
    },
    "response_mode": "blocking",
    "user": "username",
    "async": {
        "callback_url": "https://your-callback-url.com/webhook",
        "request_id": "optional-custom-id"
    }
}
```

> 注意：`file_value` 指定文件在工作流中的变量名，系统会自动创建该变量，无需在 `inputs` 中重复添加。

### 多文件URL格式

```
POST /dify/files/workflow
```

请求头:
```
Authorization: Bearer <your_dify_api_key>
Content-Type: application/json
```

请求体示例:

```json
{
    "domain": "http://dify.example.com",
    "inputs": {
        "file": {
            "file_urls": [
                "https://example.com/file1.pdf",
                "https://example.com/file2.txt"
            ],
            "file_value": "file_list"
        },
        "key1": "value1"
    },
    "response_mode": "blocking",
    "user": "username"
}
```

> 注意：`file_value` 指定文件列表在工作流中的变量名，系统会自动创建该变量，无需在 `inputs` 中重复添加。

### 单文件Form-data格式

```
POST /dify/fileSingle/formdata/workflow
```

请求头:
```
Authorization: Bearer <your_dify_api_key>
```

表单参数:

- `domain`: Dify服务器域名
- `file`: 上传的文件
- `file_value`: 文件在工作流中的映射键名
- `user`: 用户标识
- `response_mode`: 响应模式 (blocking/streaming)
- `callback_url`: 异步回调地址 (可选)
- `request_id`: 自定义请求ID (可选)
- 其他参数将作为inputs传递给工作流

### 多文件Form-data格式

```
POST /dify/files/formdata/workflow
```

请求头:
```
Authorization: Bearer <your_dify_api_key>
```

表单参数:

- `domain`: Dify服务器域名
- `files`: 上传的多个文件
- `file_value`: 文件在工作流中的映射键名
- `user`: 用户标识
- `response_mode`: 响应模式 (blocking/streaming)
- `callback_url`: 异步回调地址 (可选)
- `request_id`: 自定义请求ID (可选)
- 其他参数将作为inputs传递给工作流

### 仅上传URL文件到Dify

```
POST /dify/upload/url
```

请求头:
```
Authorization: Bearer <your_dify_api_key>
Content-Type: application/json
```

请求体示例:
```json
{
  "file_url": "https://example.com/path/to/file.pdf",
  "user": "username",
  "domain": "http://dify.example.com"
}
```

### 流式响应接口

所有主要接口都支持流式响应，只需设置 `response_mode=streaming` 参数：

- 单文件URL流式: 使用 `/dify/fileSingle/workflow` 并设置 `response_mode=streaming`
- 多文件URL流式: 使用 `/dify/files/workflow` 并设置 `response_mode=streaming`
- 单文件表单流式: 使用 `/dify/fileSingle/formdata/workflow` 并设置 `response_mode=streaming`
- 多文件表单流式: 使用 `/dify/files/formdata/workflow` 并设置 `response_mode=streaming`

## 部署说明

### 环境要求

- **Go版本**: 1.24 (与Dockerfile保持一致)
- **操作系统**: 支持Linux, Windows, macOS

### 环境变量

- `PORT`: 服务端口，默认3010
- `MAX_UPLOAD_FILES`: 最大上传文件数量，默认10
- `DEFAULT_USER`: 默认用户名，默认"user"
- `API_TIMEOUT`: API超时时间（秒），默认120秒

### Docker部署

```bash
# 构建镜像
docker build -t dify-upload-workflow:latest .

# 运行容器
docker run -d -p 3010:3010 --name dify-upload-workflow dify-upload-workflow:latest
```

### 使用docker-compose部署

```bash
docker-compose -f docker-compose-dify-upload-workflow.yml up -d
```

## 本地开发

### 前置要求

- Go 1.24 (与Dockerfile保持一致)
- Git

```bash
# 安装依赖
go mod tidy

# 运行服务
go run cmd/main.go
```

## 监控与调试

服务提供了一个健康检查端点，可用于监控服务状态：

```
GET /health
```

响应示例：

```json
{
  "status": "ok",
  "version": "1.0.0",
  "goVersion": "go1.24.3",
  "numCPU": 8,
  "numGoroutine": 10,
  "serverTime": "2023-08-15T10:20:30Z",
  "config": {
    "maxUploadFiles": 10,
    "maxFileSize": 104857600,
    "defaultTimeout": 120
  }
}
```

## 常见问题解答

### 上传文件大小限制

目前系统支持的文件大小限制如下：
- 文档：小于15MB
- 图片：小于10MB
- 音频：小于50MB
- 视频：小于100MB

实际限制可能会根据Dify平台的限制而变化。

### 文件格式支持

系统支持以下文件格式：

- 文档类：TXT, MD, MARKDOWN, PDF, HTML, XLSX, XLS, DOCX, CSV, EML, MSG, PPTX, PPT, XML, EPUB
- 图片类：JPG, JPEG, PNG, GIF, WEBP, SVG
- 音频类：MP3, M4A, WAV, WEBM, AMR
- 视频类：MP4, MOV, MPEG, MPGA

### 常见错误处理

1. **401错误**：检查您的API密钥是否正确，以及是否正确设置了Authorization头部。
2. **413错误**：检查上传的文件是否超过大小限制。
3. **415错误**：检查上传的文件类型是否被支持。
4. **400错误"未找到file字段"**：检查请求格式，确保inputs中包含file字段。
5. **异步+流式响应错误**：如果您同时使用异步请求和流式响应模式，确保您的回调服务能正确处理返回结果。本服务会自动将这类请求转为blocking模式处理。

### 流式响应问题

如果在使用流式响应（streaming）模式时遇到问题，请确保客户端正确处理Server-Sent Events (SSE)格式的响应。 