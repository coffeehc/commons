//go:build freebsd

package coder

import "encoding/json"

func (jsonCoder) Marshal(obj interface{}) ([]byte, error) {
	return json.Marshal(obj)
}
func (jsonCoder) Unmarshal(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}
