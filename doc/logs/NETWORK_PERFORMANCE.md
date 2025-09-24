# 网络性能日志格式说明

## 日志文件位置
`log/network_performance.txt`

## 日志格式

### 主进程日志格式：
```
时间戳 | 提供商 | 模型 | Input:字符数 chars | Network:网络耗时 | Total:总耗时 | Ratio:网络占比% | Token信息 | 状态
```

### Task子进程日志格式：
```
时间戳 | 提供商 | 模型 | Input:字符数 chars | Network:网络耗时 | Token信息 | Source:TASK | 状态
```

## 字段说明

- **时间戳**: `2006/01/02 15:04:05` 格式
- **提供商**: `OpenAI`、`Anthropic` 等
- **模型**: `gpt-4`、`claude-3-5-sonnet-20241022` 等
- **Input**: 输入文本的字符数
- **Network**: 纯网络请求耗时
- **Total**: 总处理时间（仅主进程有）
- **Ratio**: 网络耗时占总时间的百分比（仅主进程有）
- **Token信息**: `Prompt:数量|Completion:数量|Total:数量` 或 `Token:N/A`
- **Source**: `TASK` 标识来自Task子进程（仅子进程有）
- **状态**: `SUCCESS` 或 `ERROR: 错误信息`

## Token信息详解

Token使用数据来自`general.Usage`结构体：
```go
type Usage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`  
    TotalTokens      int `json:"total_tokens"`
}
```

- **Prompt Tokens**: 发送给AI的输入token数量
- **Completion Tokens**: AI生成的回复token数量  
- **Total Tokens**: 总token消耗（Prompt + Completion）

## 日志示例

### 主进程成功请求：
```
2024/09/24 15:30:45 | OpenAI | gpt-4 | Input:126 chars | Network:2.5s | Total:2.8s | Ratio:89.3% | Prompt:45|Completion:120|Total:165 | SUCCESS
```

### Task子进程请求：
```
2024/09/24 15:31:10 | OpenAI | gpt-4 | Input:89 chars | Network:1.8s | Prompt:32|Completion:85|Total:117 | Source:TASK | SUCCESS
```

### 错误请求：
```
2024/09/24 15:32:15 | OpenAI | gpt-4 | Input:234 chars | Network:30s | Total:30.1s | Ratio:99.7% | Token:N/A | ERROR: context deadline exceeded
```

## 性能分析用途

1. **网络延迟监控**: 通过Network字段监控API响应时间
2. **Token消耗分析**: 统计不同类型请求的token使用情况
3. **成本控制**: 根据Token数量估算API调用成本
4. **性能优化**: 分析网络占比，识别瓶颈所在
5. **错误追踪**: 记录失败的请求及其原因

## 数据处理建议

可以使用简单的脚本或工具来分析这些日志：
- 统计平均网络延迟
- 计算总Token消耗和成本
- 分析成功率和错误类型
- 识别性能异常的时间段