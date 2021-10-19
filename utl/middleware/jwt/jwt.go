package jwt

import (
	"encoding/json"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/dao"
	"github.com/knadh/listmonk/models"
	"github.com/knadh/listmonk/utl/constants"
	"github.com/labstack/echo"
	"github.com/spf13/cast"
	"log"
	"strings"
	"time"

	"github.com/casbin/casbin"
	jwt "github.com/dgrijalva/jwt-go"

	jsonadapter "github.com/casbin/json-adapter"
)

// New generates new JWT service necessery for auth middleware
func New(lo *log.Logger, db *sqlx.DB, rdb dao.RDB, pdb dao.PDB, secret, algo string, d int) *Service {
	signingMethod := jwt.GetSigningMethod(algo)
	if signingMethod == nil {
		panic("invalid jwt signing method")
	}
	svc := &Service{
		key:      []byte(secret),
		algo:     signingMethod,
		duration: time.Duration(d) * time.Minute,
		policy:   []jsonadapter.CasbinRule{},
		db:       db,
		pdb:      pdb,
		rdb:      rdb,
		lo:       lo,
	}

	svc.policy = []jsonadapter.CasbinRule{}
	svc.LoadInterceptor()
	return svc
}

// Service provides a Json-Web-Token authentication implementation
type Service struct {
	// Secret key used for signing.
	key []byte

	// Duration for which the jwt token is valid.
	duration time.Duration

	// JWT signing algorithm
	algo jwt.SigningMethod

	enforcer *casbin.Enforcer
	policy   []jsonadapter.CasbinRule
	db       *sqlx.DB
	rdb      dao.RDB
	pdb      dao.PDB
	lo       *log.Logger
}

// MWFunc makes JWT implement the Middleware interface.
func (j *Service) MWFunc() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token, err := j.ParseToken(c)
			if err != nil || !token.Valid {
				return echo.ErrUnauthorized
			}

			claims := token.Claims.(jwt.MapClaims)

			if claims["id"] != nil {
				id := int64(claims["id"].(float64))
				c.Set("id", id)
			}
			if claims["n"] != nil {
				name := claims["n"].(string)
				c.Set("name", name)
			}

			if claims["e"] != nil {
				email := claims["e"].(string)
				c.Set("email", email)
			}

			var role int64
			if claims["r"] != nil {
				role = int64(claims["r"].(float64))
				c.Set("role", role)
			}

			if j.skipCheckAcl(c) {
				return next(c)
			}

			if !j.checkAcl(c, role) {
				return echo.ErrForbidden
			}

			return next(c)
		}
	}
}

func (j *Service) skipCheckAcl(c echo.Context) bool {
	if strings.HasPrefix(c.Request().URL.String(), "/v1/api/checkout/") ||
		strings.HasPrefix(c.Request().URL.String(), "/v1/api/config") ||
		strings.HasPrefix(c.Request().URL.String(), "/v1/api/lang/") ||
		strings.HasPrefix(c.Request().URL.String(), "/v1/api/initsettings") ||
		strings.HasPrefix(c.Request().URL.String(), "/v1/api/initlists") {
		return true
	}
	return false
}

func (j *Service) checkAcl(c echo.Context, roleId int64) (next bool) {
	method := c.Request().Method
	path := c.Request().URL.Path
	return j.enforcer.Enforce(cast.ToString(roleId), path, method)
}

func (j *Service) LoadInterceptor() {
	roleMenu, err := j.FindAllMenuGroupByRole()
	if err != nil {
		j.lo.Println("err LoadInterceptor: ", err)
		return
	}

	j.policy = loadPolicy(roleMenu)
	j.newEnforcer()
}

func (j *Service) FindAllMenuGroupByRole() (out []RoleMenu, err error) {
	out = []RoleMenu{}
	roleList, err := j.rdb.FindAll(j.db)
	if err != nil {
		j.lo.Println("err FindAllMenuGroupByRole: ", err)
		return
	}

	for _, rl := range roleList {
		menus, err := j.FindMenuByRoleID(rl.ID)
		if err != nil {
			j.lo.Println("err FindMenuByRoleID: ", err)
			return nil, err
		}

		out = append(out, RoleMenu{
			Role: Role{
				ID:          rl.ID,
				Name:        rl.Name,
				Description: rl.Description,
			},
			Menu: menus,
		})
	}

	return out, nil
}

func (j *Service) FindMenuByRoleID(roleID int64) (out []Menu, err error) {
	out = []Menu{}
	entities, err := j.pdb.FindByRoleID(j.db, roleID)
	if err != nil {
		j.lo.Println("err FindMenuByRoleID: ", err)
		return
	}
	for _, entity := range entities {

		var listAccess []AccessControl
		for _, eac := range entity.AccessControls {
			ac := AccessControl{
				ID:      eac.Id,
				Access:  eac.Access,
				Control: eac.Control,
			}

			listAccess = append(listAccess, ac)
		}

		mn := Menu{
			ID:            entity.ID,
			Name:          entity.Name,
			Description:   entity.Description,
			AccessControl: listAccess,
		}

		out = append(out, mn)
	}

	return
}

func loadPolicy(roleMenu []RoleMenu) []jsonadapter.CasbinRule {
	policy := []jsonadapter.CasbinRule{}

	for _, role := range roleMenu {
		for _, menu := range role.Menu {
			for _, ac := range menu.AccessControl {
				rule := jsonadapter.CasbinRule{
					PType: "p",
					V0:    cast.ToString(role.Role.ID),
					V1:    ac.Access,
					V2:    "*",
				}

				switch ac.Control {
				case "read":
					{
						rule.V2 = "GET"
					}
				case "create":
					{
						rule.V2 = "POST"
					}
				case "update":
					{
						rule.V2 = "PUT"
					}
				case "delete":
					{
						rule.V2 = "DELETE"
					}
				default:
					rule.V2 = "*"
				}

				policy = append(policy, rule)
			}
		}
	}
	return policy
}

func (j *Service) newEnforcer() {
	b, _ := json.Marshal(j.policy)
	adp := jsonadapter.NewAdapter(&b)
	m := casbin.NewModel()
	m.AddDef("r", "r", "sub, obj, act")
	m.AddDef("p", "p", "sub, obj, act")
	m.AddDef("g", "g", "_, _")
	m.AddDef("e", "e", "some(where (p.eft == allow))")
	m.AddDef("m", "m", `g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && (r.act == p.act || p.act == "*")`)
	j.enforcer = casbin.NewEnforcer(m, adp)
	//j.enforcer = casbin.NewEnforcer("auth_model.conf", adp)
}

// ParseToken parses token from Authorization header
func (j *Service) ParseToken(c echo.Context) (*jwt.Token, error) {

	token := c.Request().Header.Get("Authorization")
	if token == "" {
		return nil, constants.ErrGeneric
	}
	parts := strings.SplitN(token, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		return nil, constants.ErrGeneric
	}

	return jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
		if j.algo != token.Method {
			return nil, constants.ErrGeneric
		}
		return j.key, nil
	})

}

// GenerateToken generates new JWT token and populates it with user data
func (j *Service) GenerateToken(u *models.Users) (string, string, error) {
	expire := time.Now().Add(j.duration)

	token := jwt.NewWithClaims((j.algo), jwt.MapClaims{
		"id":  u.Id,
		"e":   u.Email,
		"n":   u.Username,
		"r":   u.RoleId,
		"exp": expire.Unix(),
	})

	tokenString, err := token.SignedString(j.key)

	return tokenString, expire.Format(constants.LayDateTimeISO), err
}
