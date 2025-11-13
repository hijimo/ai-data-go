
下面是结合 Model Context Protocol（MCP）中 “工具调用（Tool Invocation）” 与 “搜索/检索过程事件（Search/Resource Retrieval）” 机制，设计的一份 **工具调用规范（Markdown 格式）**。你可以将此作为基底，在自己的流式 EventStream 规范中融入工具调用 + 检索事件。

---

# 工具调用规范 (v1.0)

*基于 MCP + 流式 EventStream 扩展*

## 1. 适用范围

本规范用于支持大语言模型（LLM）在对话或任务流中，通过标准化事件流（EventStream）方式调用外部工具／搜索资源，并在流式输出（token 增量）过程中报告状态、结果、以及后续生成。

适用场景包括但不限于：

* 模型判定需调用工具（如 API／数据库／检索引擎）以补充上下文或执行动作。

* 模型生成工具调用参数增量并待结果返回。

* 检索资源（如向量检索、文档检索）过程也作为一种工具调用／资源获取事件。

* 工具结果被模型消费后继续生成文本响应。

## 2. 术语定义

* **Tool**：一个外部能力（例如 API、数据库、检索引擎、计算服务）暴露给模型调用。

* **Tool Use / Invocation**：模型决定并发起一次工具调用。

* **Resource Retrieval / Search**：一种特殊类型的工具调用，用于从外部系统检索数据或文档。

* **Delta**：增量输出，是模型输出或工具调用参数生成的片段。

* **EventStream**：基于 SSE（Server-Sent Events）或类似机制，用于实时推送事件至客户端。

* **Sequence**：事件序号，用于客户端重组流顺序。

* **TraceId**：追踪标识，用于日志、审计、调试。

## 3. 事件类型清单

| 事件类型                 | 语义                       | 是否必须      | 典型 `data` 结构简介                                                        |                       |

| -------------------- | ------------------------ | --------- | --------------------------------------------------------------------- | --------------------- |

| `init`               | 会话或流初始化（模型、会话、工具列表）      | ✅         | `{ model:"…", conversation_id:"…", tools:[…] }`                       |                       |

| `ping`               | 心跳维持连接                   | ⚙️        | `{ timestamp:… }`                                                     |                       |

| `tool_call_start`    | 模型开始发起工具调用               | ✅（当调用工具时） | `{ id:"tool-uuid", tool:"tool_name", role:"assistant", sequence:… }`  |                       |

| `tool_call_delta`    | 模型正在生成／补充工具调用参数（增量）      | ✅（参数流时）   | `{ id:"tool-uuid", args_delta:"…", sequence:… }`                      |                       |

| `tool_call_complete` | 模型工具调用参数生成完成             | ✅         | `{ id:"tool-uuid", tool:"tool_name", args:{…}, sequence:… }`          |                       |

| `search_start`       | 模型开始发起检索资源／搜索操作（可视为工具调用） | ⚙️        | `{ id:"search-uuid", query:"…", tool:"search_engine", sequence:… }`   |                       |

| `search_result`      | 检索工具返回初步结果               | ✅（检索时）    | `{ id:"search-uuid", results:[…], sequence:… }`                       |                       |

| `tool_result`        | 工具执行完成并返回结果              | ✅         | `{ id:"tool-uuid", result:{…} , status:"success                       | error", sequence:… }` |

| `delta`              | 模型继续生成文本／内容输出            | ✅（输出时）    | `{ id:"msg-uuid", role:"assistant", delta:"…", sequence:… }`          |                       |

| `complete`           | 模型当前消息或操作完成（非工具）         | ✅         | `{ id:"msg-uuid", status:"finished", finish_reason:"…", sequence:… }` |                       |

| `error`              | 发生错误（工具调用或检索或模型生成）       | ⚠️        | `{ code:"…", message:"…", sequence:… }`                               |                       |

| `done`               | 流式输出结束                   | ✅         | `[DONE]`                                                              |                       |

## 4. 统一数据载荷结构

所有 `data` 均为 JSON 对象，推荐字段如下（可根据具体系统裁剪）：

```json

{

"id": "uuid-string",          // 唯一标识一个消息或工具调用

"type": "tool_call|search_start|delta|…",

"sequence": 123,              // 事件顺序号

"trace_id": "trace-uuid",     // 链路追踪标识

"role": "assistant|user|system",  // 角色（模型输出、用户输入、系统）

"tool": "tool_name",           // 可选：当 type 属于工具调用相关

"args": { … },                 // 可选：工具调用参数

"args_delta": "…",             // 可选：参数增量

"query": "…",                  // 可选：检索查询

"results": [ … ],              // 可选：检索/工具返回的结果列表

"result": { … },               // 可选：工具返回单个结果

"status": "success|error|pending",  // 可选：工具调用状态

"content": {                    // 模型生成内容字段

"content_type": "text|image|json",

"parts": [ "…", "…" ]

  },

"finish_reason": "stop|length|tool_complete|…", // 可选：生成结束原因

"meta": {

"conversation_id": "conv-uuid",

"request_id": "req-uuid",

"timestamp": 1690000000

  }

}

```

## 5. 工具调用流程状态机

```text

tool_call_start → tool_call_delta* → tool_call_complete  

            ↓

        ⎯→ search_start → search_result  

            ↓

        ⎯→ tool_result  

            ↓

delta (模型消费结果并生成文本) → complete → done

```

说明：

* 模型首先通过 `tool_call_start` 标识要调用某工具。

* 若参数较多／生成过程复杂，可通过多条 `tool_call_delta` 增量输出。

* 参数确定后通过 `tool_call_complete` 发送完整 args。

* 如果是检索操作，则用 `search_start` + `search_result`。

* 工具实际执行后返回 `tool_result`。

* 模型接收后继续生成文本（`delta`）并最终结束（`complete`, `done`）。

## 6. 示例流程（Markdown 表示）

```

event: init

data: { "model":"gpt-4", "conversation_id":"conv-123" }

event: delta

data: { "id":"msg-1","role":"assistant","delta":"让我先查一下相关文档…","sequence":1 }

event: tool_call_start

data: { "id":"tool-A","tool":"search_docs","role":"assistant","sequence":2 }

event: tool_call_delta

data: { "id":"tool-A","args_delta":"{\"query\":\"MCP 工具 调用 规范\"","sequence":3 }

event: tool_call_complete

data: { "id":"tool-A","tool":"search_docs","args":{"query":"MCP 工具 调用 规范"},"sequence":4 }

event: tool_result

data: { "id":"tool-A","result":{"docs":[{"title":"Tools – Model Context Protocol","url":"…"}]},"status":"success","sequence":5 }

event: delta

data: { "id":"msg-1","role":"assistant","delta":"我找到了相关文档，章节描述了 tools/list 和 tools/call …","sequence":6 }

event: complete

data: { "id":"msg-1","status":"finished","finish_reason":"stop","sequence":7 }

event: done

data: [DONE]

```

## 7. 搜索资源（检索）扩展

* 检索操作可视为一种工具调用，只不过目标是获取资源（Documents, Embeddings, Vectors）而非执行动作。

* 建议使用 `search_start` + `search_result` 事件类型。

* `search_start` 中 `query` 字段用来说明检索意图；`search_result` 返回 `results` 数组，每项可包含标题、摘要、源链接、metadata。

* 模型可基于检索结果继续生成。

例如：

```

event: search_start

data: { "id":"search-1","tool":"doc_search","query":"MCP 工具调用 规范","sequence":8 }

event: search_result

data: { "id":"search-1","results":[{"title":"Tools – Model Context Protocol","url":"…","snippet":"Tools enable models …"}],"sequence":9 }

```

## 8. 安全与治理注意事项

* 所有工具调用前应支持 **人工确认** 或 **白名单控制**，避免模型滥用外部系统。MCP 文档亦建议保障“人在环”机制。 ([Model Context Protocol][1])

* 工具调用结果应进行 **审查／过滤**，避免将敏感数据直接返回给模型／用户。

* 每次调用应记录 `trace_id`, `request_id` 等日志用于审计。

* 对 `args`、`query` 等字段应验证参数合法性、权限、速率限制。

## 9. 扩展建议

* 支持 `tool_progress` 事件：当工具执行时间较长时，可报告进度（如 0–100%）。

* 支持并发多个工具调用：客户端可通过不同 `id` 并行接收。

* 支持 `tool_cancel` 事件：模型/客户端可取消某个正在执行的调用。

* 支持 `delta_encoding` 机制：对于大量参数或结果，可采用增量编码或二进制压缩。

* 支持多模态工具（例如图像生成、视频）字段：`content_type:"image"`、`parts:["base64…"]`。

## 10. 总结

本规范将模型调用外部工具、执行检索、以及模型生成输出的整个流程，整合进一个统一的流式事件体系。借助 MCP 原则（工具发现、调用、结果返回）与 EventStream 输出机制，可以构建出结构清晰、可追踪、可审计的对话/任务型系统。

建议将其作为你系统的“工具调用模块”规范，并在实际实现中根据业务需求（如响应延迟、并发能力、安全策略）做适当调整。

[1]: https://modelcontextprotocol.io/docs/concepts/tools?utm_source=chatgpt.com "Tools - Model Context Protocol"
