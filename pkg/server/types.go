package server


type CallToolRequestExample struct {
	Name   string      `json:"name" example:"tool_name"`
	Params interface{} `json:"params"`
}
