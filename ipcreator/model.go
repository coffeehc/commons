package ipcreator

type ipRange struct {
	MinStr string `json:"min,omitempty"`
	MaxStr string `json:"max,omitempty"`
	Max    int64  `json:"-"`
	Min    int64  `json:"-"`
}
