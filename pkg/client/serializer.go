package client

import (
	"encoding/json"
	"fmt"
)

// CoreSerializer handles serialization/deserialization
type CoreSerializer struct{}

// SerializeMessage serializes a stream message
func (s *CoreSerializer) SerializeMessage(message interface{}) (string, error) {
	data, err := json.Marshal(message)
	if err != nil {
		fallback := map[string]interface{}{
			"error": fmt.Sprintf("Serialization failed: %s", err.Error()),
		}
		data, _ = json.Marshal(fallback)
	}

	return string(data), nil
}

// DeserializeMessage deserializes a stream message
func (s *CoreSerializer) DeserializeMessage(jsonStr string) (*StreamMessage, error) {
	var msg StreamMessage
	if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return &msg, nil
}

// DeserializeObject deserializes a JSON object
func (s *CoreSerializer) DeserializeObject(jsonResp interface{}, reconstruct bool) interface{} {
	return jsonResp
}

// NewCoreSerializer creates a new core serializer
func NewCoreSerializer() *CoreSerializer {
	return &CoreSerializer{}
}
