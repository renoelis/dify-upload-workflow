package service

import (
	"bytes"
	"dify-upload-workflow/model"
	"dify-upload-workflow/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// DifyService Dify接口服务
type DifyService struct {
	BaseURL string
	ApiKey  string
}

// NewDifyService 创建新的Dify服务实例
func NewDifyService(domain string, apiKey string) *DifyService {
	// 确保域名没有尾部斜杠
	domain = strings.TrimSuffix(domain, "/")
	return &DifyService{
		BaseURL: domain,
		ApiKey:  apiKey,
	}
}

// UploadFile 上传文件到Dify
func (s *DifyService) UploadFile(fileContent []byte, filename string, user string) (*model.DifyFileUploadResponse, error) {
	url := fmt.Sprintf("%s/v1/files/upload", s.BaseURL)

	// 创建multipart表单
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加文件
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("创建表单文件失败: %w", err)
	}
	if _, err = io.Copy(part, bytes.NewReader(fileContent)); err != nil {
		return nil, fmt.Errorf("写入文件内容失败: %w", err)
	}

	// 添加用户标识
	if err = writer.WriteField("user", user); err != nil {
		return nil, fmt.Errorf("写入用户标识失败: %w", err)
	}

	// 关闭writer
	if err = writer.Close(); err != nil {
		return nil, fmt.Errorf("关闭writer失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+s.ApiKey)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应内容失败: %w", err)
	}

	// 处理错误响应
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// 尝试解析错误响应
		var errorResp struct {
			Error string `json:"error"`
		}
		if err = json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Error != "" {
			return nil, errors.New(errorResp.Error)
		}
		return nil, fmt.Errorf("上传文件失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析成功响应
	var fileResponse model.DifyFileUploadResponse
	if err = json.Unmarshal(respBody, &fileResponse); err != nil {
		return nil, fmt.Errorf("解析响应内容失败: %w", err)
	}

	return &fileResponse, nil
}

// RunWorkflow 执行工作流
func (s *DifyService) RunWorkflow(request *model.DifyWorkflowRunRequest) ([]byte, error) {
	url := fmt.Sprintf("%s/v1/workflows/run", s.BaseURL)

	// 将请求对象转为JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("请求体序列化失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.ApiKey)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应内容失败: %w", err)
	}

	// 处理错误响应
	if resp.StatusCode != http.StatusOK {
		// 尝试解析错误响应
		var errorResp struct {
			Error string `json:"error"`
		}
		if err = json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Error != "" {
			return nil, errors.New(errorResp.Error)
		}
		return nil, fmt.Errorf("执行工作流失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// StreamWorkflow 流式执行工作流
func (s *DifyService) StreamWorkflow(request *model.DifyWorkflowRunRequest, writer http.ResponseWriter) error {
	url := fmt.Sprintf("%s/v1/workflows/run", s.BaseURL)

	// 强制设置为streaming模式
	request.ResponseMode = "streaming"

	// 将请求对象转为JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("请求体序列化失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.ApiKey)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 处理错误响应
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		// 尝试解析错误响应
		var errorResp struct {
			Error string `json:"error"`
		}
		if err = json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Error != "" {
			return errors.New(errorResp.Error)
		}
		return fmt.Errorf("执行工作流失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 设置响应头
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("Access-Control-Allow-Origin", "*")

	// 直接透传Dify API的流式响应
	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return fmt.Errorf("流式传输响应失败: %w", err)
	}

	return nil
}

// ProcessSingleFileWorkflow 处理单文件工作流
func (s *DifyService) ProcessSingleFileWorkflow(request *model.SingleFileWorkflowRequest, fileInput *model.FileSingleInput) (*model.WorkflowResponse, error) {
	response := &model.WorkflowResponse{}

	// 下载文件
	fileContent, filename, err := utils.DownloadFile(fileInput.FileURL)
	if err != nil {
		return nil, fmt.Errorf("下载文件失败: %w", err)
	}

	// 上传文件到Dify
	fileResp, err := s.UploadFile(fileContent, filename, request.User)
	if err != nil {
		return nil, fmt.Errorf("上传文件到Dify失败: %w", err)
	}

	// 添加文件上传响应
	response.FileResponse = append(response.FileResponse, *fileResp)

	// 获取文件类型
	fileType := utils.GetFileType(fileResp.Extension)

	// 构建文件映射
	fileMap := map[string]interface{}{
		"transfer_method": "local_file",
		"upload_file_id":  fileResp.ID,
		"type":            fileType,
	}

	// 严格使用用户提供的文件变量名
	// 如果用户指定的变量名存在于inputs中，先移除它
	delete(request.Inputs, fileInput.FileValue)

	// 添加文件信息到用户指定的变量名
	request.Inputs[fileInput.FileValue] = fileMap

	// 构建工作流请求
	workflowRequest := &model.DifyWorkflowRunRequest{
		Inputs:       request.Inputs,
		ResponseMode: request.ResponseMode,
		User:         request.User,
	}

	// 执行工作流
	respBody, err := s.RunWorkflow(workflowRequest)
	if err != nil {
		response.ErrorMessage = err.Error()
		return response, nil
	}

	// 解析工作流响应
	var workflowResp interface{}
	if err = json.Unmarshal(respBody, &workflowResp); err != nil {
		response.ErrorMessage = "解析工作流响应失败: " + err.Error()
		return response, nil
	}

	response.WorkflowData = workflowResp
	return response, nil
}

// ProcessMultiFilesWorkflow 处理多文件工作流
func (s *DifyService) ProcessMultiFilesWorkflow(request *model.SingleFileWorkflowRequest, filesInput *model.FilesInput) (*model.WorkflowResponse, error) {
	response := &model.WorkflowResponse{}

	// 处理多个文件
	var fileResponses []model.DifyFileUploadResponse
	var fileMapList []interface{}

	for _, fileURL := range filesInput.FileURLs {
		// 下载文件
		fileContent, filename, err := utils.DownloadFile(fileURL)
		if err != nil {
			return nil, fmt.Errorf("下载文件失败 (%s): %w", fileURL, err)
		}

		// 上传文件到Dify
		fileResp, err := s.UploadFile(fileContent, filename, request.User)
		if err != nil {
			return nil, fmt.Errorf("上传文件到Dify失败 (%s): %w", filename, err)
		}

		// 添加文件上传响应
		fileResponses = append(fileResponses, *fileResp)

		// 获取文件类型
		fileType := utils.GetFileType(fileResp.Extension)

		// 构建文件映射
		fileMap := map[string]interface{}{
			"transfer_method": "local_file",
			"upload_file_id":  fileResp.ID,
			"type":            fileType,
		}

		fileMapList = append(fileMapList, fileMap)
	}

	// 更新response的文件上传响应
	response.FileResponse = fileResponses

	// 严格使用用户提供的文件变量名
	// 如果用户指定的变量名存在于inputs中，先移除它
	delete(request.Inputs, filesInput.FileValue)

	// 添加文件信息到用户指定的变量名
	request.Inputs[filesInput.FileValue] = fileMapList

	// 构建工作流请求
	workflowRequest := &model.DifyWorkflowRunRequest{
		Inputs:       request.Inputs,
		ResponseMode: request.ResponseMode,
		User:         request.User,
	}

	// 执行工作流
	respBody, err := s.RunWorkflow(workflowRequest)
	if err != nil {
		response.ErrorMessage = err.Error()
		return response, nil
	}

	// 解析工作流响应
	var workflowResp interface{}
	if err = json.Unmarshal(respBody, &workflowResp); err != nil {
		response.ErrorMessage = "解析工作流响应失败: " + err.Error()
		return response, nil
	}

	response.WorkflowData = workflowResp
	return response, nil
}
