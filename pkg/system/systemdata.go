package system

type Data struct {
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	Constellation string   `json:"constellation"`
	Region        string   `json:"region"`
	Destinations  []string `json:"destinations"`
	SecStatus     string   `json:"sec_status"`
	Tags          []string `json:"tags"`
}
