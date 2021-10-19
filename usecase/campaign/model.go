package campaign

type MessengersResponse struct {
	Messenger string        `json:"messenger"`
	Name      string        `json:"name"`
	Product   []ListProduct `json:"product"`
}

type ListProduct struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
