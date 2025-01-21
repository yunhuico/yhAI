package payload

type CreateUserTourReq struct {
	Path   string `json:"path"`
	Step   string `json:"step"`
	Status string `json:"status"` // skipped,complete,processing
}

type UpdateUserTourReq struct {
	ID     string `json:"id"`
	Step   string `json:"step"`
	Status string `json:"status"` // skipped,complete,processing
}

type CreateTourConfigReq struct {
	Path        string `json:"path"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateTourConfigReq struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateTourStepConfigReq struct {
	ID        string `json:"id"`
	ElementID string `json:"elementId"`
}

type CreateTourStepConfigReq struct {
	TourConfigID string `json:"tourConfigId"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Index        int    `json:"index"`
	Status       string `json:"status"`
	ElementID    string `json:"elementId"`
}
