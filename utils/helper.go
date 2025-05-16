package utils

import (
	"dify-upload-workflow/model"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// GetFileType 根据文件扩展名获取类型
func GetFileType(fileExt string) string {
	ext := strings.TrimPrefix(strings.ToLower(fileExt), ".")
	if fileType, ok := model.FileTypeMapping[ext]; ok {
		return fileType
	}
	return "custom" // 默认为自定义类型
}

// SanitizeFilename 处理文件名，移除不允许的字符
func SanitizeFilename(filename string) string {
	// 移除查询参数
	if idx := strings.Index(filename, "?"); idx > 0 {
		filename = filename[:idx]
	}

	// 只过滤掉文件系统中绝对不允许的字符: / \ : * ? " < > |
	re := regexp.MustCompile(`[/\\:\*\?"<>|]`)
	filename = re.ReplaceAllString(filename, "_")

	// 确保文件名不为空
	if filename == "" || filename == "." {
		filename = "file_" + time.Now().Format("20060102150405")
	}

	// 确保文件名长度不超过255个字符
	if len(filename) > 255 {
		ext := filepath.Ext(filename)
		filename = filename[:255-len(ext)] + ext
	}

	return filename
}

// DownloadFile 从URL下载文件
func DownloadFile(url string) ([]byte, string, error) {
	// 设置超时时间
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// 发起GET请求
	resp, err := client.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, "", errors.New("下载文件失败，HTTP状态码: " + resp.Status)
	}

	// 获取文件名
	filename := getFilenameFromURL(url)
	if filename == "" {
		// 尝试从Content-Disposition获取
		filename = getFilenameFromHeader(resp.Header.Get("Content-Disposition"))
		if filename == "" {
			// 生成随机文件名
			filename = "download_" + time.Now().Format("20060102150405")
		}
	}

	// 规范化文件名
	filename = SanitizeFilename(filename)

	// 读取文件内容
	fileContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return fileContent, filename, nil
}

// getFilenameFromURL 从URL获取文件名
func getFilenameFromURL(url string) string {
	// 移除查询参数
	if idx := strings.Index(url, "?"); idx > 0 {
		url = url[:idx]
	}

	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// getFilenameFromHeader 从Content-Disposition头获取文件名
func getFilenameFromHeader(header string) string {
	if header == "" {
		return ""
	}

	// 检查Content-Disposition头中的filename参数
	if strings.Contains(header, "filename=") {
		parts := strings.Split(header, "filename=")
		if len(parts) > 1 {
			filename := parts[1]
			// 处理引号
			filename = strings.Trim(filename, "\"'")
			return filename
		}
	}
	return ""
}

// GetFileExtension 获取文件扩展名
func GetFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	return strings.ToLower(ext)
}

// GetFileReader 从文件内容创建io.Reader
func GetFileReader(fileContent []byte) io.Reader {
	return strings.NewReader(string(fileContent))
}

// GetFormFile 从表单获取文件
func GetFormFile(form *multipart.Form, key string) ([]*multipart.FileHeader, error) {
	if form.File == nil {
		return nil, errors.New("没有文件上传")
	}

	files := form.File[key]
	if len(files) == 0 {
		return nil, errors.New("未找到名为 " + key + " 的文件")
	}

	return files, nil
}

// BuildAPIResponse 构建统一API响应
func BuildAPIResponse(code int, message string, data interface{}) model.ApiResponse {
	return model.ApiResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
}
