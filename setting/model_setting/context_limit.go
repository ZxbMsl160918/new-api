package model_setting

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

// ContextLimitSettings 按模型名设置上下文窗口上限（token 数）。
// 未配置的模型不做限制；估算的输入 token 与请求的 max_tokens 之和超过上限时拒绝请求。
type ContextLimitSettings struct {
	ModelContextLimits map[string]int `json:"model_context_limits"`
}

var contextLimitSettings = ContextLimitSettings{
	// 初始化为空 map：配置导出/前端展示为 "{}"，而不是 nil map 序列化出的 "null"。
	ModelContextLimits: map[string]int{},
}

func init() {
	config.GlobalConfig.Register("context_limit", &contextLimitSettings)
}

func GetContextLimitSettings() *ContextLimitSettings {
	return &contextLimitSettings
}

// GetModelContextLimit 返回模型配置的上下文窗口上限；未配置时第二个返回值为 false。
func GetModelContextLimit(model string) (int, bool) {
	limit, ok := contextLimitSettings.ModelContextLimits[model]
	if !ok || limit <= 0 {
		return 0, false
	}
	return limit, true
}

func ValidateModelContextLimits(value string) error {
	var limits map[string]int
	if err := common.UnmarshalJsonStr(value, &limits); err != nil {
		return fmt.Errorf("模型上下文限制必须是 {\"模型名\": token数} 形式的 JSON: %s", err.Error())
	}
	for model, limit := range limits {
		if model == "" {
			return fmt.Errorf("模型上下文限制中的模型名不能为空")
		}
		if limit <= 0 {
			return fmt.Errorf("模型 %s 的上下文窗口限制必须为正整数，当前为 %d", model, limit)
		}
	}
	return nil
}
