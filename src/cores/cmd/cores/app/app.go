package app

import (
	"cos-backend-com/src/common/app"
	"cos-backend-com/src/common/providers/session"
	"cos-backend-com/src/common/util"
	"cos-backend-com/src/cores"
	"cos-backend-com/src/cores/routers/bounties"
	"cos-backend-com/src/cores/routers/categories"
	"cos-backend-com/src/cores/routers/files"
	"cos-backend-com/src/cores/routers/follows"
	"cos-backend-com/src/cores/routers/hunters"
	"cos-backend-com/src/cores/routers/startups"
	"cos-backend-com/src/cores/routers/tags"
	"cos-backend-com/src/libs/auth"
	"cos-backend-com/src/libs/filters"
	filesSdk "cos-backend-com/src/libs/sdk/files"
	"net/http"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/mediocregopher/radix.v2/pool"
	s "github.com/wujiu2020/strip"
)

const (
	AppName = "cores"
)

func AppInit(tea *s.Strip, confPath string, files ...string) *appConfig {
	app := &appConfig{app.New(tea, AppName), cores.Env}
	app.ConfigLoad(app.Env, confPath, files...)
	app.ConfigCheck()
	app.ConfigDB()
	app.ConfigProviders()
	app.ConfigFilters()
	app.ConfigRoutes()
	app.ConfigDone()
	return app
}

type appConfig struct {
	app.AppConfig
	Env *cores.Config
}

// 配置完毕，做一些运行时检查与初始化
func (p *appConfig) Start() {
}

func (p *appConfig) ConfigCheck() {
}

func (p *appConfig) ConfigDB() *sqlx.DB {
	return p.ConnectDB()
}

func (p *appConfig) ConfigProviders() {
	redisPool, err := pool.NewCustom("tcp",
		p.Env.Redis.Addr,
		p.Env.Redis.PoolSize,
		util.RedisDialWithSecret(p.Env.Redis.Secret),
	)
	if err != nil {
		p.Logger().Error("redis pool.New:", err)
		os.Exit(1)
	}
	p.Provide(redisPool)

	if err := session.ConfigSessions(p.Strip, redisPool, p.Env.Session); err != nil {
		p.Logger().Error("config sessions:", err)
		os.Exit(1)
	}

	var rt http.RoundTripper
	if err := p.Injector().Find(&rt, ""); err != nil {
		panic(err)
	}
	oauth2TokenURL := p.Env.Service.Account + "/oauth2/token"
	p.Provide(auth.AuthTransportProvider(oauth2TokenURL))
	p.Provide(filesSdk.NewFileService(filesSdk.BaseConfig{
		cores.Env.Minio.Endpoint,
		cores.Env.Minio.Secure,
		cores.Env.Minio.AccessKey,
		cores.Env.Minio.SecretKey,
	}))
}

func (p *appConfig) ConfigFilters() {
}

func (p *appConfig) ConfigRoutes() {
	p.Routers(util.VersionRouter())
	p.Routers(
		s.Router("/startups",
			s.Get(startups.StartUpsHandler{}).Action("List"),
			s.Router("/:id",
				s.Get(startups.StartUpsHandler{}).Action("Get"),
			),
		),

		s.Router("/startups",
			s.Filter(filters.LoginRequiredInner),
			s.Post(startups.StartUpsHandler{}).Action("Create"),
			s.Router("/me",
				s.Get(startups.StartUpsHandler{}).Action("ListMe"),
				s.Router("/:id",
					s.Get(startups.StartUpsHandler{}).Action("GetMe"),
				),
				s.Router("/followed",
					s.Get(startups.StartUpsHandler{}).Action("ListMeFollowed"),
				),
			),
			s.Router("/prepareId",
				s.Get(startups.StartUpsHandler{}).Action("GetPrepareId"),
			),
			s.Router("/:id",
				s.Put(startups.StartUpsHandler{}).Action("Update"),
				s.Router("/settings",
					s.Put(startups.StartUpSettingsHandler{}).Action("Update"),
				),
			),
			//restore startup
			s.Router("/:id:restore",
				s.Post(startups.StartUpsHandler{}).Action("Restore"),
			),
			//restore startup settings
			s.Router("/:id/settings:restore",
				s.Post(startups.StartUpSettingsHandler{}).Action("Restore"),
			),

			s.Router("/:id/follows",
				s.Post(follows.FollowsHandler{}).Action("Create"),
				s.Delete(follows.FollowsHandler{}).Action("Delete"),
			),
		),

		s.Router("/categories",
			s.Get(categories.CategoriesHandler{}).Action("List"),
			s.Router("/:id",
				s.Get(categories.CategoriesHandler{}).Action("Get"),
			),
		),

		s.Router("/tags",
			s.Get(tags.TagsHandler{}).Action("List"),
		),

		s.Router("/hunters",
			s.Router("/:id",
				s.Get(hunters.HuntersHandler{}).Action("Get"),
			),
		),

		s.Router("/hunters",
			s.Filter(filters.LoginRequiredInner),
			s.Post(hunters.HuntersHandler{}).Action("Create"),
			s.Put(hunters.HuntersHandler{}).Action("Update"),
			s.Router("/:id",
				s.Get(hunters.HuntersHandler{}).Action("Get"),
			),
			s.Router("/me",
				s.Get(hunters.HuntersHandler{}).Action("GetMe"),
			),
		),

		s.Router("/files",
			s.Filter(filters.LoginRequiredInner),
			s.Post(files.FilesHandler{}).Action("SignUploadFile"),
		),

		s.Router("/prepareId",
			s.Get(startups.StartUpsHandler{}).Action("GetPrepareId"),
		),

		s.Router("/startups/:id/bounties",
			s.Get(bounties.BountiesHandler{}).Action("ListStartupBounties"),
			s.Router("/me",
				s.Get(bounties.BountiesHandler{}).Filter(filters.LoginRequiredInner).Action("ListStartupBountiesMe"),
			),
			s.Post(bounties.BountiesHandler{}).Filter(filters.LoginRequiredInner).Action("Create"),
		),

		s.Router("/bounties",
			s.Get(bounties.BountiesHandler{}).Action("ListBounties"),
			s.Router("/me",
				s.Filter(filters.LoginRequiredInner),
				s.Get(bounties.BountiesHandler{}).Action("ListBountiesMe"),
			),
			s.Router("/:id",
				s.Get(bounties.BountiesHandler{}).Action("GetBounty"),
				s.Router("/me",
					s.Get(bounties.BountiesHandler{}).Filter(filters.LoginRequiredInner).Action("GetBountyMe"),
				),
				//s.Router("/:startWork",
				//	s.Get(bounties.BountiesHandler{}).Action("StartWork"),
				//),
			),
			s.Router("/users/:userId",
				s.Get(bounties.BountiesHandler{}).Action("ListUserBounties"),
			),
		),
	)
}

func (p *appConfig) ConfigDone() {
}
