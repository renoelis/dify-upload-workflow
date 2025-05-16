package controller

import (
	"dify-upload-workflow/config"
	"dify-upload-workflow/model"
	"dify-upload-workflow/service"
	"dify-upload-workflow/utils"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// 从请求头获取API密钥
func getAPIKeyFromHeader(c *gin.Context) string {
	// 获取Authorization头
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	// 提取Bearer Token
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return authHeader
}

// SingleFileHandler 处理单文件URL格式工作流请求
func SingleFileHandler(c *gin.Context) {
	var request model.SingleFileWorkflowRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "请求格式错误: "+err.Error(), nil))
		return
	}

	// 验证域名
	if request.Domain == "" {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "域名不能为空", nil))
		return
	}

	// 从请求头获取API密钥
	apiKey := getAPIKeyFromHeader(c)
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, utils.BuildAPIResponse(401, "未提供API密钥", nil))
		return
	}

	// 检查是否有file字段
	fileInput, ok := request.Inputs["file"]
	if !ok {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "未找到file字段", nil))
		return
	}

	// 解析file字段
	fileInputMap, ok := fileInput.(map[string]interface{})
	if !ok {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "file字段格式错误", nil))
		return
	}

	// 获取文件URL和映射键
	var fileSingleInput model.FileSingleInput
	fileURL, ok := fileInputMap["file_url"].(string)
	if !ok || fileURL == "" {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "文件URL不能为空", nil))
		return
	}
	fileSingleInput.FileURL = fileURL

	fileValue, ok := fileInputMap["file_value"].(string)
	if !ok || fileValue == "" {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "文件映射键不能为空", nil))
		return
	}
	fileSingleInput.FileValue = fileValue

	// 移除file字段，避免重复
	delete(request.Inputs, "file")

	// 创建Dify服务
	difyService := service.NewDifyService(request.Domain, apiKey)

	// 检查是否为异步请求
	if request.Async != nil && request.Async.CallbackURL != "" {
		// 创建异步处理器
		asyncProcessor := service.NewAsyncProcessor(difyService)

		// 异步处理单文件请求
		asyncResp, err := asyncProcessor.ProcessSingleFileAsync(&request, &fileSingleInput)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "初始化异步请求失败: "+err.Error(), nil))
			return
		}

		// 返回异步响应
		c.JSON(http.StatusAccepted, utils.BuildAPIResponse(202, "请求已接受，正在异步处理", asyncResp))
		return
	}

	// 如果是流式响应模式，直接调用流式处理端点
	if strings.ToLower(request.ResponseMode) == "streaming" {
		// 下载文件
		fileContent, filename, err := utils.DownloadFile(fileSingleInput.FileURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "下载文件失败: "+err.Error(), nil))
			return
		}

		// 上传文件到Dify
		fileResp, err := difyService.UploadFile(fileContent, filename, request.User)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "上传文件到Dify失败: "+err.Error(), nil))
			return
		}

		// 获取文件类型
		fileType := utils.GetFileType(fileResp.Extension)

		// 构建文件映射
		fileMap := map[string]interface{}{
			"transfer_method": "local_file",
			"upload_file_id":  fileResp.ID,
			"type":            fileType,
		}

		// 更新inputs中的文件信息
		request.Inputs[fileSingleInput.FileValue] = fileMap

		// 构建工作流请求
		workflowRequest := &model.DifyWorkflowRunRequest{
			Inputs:       request.Inputs,
			ResponseMode: "streaming", // 强制使用streaming模式
			User:         request.User,
		}

		// 直接执行流式工作流并透传响应
		if err := difyService.StreamWorkflow(workflowRequest, c.Writer); err != nil {
			// 由于可能已经设置了响应头，这里只能记录错误
			c.Error(err)
		}
		return
	}

	// 处理单文件工作流
	resp, err := difyService.ProcessSingleFileWorkflow(&request, &fileSingleInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "处理工作流失败: "+err.Error(), nil))
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, utils.BuildAPIResponse(200, "成功", resp))
}

// MultiFilesHandler 处理多文件URL格式工作流请求
func MultiFilesHandler(c *gin.Context) {
	var request model.SingleFileWorkflowRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "请求格式错误: "+err.Error(), nil))
		return
	}

	// 验证域名
	if request.Domain == "" {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "域名不能为空", nil))
		return
	}

	// 从请求头获取API密钥
	apiKey := getAPIKeyFromHeader(c)
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, utils.BuildAPIResponse(401, "未提供API密钥", nil))
		return
	}

	// 检查是否有file字段
	fileInput, ok := request.Inputs["file"]
	if !ok {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "未找到file字段", nil))
		return
	}

	// 解析file字段
	fileInputMap, ok := fileInput.(map[string]interface{})
	if !ok {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "file字段格式错误", nil))
		return
	}

	// 获取文件URL列表和映射键
	var filesInput model.FilesInput
	fileURLs, ok := fileInputMap["file_urls"].([]interface{})
	if !ok || len(fileURLs) == 0 {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "文件URL列表不能为空", nil))
		return
	}

	// 检查文件数量限制
	if len(fileURLs) > config.Config.MaxUploadFiles {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "文件数量超过限制", nil))
		return
	}

	// 转换文件URL列表
	for _, urlInterface := range fileURLs {
		url, ok := urlInterface.(string)
		if !ok || url == "" {
			c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "文件URL列表中包含无效URL", nil))
			return
		}
		filesInput.FileURLs = append(filesInput.FileURLs, url)
	}

	fileValue, ok := fileInputMap["file_value"].(string)
	if !ok || fileValue == "" {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "文件映射键不能为空", nil))
		return
	}
	filesInput.FileValue = fileValue

	// 移除file字段，避免重复
	delete(request.Inputs, "file")

	// 创建Dify服务
	difyService := service.NewDifyService(request.Domain, apiKey)

	// 检查是否为异步请求
	if request.Async != nil && request.Async.CallbackURL != "" {
		// 创建异步处理器
		asyncProcessor := service.NewAsyncProcessor(difyService)

		// 异步处理多文件请求
		asyncResp, err := asyncProcessor.ProcessMultiFilesAsync(&request, &filesInput)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "初始化异步请求失败: "+err.Error(), nil))
			return
		}

		// 返回异步响应
		c.JSON(http.StatusAccepted, utils.BuildAPIResponse(202, "请求已接受，正在异步处理", asyncResp))
		return
	}

	// 如果是流式响应模式，直接调用流式处理
	if strings.ToLower(request.ResponseMode) == "streaming" {
		// 处理多个文件
		var fileMapList []interface{}

		for _, fileURL := range filesInput.FileURLs {
			// 下载文件
			fileContent, filename, err := utils.DownloadFile(fileURL)
			if err != nil {
				c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "下载文件失败 ("+fileURL+"): "+err.Error(), nil))
				return
			}

			// 上传文件到Dify
			fileResp, err := difyService.UploadFile(fileContent, filename, request.User)
			if err != nil {
				c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "上传文件到Dify失败 ("+filename+"): "+err.Error(), nil))
				return
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

		// 更新inputs中的文件信息列表
		request.Inputs[filesInput.FileValue] = fileMapList

		// 构建工作流请求
		workflowRequest := &model.DifyWorkflowRunRequest{
			Inputs:       request.Inputs,
			ResponseMode: "streaming", // 强制使用streaming模式
			User:         request.User,
		}

		// 直接执行流式工作流并透传响应
		if err := difyService.StreamWorkflow(workflowRequest, c.Writer); err != nil {
			// 由于可能已经设置了响应头，这里只能记录错误
			c.Error(err)
		}
		return
	}

	// 处理多文件工作流
	resp, err := difyService.ProcessMultiFilesWorkflow(&request, &filesInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "处理工作流失败: "+err.Error(), nil))
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, utils.BuildAPIResponse(200, "成功", resp))
}

// SingleFileFormHandler 处理单文件form-data格式工作流请求
func SingleFileFormHandler(c *gin.Context) {
	// 获取域名
	domain := c.PostForm("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "域名不能为空", nil))
		return
	}

	// 获取响应模式
	responseMode := c.PostForm("response_mode")
	if responseMode == "" {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "响应模式不能为空", nil))
		return
	}

	// 获取用户标识
	user := c.PostForm("user")
	if user == "" {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "用户标识不能为空", nil))
		return
	}

	// 获取文件映射键
	fileValue := c.PostForm("file_value")
	if fileValue == "" {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "文件映射键不能为空", nil))
		return
	}

	// 获取其他输入参数
	inputs := make(map[string]interface{})
	for key, values := range c.Request.PostForm {
		if key != "file_value" && key != "domain" && key != "user" && key != "response_mode" && key != "callback_url" && key != "request_id" && len(values) > 0 {
			inputs[key] = values[0]
		}
	}

	// 检查是否有异步请求参数
	var asyncRequest *model.AsyncRequest
	callbackURL := c.PostForm("callback_url")
	requestID := c.PostForm("request_id")
	if callbackURL != "" {
		asyncRequest = &model.AsyncRequest{
			CallbackURL: callbackURL,
			RequestID:   requestID,
		}
	}

	// 从请求头获取API密钥
	apiKey := getAPIKeyFromHeader(c)
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, utils.BuildAPIResponse(401, "未提供API密钥", nil))
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "文件上传失败: "+err.Error(), nil))
		return
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "打开文件失败: "+err.Error(), nil))
		return
	}
	defer src.Close()

	// 读取文件内容
	fileContent, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "读取文件内容失败: "+err.Error(), nil))
		return
	}

	// 构建请求结构体
	request := &model.SingleFileWorkflowRequest{
		Domain:       domain,
		Inputs:       inputs,
		ResponseMode: responseMode,
		User:         user,
		Async:        asyncRequest,
	}

	// 构建文件输入结构体
	fileSingleInput := &model.FileSingleInput{
		FileValue: fileValue,
	}

	// 创建Dify服务
	difyService := service.NewDifyService(domain, apiKey)

	// 检查是否为异步请求
	if asyncRequest != nil {
		// 异步处理单文件请求（使用文件内容而不是URL）
		asyncResp := utils.InitAsyncRequest(asyncRequest)
		requestID := asyncResp.RequestID

		// 在goroutine中处理文件上传和工作流
		go func() {
			// 上传文件到Dify
			fileResp, err := difyService.UploadFile(fileContent, file.Filename, request.User)
			if err != nil {
				utils.UpdateAsyncRequestStatus(requestID, "failed", "上传文件到Dify失败: "+err.Error())
				utils.CallbackResult(asyncRequest.CallbackURL, requestID, model.WorkflowResponse{
					ErrorMessage: "上传文件到Dify失败: " + err.Error(),
				})
				return
			}

			// 获取文件类型
			fileType := utils.GetFileType(fileResp.Extension)

			// 构建文件映射
			fileMap := map[string]interface{}{
				"transfer_method": "local_file",
				"upload_file_id":  fileResp.ID,
				"type":            fileType,
			}

			// 更新inputs中的文件信息
			request.Inputs[fileSingleInput.FileValue] = fileMap

			// 构建工作流请求
			workflowRequest := &model.DifyWorkflowRunRequest{
				Inputs:       request.Inputs,
				ResponseMode: "blocking", // 强制使用blocking模式处理异步请求，即使用户请求streaming
				User:         request.User,
			}

			// 执行工作流
			respBody, err := difyService.RunWorkflow(workflowRequest)
			if err != nil {
				utils.UpdateAsyncRequestStatus(requestID, "failed", "执行工作流失败: "+err.Error())
				utils.CallbackResult(asyncRequest.CallbackURL, requestID, model.WorkflowResponse{
					ErrorMessage: "执行工作流失败: " + err.Error(),
				})
				return
			}

			// 解析工作流响应
			var workflowResp interface{}
			if err := json.Unmarshal(respBody, &workflowResp); err != nil {
				utils.UpdateAsyncRequestStatus(requestID, "failed", "解析工作流响应失败: "+err.Error())
				utils.CallbackResult(asyncRequest.CallbackURL, requestID, model.WorkflowResponse{
					ErrorMessage: "解析工作流响应失败: " + err.Error(),
				})
				return
			}

			// 构建成功响应
			result := &model.WorkflowResponse{
				FileResponse: []model.DifyFileUploadResponse{*fileResp},
				WorkflowData: workflowResp,
			}

			// 更新状态为完成
			utils.UpdateAsyncRequestStatus(requestID, "completed", "处理完成")

			// 回调成功结果
			utils.CallbackResult(asyncRequest.CallbackURL, requestID, result)
		}()

		// 返回异步响应
		c.JSON(http.StatusAccepted, utils.BuildAPIResponse(202, "请求已接受，正在异步处理", asyncResp))
		return
	}

	// 上传文件到Dify
	fileResp, err := difyService.UploadFile(fileContent, file.Filename, request.User)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "上传文件到Dify失败: "+err.Error(), nil))
		return
	}

	// 获取文件类型
	fileType := utils.GetFileType(fileResp.Extension)

	// 构建文件映射
	fileMap := map[string]interface{}{
		"transfer_method": "local_file",
		"upload_file_id":  fileResp.ID,
		"type":            fileType,
	}

	// 更新inputs中的文件信息
	request.Inputs[fileSingleInput.FileValue] = fileMap

	// 如果是流式响应模式，直接调用流式处理端点
	if strings.ToLower(responseMode) == "streaming" {
		// 构建工作流请求
		workflowRequest := &model.DifyWorkflowRunRequest{
			Inputs:       request.Inputs,
			ResponseMode: "streaming", // 强制使用streaming模式
			User:         request.User,
		}

		// 直接执行流式工作流并透传响应
		if err := difyService.StreamWorkflow(workflowRequest, c.Writer); err != nil {
			// 由于可能已经设置了响应头，这里只能记录错误
			c.Error(err)
		}
		return
	}

	// 构建工作流请求
	workflowRequest := &model.DifyWorkflowRunRequest{
		Inputs:       request.Inputs,
		ResponseMode: request.ResponseMode,
		User:         request.User,
	}

	// 执行工作流
	respBody, err := difyService.RunWorkflow(workflowRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "执行工作流失败: "+err.Error(), nil))
		return
	}

	// 解析工作流响应
	var workflowResp interface{}
	if err := json.Unmarshal(respBody, &workflowResp); err != nil {
		c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "解析工作流响应失败: "+err.Error(), nil))
		return
	}

	// 构建响应
	result := &model.WorkflowResponse{
		FileResponse: []model.DifyFileUploadResponse{*fileResp},
		WorkflowData: workflowResp,
	}

	// 返回响应
	c.JSON(http.StatusOK, utils.BuildAPIResponse(200, "成功", result))
}

// MultiFilesFormHandler 处理多文件表单格式工作流请求
func MultiFilesFormHandler(c *gin.Context) {
	// 解析表单
	if err := c.Request.ParseMultipartForm(config.Config.MaxFileSize * int64(config.Config.MaxUploadFiles)); err != nil {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "解析表单失败: "+err.Error(), nil))
		return
	}

	// 获取domain参数
	domain := c.Request.FormValue("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "域名不能为空", nil))
		return
	}

	// 从请求头获取API密钥
	apiKey := getAPIKeyFromHeader(c)
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, utils.BuildAPIResponse(401, "未提供API密钥", nil))
		return
	}

	// 获取用户参数
	user := c.Request.FormValue("user")
	if user == "" {
		user = config.Config.DefaultUser
	}

	// 获取响应模式
	responseMode := c.Request.FormValue("response_mode")
	if responseMode == "" {
		responseMode = "blocking"
	}

	// 检查是否有异步请求参数
	var asyncRequest *model.AsyncRequest
	callbackURL := c.Request.FormValue("callback_url")
	requestID := c.Request.FormValue("request_id")
	if callbackURL != "" {
		asyncRequest = &model.AsyncRequest{
			CallbackURL: callbackURL,
			RequestID:   requestID,
		}
	}

	// 如果是异步请求
	if asyncRequest != nil {
		// 初始化异步响应
		asyncResp := utils.InitAsyncRequest(asyncRequest)
		requestID := asyncResp.RequestID

		// 在goroutine中处理文件上传和工作流
		go func() {
			// 创建上传服务
			uploadService := service.NewUploadService()

			// 处理多文件表单工作流，强制使用blocking模式
			resp, err := uploadService.ProcessMultiFormFiles(domain, apiKey, c.Request.MultipartForm, user, "blocking")
			if err != nil {
				utils.UpdateAsyncRequestStatus(requestID, "failed", "处理工作流失败: "+err.Error())
				utils.CallbackResult(asyncRequest.CallbackURL, requestID, model.WorkflowResponse{
					ErrorMessage: "处理工作流失败: " + err.Error(),
				})
				return
			}

			// 更新状态为完成
			utils.UpdateAsyncRequestStatus(requestID, "completed", "处理完成")

			// 回调成功结果
			utils.CallbackResult(asyncRequest.CallbackURL, requestID, resp)
		}()

		// 返回异步响应
		c.JSON(http.StatusAccepted, utils.BuildAPIResponse(202, "请求已接受，正在异步处理", asyncResp))
		return
	}

	// 创建上传服务
	uploadService := service.NewUploadService()

	// 处理多文件表单工作流
	if strings.ToLower(responseMode) == "streaming" {
		// 流式响应
		err := uploadService.StreamMultiFormFiles(domain, apiKey, c.Request.MultipartForm, user, c.Writer)
		if err != nil {
			// 注意：这里可能已经开始发送响应，所以错误只能记录
			c.Error(err)
			return
		}
		return
	}

	// 阻塞响应
	resp, err := uploadService.ProcessMultiFormFiles(domain, apiKey, c.Request.MultipartForm, user, responseMode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "处理工作流失败: "+err.Error(), nil))
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, utils.BuildAPIResponse(200, "成功", resp))
}

// QueryAsyncStatus 查询异步请求的处理状态
func QueryAsyncStatus(c *gin.Context) {
	// 获取请求ID
	requestID := c.Param("requestID")
	if requestID == "" {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "请求ID不能为空", nil))
		return
	}

	// 查询请求状态
	status, exists := utils.GetAsyncRequestStatus(requestID)
	if !exists {
		c.JSON(http.StatusNotFound, utils.BuildAPIResponse(404, "未找到指定请求ID的处理记录", nil))
		return
	}

	// 返回状态信息
	c.JSON(http.StatusOK, utils.BuildAPIResponse(200, "成功", status))
}

// UploadURLFileHandler 处理URL文件上传到Dify，不调用工作流
func UploadURLFileHandler(c *gin.Context) {
	var request struct {
		FileURL string `json:"file_url" binding:"required"`
		User    string `json:"user" binding:"required"`
		Domain  string `json:"domain" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, utils.BuildAPIResponse(400, "请求格式错误: "+err.Error(), nil))
		return
	}

	// 从请求头获取API密钥
	apiKey := getAPIKeyFromHeader(c)
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, utils.BuildAPIResponse(401, "未提供API密钥", nil))
		return
	}

	// 下载文件
	fileContent, filename, err := utils.DownloadFile(request.FileURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "下载文件失败: "+err.Error(), nil))
		return
	}

	// 创建Dify服务
	difyService := service.NewDifyService(request.Domain, apiKey)

	// 上传文件到Dify
	fileResp, err := difyService.UploadFile(fileContent, filename, request.User)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.BuildAPIResponse(500, "上传文件到Dify失败: "+err.Error(), nil))
		return
	}

	// 返回上传结果
	c.JSON(http.StatusOK, utils.BuildAPIResponse(200, "上传成功", fileResp))
}
