//go:build !freebsd

package coder

import (
	jsoniter "github.com/json-iterator/go"
)

func (jsonCoder) Marshal(obj interface{}) ([]byte, error) {
	//return sonic.Marshal(obj)
	//return json.Marshal(obj)
	return jsoniter.Marshal(obj)
}
func (jsonCoder) Unmarshal(data []byte, target interface{}) error {
	//return json.Unmarshal(data, target)
	//return sonic.Unmarshal(data, target)
	return jsoniter.Unmarshal(data, target)
}
