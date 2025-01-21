package types

type RespCommon struct {
	Success bool   `json:"success"`
	Errmsg  string `json:"errmsg"`
}

type RespPutRules struct {
	RespCommon
}
