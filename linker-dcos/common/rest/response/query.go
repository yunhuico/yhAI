package response

type QueryStruct struct {
	Success bool        `json:"success"`
	Count   interface{} `json:"count,omitempty"`
	Prev    string      `json:"prev_url,omitempty"`
	Next    string      `json:"next_url,omitempty"`
	Data    interface{} `json:"data"`
}

type UpdateStruct struct {
	Created bool   `json:"created"`
	Url     string `json:"url"`
}
