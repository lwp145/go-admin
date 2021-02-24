package database

import (
	"gorm.io/plugin/dbresolver"
	. "log"
	"time"

	logCore "github.com/go-admin-team/go-admin-core/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"go-admin/common/global"
	"go-admin/common/log"
	mycasbin "go-admin/pkg/casbin"
	"go-admin/tools"
	toolsConfig "go-admin/tools/config"
)

// Setup 配置数据库
func Setup() {
	for k := range toolsConfig.DatabasesConfig {
		setupSimpleDatabase(k, toolsConfig.DatabasesConfig[k])
	}
}

func setupSimpleDatabase(host string, c *toolsConfig.Database) {
	if global.Driver == "" {
		global.Driver = c.Driver
	}
	log.Infof("%s => %s", host, tools.Green(c.Source))
	db, err := gorm.Open(open[c.Driver](c.Source), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		Logger: logger.New(
			New(logCore.DefaultLogger.Options().Out, "\r\n", LstdFlags),
			logger.Config{
				SlowThreshold: time.Second,
				Colorful:      true,
				LogLevel: logger.LogLevel(
					logCore.DefaultLogger.Options().Level.LevelForGorm()),
			},
		),
	})
	if err != nil {
		log.Fatal(tools.Red(c.Driver+" connect error :"), err)
	} else {
		log.Info(tools.Green(c.Driver + " connect success !"))
	}
	register := resolver(c, db)
	if register != nil {
		err = db.Use(register)
		if err != nil {
			log.Fatal(tools.Red(c.Driver+" connect DBResolver config error :"), err)
		}
	}

	e := mycasbin.Setup(db, "sys_")

	if host == "*" {
		global.Eloquent = db
	}

	global.Cfg.SetDb(host, db)
	global.Cfg.SetCasbin(host, e)
}

// resolver 支持DBResolver，读写分离请不要设置policy，⚠️数据同步问题请自己妥善解决
func resolver(c *toolsConfig.Database, db *gorm.DB) *dbresolver.DBResolver {
	register := dbresolver.Register(dbresolver.Config{})
	for i := range c.Registers {
		var config dbresolver.Config
		if len(c.Registers[i].Sources) > 0 {
			config.Sources = make([]gorm.Dialector, len(c.Registers[i].Sources))
			for _, dsn := range c.Registers[i].Sources {
				config.Sources[i] = open[c.Driver](dsn)
			}
		}
		if len(c.Registers[i].Replicas) > 0 {
			config.Replicas = make([]gorm.Dialector, len(c.Registers[i].Replicas))
			for _, dsn := range c.Registers[i].Replicas {
				config.Replicas[i] = open[c.Driver](dsn)
			}
		}
		if c.Registers[i].Policy != "" {
			policy, ok := toolsConfig.Policies[c.Registers[i].Policy]
			if ok {
				config.Policy = policy
			}
		}
		if i == 0 || register == nil {
			register = dbresolver.Register(config)
			continue
		}
		register = register.Register(config)
	}
	if c.ConnMaxIdleTime > 0 {
		register = register.SetConnMaxIdleTime(time.Duration(c.ConnMaxIdleTime) * time.Second)
	}
	if c.ConnMaxLifetime > 0 {
		register = register.SetConnMaxLifetime(time.Duration(c.ConnMaxLifetime) * time.Second)
	}
	if c.MaxOpenConns > 0 {
		register = register.SetMaxOpenConns(c.MaxOpenConns)
	}
	if c.MaxIdleConns > 0 {
		register = register.SetMaxIdleConns(c.MaxIdleConns)
	}
	return register
}
