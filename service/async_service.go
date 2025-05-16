package service

import (
	"dify-upload-workflow/model"
	"dify-upload-workflow/utils"
	"encoding/json"
	"log"
	"strings"
)

// AsyncProcessor 异步处理器
type AsyncProcessor struct {
	DifyService *DifyService
}

// NewAsyncProcessor 创建新的异步处理器
func NewAsyncProcessor(difyService *DifyService) *AsyncProcessor {
	return &AsyncProcessor{
		DifyService: difyService,
	}
}

// ProcessSingleFileAsync 异步处理单文件请求
func (p *AsyncProcessor) ProcessSingleFileAsync(request *model.SingleFileWorkflowRequest, fileInput *model.FileSingleInput) (model.AsyncResponse, error) {
	// 初始化异步请求
	asyncResp := utils.InitAsyncRequest(request.Async)
	requestID := asyncResp.RequestID

	// 更新状态为处理中
	utils.UpdateAsyncRequestStatus(requestID, "processing", "请求正在处理中")

	// 启动goroutine处理请求
	go func() {
		// 检查是否为streaming模式
		if strings.ToLower(request.ResponseMode) == "streaming" {
			// 对于streaming模式，我们需要特殊处理
			// 1. 下载文件
			fileContent, filename, err := utils.DownloadFile(fileInput.FileURL)
			if err != nil {
				utils.UpdateAsyncRequestStatus(requestID, "failed", "下载文件失败: "+err.Error())
				utils.CallbackResult(request.Async.CallbackURL, requestID, model.WorkflowResponse{
					ErrorMessage: "下载文件失败: " + err.Error(),
				})
				return
			}

			// 2. 上传文件到Dify
			fileResp, err := p.DifyService.UploadFile(fileContent, filename, request.User)
			if err != nil {
				utils.UpdateAsyncRequestStatus(requestID, "failed", "上传文件到Dify失败: "+err.Error())
				utils.CallbackResult(request.Async.CallbackURL, requestID, model.WorkflowResponse{
					ErrorMessage: "上传文件到Dify失败: " + err.Error(),
				})
				return
			}

			// 3. 获取文件类型
			fileType := utils.GetFileType(fileResp.Extension)

			// 4. 构建文件映射
			fileMap := map[string]interface{}{
				"transfer_method": "local_file",
				"upload_file_id":  fileResp.ID,
				"type":            fileType,
			}

			// 5. 更新inputs中的文件信息
			// 如果用户指定的变量名存在于inputs中，先移除它
			delete(request.Inputs, fileInput.FileValue)
			request.Inputs[fileInput.FileValue] = fileMap

			// 6. 构建工作流请求，但强制使用blocking模式
			workflowRequest := &model.DifyWorkflowRunRequest{
				Inputs:       request.Inputs,
				ResponseMode: "blocking", // 强制使用blocking模式处理异步请求
				User:         request.User,
			}

			// 7. 执行工作流
			respBody, err := p.DifyService.RunWorkflow(workflowRequest)
			if err != nil {
				utils.UpdateAsyncRequestStatus(requestID, "failed", "执行工作流失败: "+err.Error())
				utils.CallbackResult(request.Async.CallbackURL, requestID, model.WorkflowResponse{
					ErrorMessage: "执行工作流失败: " + err.Error(),
				})
				return
			}

			// 8. 解析工作流响应
			var workflowResp interface{}
			if err = json.Unmarshal(respBody, &workflowResp); err != nil {
				utils.UpdateAsyncRequestStatus(requestID, "failed", "解析工作流响应失败: "+err.Error())
				utils.CallbackResult(request.Async.CallbackURL, requestID, model.WorkflowResponse{
					ErrorMessage: "解析工作流响应失败: " + err.Error(),
				})
				return
			}

			// 9. 构建成功响应
			result := &model.WorkflowResponse{
				FileResponse: []model.DifyFileUploadResponse{*fileResp},
				WorkflowData: workflowResp,
			}

			// 更新状态为完成
			utils.UpdateAsyncRequestStatus(requestID, "completed", "处理完成")

			// 回调成功结果
			callbackError := utils.CallbackResult(request.Async.CallbackURL, requestID, result)
			if callbackError != nil {
				log.Printf("回调成功结果失败: %v", callbackError)
			}
			return
		}

		// 对于非streaming模式，使用原有的处理逻辑
		resp, err := p.DifyService.ProcessSingleFileWorkflow(request, fileInput)

		if err != nil {
			// 更新状态为失败
			utils.UpdateAsyncRequestStatus(requestID, "failed", "处理失败: "+err.Error())

			// 回调错误结果
			callbackError := utils.CallbackResult(request.Async.CallbackURL, requestID, model.WorkflowResponse{
				ErrorMessage: err.Error(),
			})

			if callbackError != nil {
				log.Printf("回调错误结果失败: %v", callbackError)
			}
			return
		}

		// 更新状态为完成
		utils.UpdateAsyncRequestStatus(requestID, "completed", "处理完成")

		// 回调成功结果
		callbackError := utils.CallbackResult(request.Async.CallbackURL, requestID, resp)
		if callbackError != nil {
			log.Printf("回调成功结果失败: %v", callbackError)
		}
	}()

	return asyncResp, nil
}

// ProcessMultiFilesAsync 异步处理多文件请求
func (p *AsyncProcessor) ProcessMultiFilesAsync(request *model.SingleFileWorkflowRequest, filesInput *model.FilesInput) (model.AsyncResponse, error) {
	// 初始化异步请求
	asyncResp := utils.InitAsyncRequest(request.Async)
	requestID := asyncResp.RequestID

	// 更新状态为处理中
	utils.UpdateAsyncRequestStatus(requestID, "processing", "请求正在处理中")

	// 启动goroutine处理请求
	go func() {
		// 检查是否为streaming模式
		if strings.ToLower(request.ResponseMode) == "streaming" {
			// 对于streaming模式，我们需要特殊处理
			// 处理多个文件
			var fileResponses []model.DifyFileUploadResponse
			var fileMapList []interface{}

			for _, fileURL := range filesInput.FileURLs {
				// 下载文件
				fileContent, filename, err := utils.DownloadFile(fileURL)
				if err != nil {
					utils.UpdateAsyncRequestStatus(requestID, "failed", "下载文件失败 ("+fileURL+"): "+err.Error())
					utils.CallbackResult(request.Async.CallbackURL, requestID, model.WorkflowResponse{
						ErrorMessage: "下载文件失败 (" + fileURL + "): " + err.Error(),
					})
					return
				}

				// 上传文件到Dify
				fileResp, err := p.DifyService.UploadFile(fileContent, filename, request.User)
				if err != nil {
					utils.UpdateAsyncRequestStatus(requestID, "failed", "上传文件到Dify失败 ("+filename+"): "+err.Error())
					utils.CallbackResult(request.Async.CallbackURL, requestID, model.WorkflowResponse{
						ErrorMessage: "上传文件到Dify失败 (" + filename + "): " + err.Error(),
					})
					return
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

			// 更新inputs中的文件信息列表
			// 如果用户指定的变量名存在于inputs中，先移除它
			delete(request.Inputs, filesInput.FileValue)
			request.Inputs[filesInput.FileValue] = fileMapList

			// 构建工作流请求，但强制使用blocking模式
			workflowRequest := &model.DifyWorkflowRunRequest{
				Inputs:       request.Inputs,
				ResponseMode: "blocking", // 强制使用blocking模式处理异步请求
				User:         request.User,
			}

			// 执行工作流
			respBody, err := p.DifyService.RunWorkflow(workflowRequest)
			if err != nil {
				utils.UpdateAsyncRequestStatus(requestID, "failed", "执行工作流失败: "+err.Error())
				utils.CallbackResult(request.Async.CallbackURL, requestID, model.WorkflowResponse{
					ErrorMessage: "执行工作流失败: " + err.Error(),
				})
				return
			}

			// 解析工作流响应
			var workflowResp interface{}
			if err = json.Unmarshal(respBody, &workflowResp); err != nil {
				utils.UpdateAsyncRequestStatus(requestID, "failed", "解析工作流响应失败: "+err.Error())
				utils.CallbackResult(request.Async.CallbackURL, requestID, model.WorkflowResponse{
					ErrorMessage: "解析工作流响应失败: " + err.Error(),
				})
				return
			}

			// 构建成功响应
			result := &model.WorkflowResponse{
				FileResponse: fileResponses,
				WorkflowData: workflowResp,
			}

			// 更新状态为完成
			utils.UpdateAsyncRequestStatus(requestID, "completed", "处理完成")

			// 回调成功结果
			callbackError := utils.CallbackResult(request.Async.CallbackURL, requestID, result)
			if callbackError != nil {
				log.Printf("回调成功结果失败: %v", callbackError)
			}
			return
		}

		// 对于非streaming模式，使用原有的处理逻辑
		resp, err := p.DifyService.ProcessMultiFilesWorkflow(request, filesInput)

		if err != nil {
			// 更新状态为失败
			utils.UpdateAsyncRequestStatus(requestID, "failed", "处理失败: "+err.Error())

			// 回调错误结果
			callbackError := utils.CallbackResult(request.Async.CallbackURL, requestID, model.WorkflowResponse{
				ErrorMessage: err.Error(),
			})

			if callbackError != nil {
				log.Printf("回调错误结果失败: %v", callbackError)
			}
			return
		}

		// 更新状态为完成
		utils.UpdateAsyncRequestStatus(requestID, "completed", "处理完成")

		// 回调成功结果
		callbackError := utils.CallbackResult(request.Async.CallbackURL, requestID, resp)
		if callbackError != nil {
			log.Printf("回调成功结果失败: %v", callbackError)
		}
	}()

	return asyncResp, nil
}
