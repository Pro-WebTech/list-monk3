package impl

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
	"log"
	"strings"
)

func NewPrivilegeDaoImpl(lo *log.Logger) *PrivilegeDaoImpl {
	return &PrivilegeDaoImpl{lo: lo}
}

type PrivilegeDaoImpl struct{ lo *log.Logger }

func (u *PrivilegeDaoImpl) FindByRoleID(db *sqlx.DB, roleID int64) (entities []models.Privilege, err error) {
	entities = []models.Privilege{}
	var buf strings.Builder
	buf.WriteString("SELECT pr.id role_menu_id, mn.id, mn.name menu_name, mn.description FROM ")
	buf.WriteString(models.TblPrivilege)
	buf.WriteString(" pr, menu mn where pr.menu_id = mn.id and pr.role_id = $1 ")
	bind := []interface{}{roleID}

	rows, err := db.Query(buf.String(), bind...)

	if err != nil {
		u.lo.Printf("err FindByRoleID: %v", err)
		return entities, err
	}

	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()

	var menuIds []int64
	for rows.Next() {
		var roleMenuId int64
		var id int64
		var name string
		var description string

		_ = rows.Scan(
			&roleMenuId,
			&id,
			&name,
			&description)

		menuIds = append(menuIds, roleMenuId)
		entities = append(entities, models.Privilege{
			RoleMenuId:     roleMenuId,
			ID:             id,
			Name:           name,
			Description:    description,
			AccessControls: nil,
		})
	}

	var buf2 strings.Builder
	buf2.WriteString("select pra.role_menu_id, mna.id menu_access_id, mna.access menu_access, ")
	buf2.WriteString(" mna.control from privilege pr, privilege_access_control pra, menu mn, menu_access_control mna ")
	buf2.WriteString(" where pr.id = pra.role_menu_id and pr.menu_id = mn.id and pra.menu_access_control = mna.id and pra.role_menu_id = any($1) order by pra.role_menu_id, mna.id ")
	bind2 := []interface{}{pq.Array(menuIds)}

	rows, err = db.Query(buf2.String(), bind2...)
	if err != nil {
		u.lo.Printf("err Select menuIds: %v", err)
		return entities, err
	}

	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()

	privilegeAccessControl := map[int64][]models.AccessControl{}
	for rows.Next() {
		var roleMenuId int64
		var menuAccessId int64
		var menuAccess string
		var control string

		_ = rows.Scan(
			&roleMenuId,
			&menuAccessId,
			&menuAccess,
			&control)

		ac := models.AccessControl{
			Id:      menuAccessId,
			Access:  menuAccess,
			Control: control,
		}

		acl := privilegeAccessControl[roleMenuId]
		if acl == nil {
			acl = []models.AccessControl{}
		}

		acl = append(acl, ac)
		privilegeAccessControl[roleMenuId] = acl
	}

	if len(privilegeAccessControl) == 0 {
		return
	}

	var res []models.Privilege
	for _, entity := range entities {
		entity.AccessControls = privilegeAccessControl[entity.RoleMenuId]
		res = append(res, entity)
	}

	return res, nil
}
