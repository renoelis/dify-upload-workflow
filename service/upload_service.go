package service

import (
	"dify-upload-workflow/config"
	"dify-upload-workflow/model"
	"dify-upload-workflow/utils"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
)

// UploadService 文件上传服务
type UploadService struct{}

// NewUploadService 创建新的上传服务实例
func NewUploadService() *UploadService {
	return &UploadService{}
}

// ProcessSingleFormFile 处理单文件表单上传工作流
func (s *UploadService) ProcessSingleFormFile(domain string, apiKey string, form *multipart.Form, user string, responseMode string) (*model.WorkflowResponse, error) {
	response := &model.WorkflowResponse{}

	// 获取文件
	files, err := utils.GetFormFile(form, "file")
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("未找到上传文件")
	}

	if len(files) > 1 {
		return nil, fmt.Errorf("单文件模式只支持上传一个文件")
	}

	// 获取文件映射键
	fileValue := form.Value["file_value"]
	if len(fileValue) == 0 {
		return nil, fmt.Errorf("未指定文件映射键")
	}

	// 创建Dify服务
	difyService := NewDifyService(domain, apiKey)

	// 打开文件
	file, err := files[0].Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 读取文件内容
	fileContent := make([]byte, files[0].Size)
	_, err = file.Read(fileContent)
	if err != nil {
		return nil, fmt.Errorf("读取文件内容失败: %w", err)
	}

	// 规范化文件名
	sanitizedFilename := utils.SanitizeFilename(files[0].Filename)

	// 上传文件到Dify
	fileResp, err := difyService.UploadFile(fileContent, sanitizedFilename, user)
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

	// 构建工作流请求inputs
	inputs := make(map[string]interface{})

	// 添加文件映射
	inputs[fileValue[0]] = fileMap

	// 添加其他表单参数
	for key, values := range form.Value {
		if key != "file_value" && len(values) > 0 {
			inputs[key] = values[0]
		}
	}

	// 构建工作流请求
	workflowRequest := &model.DifyWorkflowRunRequest{
		Inputs:       inputs,
		ResponseMode: responseMode,
		User:         user,
	}

	// 执行工作流
	respBody, err := difyService.RunWorkflow(workflowRequest)
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

// ProcessMultiFormFiles 处理多文件表单上传工作流
func (s *UploadService) ProcessMultiFormFiles(domain string, apiKey string, form *multipart.Form, user string, responseMode string) (*model.WorkflowResponse, error) {
	response := &model.WorkflowResponse{}

	// 获取文件
	files, err := utils.GetFormFile(form, "files")
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("未找到上传文件")
	}

	if len(files) > config.Config.MaxUploadFiles {
		return nil, fmt.Errorf("最多只能上传 %d 个文件", config.Config.MaxUploadFiles)
	}

	// 获取文件映射键
	fileValue := form.Value["file_value"]
	if len(fileValue) == 0 {
		return nil, fmt.Errorf("未指定文件映射键")
	}

	// 创建Dify服务
	difyService := NewDifyService(domain, apiKey)

	// 处理多个文件
	var fileResponses []model.DifyFileUploadResponse
	var fileMapList []interface{}

	for _, fileHeader := range files {
		// 打开文件
		file, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("打开文件失败: %w", err)
		}

		// 读取文件内容
		fileContent := make([]byte, fileHeader.Size)
		_, err = file.Read(fileContent)
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("读取文件内容失败: %w", err)
		}

		// 上传文件到Dify
		fileResp, err := difyService.UploadFile(fileContent, fileHeader.Filename, user)
		if err != nil {
			return nil, fmt.Errorf("上传文件到Dify失败: %w", err)
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

	// 构建工作流请求inputs
	inputs := make(map[string]interface{})

	// 添加文件映射列表
	inputs[fileValue[0]] = fileMapList

	// 添加其他表单参数
	for key, values := range form.Value {
		if key != "file_value" && len(values) > 0 {
			inputs[key] = values[0]
		}
	}

	// 构建工作流请求
	workflowRequest := &model.DifyWorkflowRunRequest{
		Inputs:       inputs,
		ResponseMode: responseMode,
		User:         user,
	}

	// 执行工作流
	respBody, err := difyService.RunWorkflow(workflowRequest)
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

// StreamSingleFormFile 流式处理单文件表单上传工作流
func (s *UploadService) StreamSingleFormFile(domain string, apiKey string, form *multipart.Form, user string, w http.ResponseWriter) error {
	// 获取文件
	files, err := utils.GetFormFile(form, "file")
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("未找到上传文件")
	}

	if len(files) > 1 {
		return fmt.Errorf("单文件模式只支持上传一个文件")
	}

	// 获取文件映射键
	fileValue := form.Value["file_value"]
	if len(fileValue) == 0 {
		return fmt.Errorf("未指定文件映射键")
	}

	// 创建Dify服务
	difyService := NewDifyService(domain, apiKey)

	// 打开文件
	file, err := files[0].Open()
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 读取文件内容
	fileContent := make([]byte, files[0].Size)
	_, err = file.Read(fileContent)
	if err != nil {
		return fmt.Errorf("读取文件内容失败: %w", err)
	}

	// 上传文件到Dify
	fileResp, err := difyService.UploadFile(fileContent, files[0].Filename, user)
	if err != nil {
		return fmt.Errorf("上传文件到Dify失败: %w", err)
	}

	// 获取文件类型
	fileType := utils.GetFileType(fileResp.Extension)

	// 构建文件映射
	fileMap := map[string]interface{}{
		"transfer_method": "local_file",
		"upload_file_id":  fileResp.ID,
		"type":            fileType,
	}

	// 构建工作流请求inputs
	inputs := make(map[string]interface{})

	// 添加文件映射
	inputs[fileValue[0]] = fileMap

	// 添加其他表单参数
	for key, values := range form.Value {
		if key != "file_value" && len(values) > 0 {
			inputs[key] = values[0]
		}
	}

	// 构建工作流请求
	workflowRequest := &model.DifyWorkflowRunRequest{
		Inputs:       inputs,
		ResponseMode: "streaming", // 强制使用streaming模式
		User:         user,
	}

	// 流式执行工作流
	err = difyService.StreamWorkflow(workflowRequest, w)
	if err != nil {
		return fmt.Errorf("流式执行工作流失败: %w", err)
	}

	return nil
}

// StreamMultiFormFiles 流式处理多文件表单上传工作流
func (s *UploadService) StreamMultiFormFiles(domain string, apiKey string, form *multipart.Form, user string, w http.ResponseWriter) error {
	// 获取文件
	files, err := utils.GetFormFile(form, "files")
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("未找到上传文件")
	}

	if len(files) > config.Config.MaxUploadFiles {
		return fmt.Errorf("最多只能上传 %d 个文件", config.Config.MaxUploadFiles)
	}

	// 获取文件映射键
	fileValue := form.Value["file_value"]
	if len(fileValue) == 0 {
		return fmt.Errorf("未指定文件映射键")
	}

	// 创建Dify服务
	difyService := NewDifyService(domain, apiKey)

	// 处理多个文件
	var fileMapList []interface{}

	for _, fileHeader := range files {
		// 打开文件
		file, err := fileHeader.Open()
		if err != nil {
			return fmt.Errorf("打开文件失败: %w", err)
		}

		// 读取文件内容
		fileContent := make([]byte, fileHeader.Size)
		_, err = file.Read(fileContent)
		file.Close()
		if err != nil {
			return fmt.Errorf("读取文件内容失败: %w", err)
		}

		// 上传文件到Dify
		fileResp, err := difyService.UploadFile(fileContent, fileHeader.Filename, user)
		if err != nil {
			return fmt.Errorf("上传文件到Dify失败: %w", err)
		}

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

	// 构建工作流请求inputs
	inputs := make(map[string]interface{})

	// 添加文件映射列表
	inputs[fileValue[0]] = fileMapList

	// 添加其他表单参数
	for key, values := range form.Value {
		if key != "file_value" && len(values) > 0 {
			inputs[key] = values[0]
		}
	}

	// 构建工作流请求
	workflowRequest := &model.DifyWorkflowRunRequest{
		Inputs:       inputs,
		ResponseMode: "streaming", // 强制使用streaming模式
		User:         user,
	}

	// 流式执行工作流
	err = difyService.StreamWorkflow(workflowRequest, w)
	if err != nil {
		return fmt.Errorf("流式执行工作流失败: %w", err)
	}

	return nil
}
