package message

type Error struct {
	Code      int    `json:"code"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Href      string `json:"href"`
	ErrorData struct {
		Details string `json:"details"`
	} `json:"error_data"`
}
