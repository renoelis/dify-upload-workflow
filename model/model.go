package model

// DifyFileUploadResponse Dify文件上传响应
type DifyFileUploadResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Size      int    `json:"size"`
	Extension string `json:"extension"`
	MimeType  string `json:"mime_type"`
	CreatedBy string `json:"created_by"`
	CreatedAt int64  `json:"created_at"`
}

// DifyWorkflowRunRequest Dify工作流运行请求
type DifyWorkflowRunRequest struct {
	Inputs       map[string]interface{} `json:"inputs"`
	ResponseMode string                 `json:"response_mode"`
	User         string                 `json:"user"`
}

// DifyWorkflowRunResponse Dify工作流运行响应
type DifyWorkflowRunResponse struct {
	TaskID        string                 `json:"task_id,omitempty"`
	Answer        string                 `json:"answer,omitempty"`
	AnswerSeconds float64                `json:"answer_seconds,omitempty"`
	Message       string                 `json:"message,omitempty"`
	Status        string                 `json:"status,omitempty"`
	Extra         map[string]interface{} `json:"extra,omitempty"`
	RawResponse   interface{}            `json:"raw_response,omitempty"`
}

// SingleFileWorkflowRequest 单文件工作流请求
type SingleFileWorkflowRequest struct {
	Domain       string                 `json:"domain" binding:"required"`
	Inputs       map[string]interface{} `json:"inputs" binding:"required"`
	ResponseMode string                 `json:"response_mode" binding:"required"`
	User         string                 `json:"user" binding:"required"`
	Async        *AsyncRequest          `json:"async,omitempty"` // 异步请求配置，为空则为同步请求
}

// FileSingleInput 单文件输入结构
type FileSingleInput struct {
	FileURL   string `json:"file_url"`
	FileValue string `json:"file_value" binding:"required"`
}

// FilesInput 多文件输入结构
type FilesInput struct {
	FileURLs  []string `json:"file_urls"`
	FileValue string   `json:"file_value" binding:"required"`
}

// ApiResponse API统一响应格式
type ApiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// WorkflowResponse 工作流统一响应
type WorkflowResponse struct {
	FileResponse []DifyFileUploadResponse `json:"file_response,omitempty"`
	WorkflowData interface{}              `json:"workflow_data,omitempty"`
	ErrorMessage string                   `json:"error_message,omitempty"`
}

// FileTypeMapping 文件类型映射
var FileTypeMapping = map[string]string{
	// document
	"txt":      "document",
	"md":       "document",
	"markdown": "document",
	"pdf":      "document",
	"html":     "document",
	"xlsx":     "document",
	"xls":      "document",
	"docx":     "document",
	"csv":      "document",
	"eml":      "document",
	"msg":      "document",
	"pptx":     "document",
	"ppt":      "document",
	"xml":      "document",
	"epub":     "document",

	// image
	"jpg":  "image",
	"jpeg": "image",
	"png":  "image",
	"gif":  "image",
	"webp": "image",
	"svg":  "image",

	// audio
	"mp3":  "audio",
	"m4a":  "audio",
	"wav":  "audio",
	"webm": "audio",
	"amr":  "audio",

	// video
	"mp4":  "video",
	"mov":  "video",
	"mpeg": "video",
	"mpga": "video",
}

// AsyncRequest 异步请求结构体
type AsyncRequest struct {
	CallbackURL string `json:"callback_url" binding:"required"` // 回调URL
	RequestID   string `json:"request_id,omitempty"`            // 请求ID，如果为空则自动生成
}

// AsyncResponse 异步响应结构体
type AsyncResponse struct {
	RequestID string `json:"request_id"`        // 请求ID
	Status    string `json:"status"`            // 状态: pending, processing, completed, failed
	Message   string `json:"message,omitempty"` // 可选的消息
}
