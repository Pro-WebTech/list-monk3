package jwt

type Role struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RoleMenu struct {
	Role
	Menu []Menu `json:"menu"`
}

type AccessControl struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
	Access      string `json:"access"`
	Control     string `json:"control"`
}

type Menu struct {
	ID            int64           `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	AccessControl []AccessControl `json:"accessControl"`
}
