package core

import "encoding/json"

// CommandMessage: Java 서버와 통신하는 표준 JSON 포맷
type CommandMessage struct {
	CmdType   string          `json:"cmdType"`   // 예: CREATE_CONTAINER
	TraceID   string          `json:"traceId"`   // 추적용 ID
	Payload   json.RawMessage `json:"payload"`   // 실제 데이터
	Timestamp int64           `json:"timestamp"` // Unix Time
}
