package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 无渠道映射时，上游自报的别名也必须回写成客户端请求的公开名；同时审计仍要拿到上游原始值。
func TestUnmappedResponseModelForwardsPublicName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const publicModel = "gpt-5.6-sol"
	const upstreamAlias = "gpt-6-sol"
	for _, passthrough := range []bool{false, true} {
		for _, kind := range []string{"json", "chat", "responses"} {
			name := kind
			if passthrough {
				name += "/passthrough"
			}
			t.Run(name, func(t *testing.T) {
				payload := `{"model":"` + upstreamAlias + `","choices":[{"delta":{"content":"keep ` + upstreamAlias + `","tool_calls":[{"function":{"arguments":"{\"model\":\"` + upstreamAlias + `\"}"}}]}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`
				if kind == "responses" {
					payload = `{"type":"response.completed","response":{"id":"resp_1","model":"` + upstreamAlias + `","output":[],"usage":{"input_tokens":1,"output_tokens":1}}}`
				}
				want := strings.Replace(payload, `"model":"`+upstreamAlias+`"`, `"model":"`+publicModel+`"`, 1)
				contentType := "application/json"
				body := payload
				if kind != "json" {
					contentType = "text/event-stream"
					body = "data: " + payload + "\n\ndata: [DONE]\n\n"
				}
				rec := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(rec)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
				resp := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {contentType}}, Body: io.NopCloser(strings.NewReader(body))}
				svc := &OpenAIGatewayService{}
				account := &Account{ID: 1}
				var err error
				// originalModel == mappedModel：客户端请求的模型没有配置任何渠道映射。
				if kind == "json" {
					if passthrough {
						_, err = svc.handleNonStreamingResponsePassthrough(context.Background(), resp, c, account, publicModel, publicModel)
					} else {
						_, err = svc.handleNonStreamingResponse(context.Background(), resp, c, account, publicModel, publicModel)
					}
				} else {
					if passthrough {
						_, err = svc.handleStreamingResponsePassthrough(context.Background(), resp, c, account, time.Now(), publicModel, publicModel)
					} else {
						_, err = svc.handleStreamingResponse(context.Background(), resp, c, account, time.Now(), publicModel, publicModel)
					}
				}
				require.NoError(t, err)
				written := rec.Body.String()
				require.Contains(t, written, want)
				// 只改协议字段：正文与工具参数里的同名字符串保持原样。
				if kind != "responses" {
					require.Contains(t, written, `keep `+upstreamAlias)
					require.Contains(t, written, `{\"model\":\"`+upstreamAlias+`\"}`)
				}
				// 审计观测的是重写前的上游原始字节。
				require.Equal(t, upstreamAlias, observedUpstreamResponseModel(c))
			})
		}
	}
}

func TestRewriteClientModelGuards(t *testing.T) {
	svc := &OpenAIGatewayService{}
	body := `{"model":"alias","response":{"model":"alias"}}`

	// 公开名为空：不动。
	require.Equal(t, body, string(svc.rewriteClientModelInResponseBody([]byte(body), "mapped", "")))
	require.Equal(t, "data: "+body, svc.rewriteClientModelInSSELine("data: "+body, "mapped", ""))

	// 有映射：沿用上游行为。响应体只改顶层 model，SSE 行还会改 response.model。
	bodyWant := `{"model":"public","response":{"model":"alias"}}`
	lineWant := `data: {"model":"public","response":{"model":"public"}}`
	require.Equal(t, bodyWant, string(svc.rewriteClientModelInResponseBody([]byte(body), "mapped", "public")))
	require.Equal(t, lineWant, svc.rewriteClientModelInSSELine("data: "+body, "mapped", "public"))

	// 无映射：按上游自报名改写，结果与有映射时一致。
	require.Equal(t, bodyWant, string(svc.rewriteClientModelInResponseBody([]byte(body), "public", "public")))
	require.Equal(t, lineWant, svc.rewriteClientModelInSSELine("data: "+body, "", "public"))

	// 无映射且上游自报的就是公开名：不动。
	same := `{"model":"public","response":{"model":"public"}}`
	require.Equal(t, same, string(svc.rewriteClientModelInResponseBody([]byte(same), "public", "public")))
	require.Equal(t, "data: "+same, svc.rewriteClientModelInSSELine("data: "+same, "public", "public"))

	// 未声明模型的帧、非 JSON、非 data 行：不动。
	for _, line := range []string{
		`data: {"type":"response.output_text.delta","delta":"model"}`,
		`data: [DONE]`,
		`data: {"model":"alias"} trailing`,
		`event: response.completed`,
		``,
	} {
		require.Equal(t, line, svc.rewriteClientModelInSSELine(line, "public", "public"))
	}
	for _, raw := range []string{`{"text":"alias"}`, `{"model":42}`, `{"model":null}`, `{"model":"alias",`} {
		require.Equal(t, raw, string(svc.rewriteClientModelInResponseBody([]byte(raw), "public", "public")))
	}

	// 整段 SSE 文本逐行重写。
	sseBody := "data: " + body + "\n\ndata: [DONE]\n\n"
	require.Equal(t, lineWant+"\n\ndata: [DONE]\n\n", svc.rewriteClientModelInSSEBody(sseBody, "public", "public"))
	require.Equal(t, sseBody, svc.rewriteClientModelInSSEBody(sseBody, "public", ""))
}
