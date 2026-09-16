package webadmin

type HomeData struct {
	Status bool   `json:"status"`
	Time   string `json:"time"`
	Info   string `json:"info"`
}
