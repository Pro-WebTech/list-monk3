package models

const (
	TblSettings string = "settings"
)

type Settings struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SettingReq struct {
	Settings []Settings `json:"settings"`
}
