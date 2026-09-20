package service

import "strings"

// 分叉本地改动：客户端必须始终看到自己请求的公开模型名。
//
// 上游只在存在渠道映射时（originalModel != mappedModel）重写协议里的 model 字段，
// 于是没配映射、但上游自报别名的模型会把上游名字泄露给客户端——例如请求
// gpt-5.6-sol、上游按 gpt-6-sol 应答。下面的包装保留上游"有映射"分支的原样行为，
// 只补齐无映射那一半：把上游自报的名字改写成公开名。
//
// 审计不受影响：upstreamResponseModelObserver 在任何重写之前观测上游原始字节，
// usage_logs 仍然记录真实的 upstream_response_model，管理端用量页照常标记不一致。

// rewriteClientModelInSSELine 重写一行 SSE 里的模型字段。
func (s *OpenAIGatewayService) rewriteClientModelInSSELine(line, mappedModel, publicModel string) string {
	publicModel = strings.TrimSpace(publicModel)
	if publicModel == "" {
		return line
	}
	if mapped := strings.TrimSpace(mappedModel); mapped != "" && mapped != publicModel {
		return s.replaceModelInSSELine(line, mapped, publicModel)
	}
	data, ok := extractOpenAISSEDataLine(line)
	if !ok {
		return line
	}
	declared := declaredOpenAIResponseModel([]byte(data))
	if declared == "" || declared == publicModel {
		return line
	}
	return s.replaceModelInSSELine(line, declared, publicModel)
}

// rewriteClientModelInSSEBody 对整段 SSE 文本逐行重写，用于把流式响应兜底成文本返回的分支。
func (s *OpenAIGatewayService) rewriteClientModelInSSEBody(body, mappedModel, publicModel string) string {
	if strings.TrimSpace(publicModel) == "" {
		return body
	}
	lines := strings.Split(body, "\n")
	changed := false
	for i, line := range lines {
		if _, ok := extractOpenAISSEDataLine(line); !ok {
			continue
		}
		rewritten := s.rewriteClientModelInSSELine(line, mappedModel, publicModel)
		if rewritten != line {
			lines[i] = rewritten
			changed = true
		}
	}
	if !changed {
		return body
	}
	return strings.Join(lines, "\n")
}

// rewriteClientModelInResponseBody 重写非流式响应体里的模型字段。
func (s *OpenAIGatewayService) rewriteClientModelInResponseBody(body []byte, mappedModel, publicModel string) []byte {
	publicModel = strings.TrimSpace(publicModel)
	if publicModel == "" {
		return body
	}
	if mapped := strings.TrimSpace(mappedModel); mapped != "" && mapped != publicModel {
		return s.replaceModelInResponseBody(body, mapped, publicModel)
	}
	declared := declaredOpenAIResponseModel(body)
	if declared == "" || declared == publicModel {
		return body
	}
	return s.replaceModelInResponseBody(body, declared, publicModel)
}

// declaredOpenAIResponseModel 返回 OpenAI 形状负载自报的模型名，字段优先级与响应模型观测器一致。
// 未声明模型的增量帧在这里直接得到空串，不会触发任何重写。
func declaredOpenAIResponseModel(payload []byte) string {
	return firstValidTrimmedGJSONString(payload, "response.model", "model")
}
