//go:build !freebsd

package coder

import (
	"github.com/bytedance/sonic"
)

func (jsonCoder) Marshal(obj interface{}) ([]byte, error) {
	return sonic.Marshal(obj)
	//return json.Marshal(obj)
}
func (jsonCoder) Unmarshal(data []byte, target interface{}) error {
	//return json.Unmarshal(data, target)
	return sonic.Unmarshal(data, target)
}
