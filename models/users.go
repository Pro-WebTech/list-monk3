package models

const (
	TblUsers string = "users"
)

type Users struct {
	Id          int64  `json:"id"`
	Email       string `json:"email"`
	Pass        string `json:"pass"`
	Username    string `json:"username"`
	RoleId      int64  `json:"role_id"`
	AccessLevel string `json:"access_level"`
	TokenJwt    string `json:"token_jwt"`
	Active      int    `json:"active"`
}

type UserReq struct {
	Email string `json:"email,omitempty"`
	Id    int64  `json:"id,omitempty"`
}
