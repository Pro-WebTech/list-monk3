package models

import "time"

const (
	TblRole string = "role"
)

type RoleEntity struct {
	ID           int64
	Name         string
	Description  string
	Status       int64
	ParentRoleID int64
	CreatedBy    string
	CreatedDate  time.Time
	UpdatedBy    string
	UpdatedDate  time.Time
}

type RoleMenu struct {
	Role *Role   `json:"role,omitempty"`
	Menu []*Menu `json:"menu,omitempty"`
}

type Role struct {
	Id          int64  `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type Menu struct {
	Id            int64            `json:"id,omitempty"`
	Name          string           `json:"name,omitempty"`
	Description   string           `json:"description,omitempty"`
	AccessControl []*AccessControl `json:"accessControl,omitempty"`
}

type AccessControl struct {
	Id      int64  `json:"id,omitempty"`
	Access  string `json:"access,omitempty"`
	Control string `json:"control,omitempty"`
}
