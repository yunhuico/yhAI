package usermgmt

import (
	"github.com/emicklei/go-restful"
	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/user/services"
)

var ParamID = dao.ParamID // mongo id parameter

// Creates and adds documents resource to container
func Register(container *restful.Container, cors bool) {
	dc := Resource{}
	dc.Register(container, cors)
}

// Adds documents resource to container
func (p Resource) Register(container *restful.Container, cors bool) {
	wss := []*restful.WebService{}
	wss = append(wss,
		p.UserService(),
		p.TenantService(),
		p.RoleService(),
		p.TokenService())

	for _, ws := range wss {
		// Cross Origin Resource Sharing filter
		if cors {
			corsRule := restful.CrossOriginResourceSharing{ExposeHeaders: []string{"Content-Type"}, CookiesAllowed: false, Container: container}
			ws.Filter(corsRule.Filter)
		}
		// Add webservice to container
		container.Add(ws)
	}

	p.initServices()

}

func (p Resource) initServices() {
	services.GetTenantService()
	services.GetTokenService()
	services.GetUserService()
	services.GetRoleService()
	services.GetAuthService()
}
