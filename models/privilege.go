package models

const (
	TblPrivilege string = "privilege"
)

type Privilege struct {
	RoleMenuId     int64
	ID             int64
	Name           string
	Description    string
	AccessControls []AccessControl
}
