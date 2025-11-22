package db

import (
	"log"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
	_ "github.com/go-sql-driver/mysql"
)

func init() {
	dsn, err := web.AppConfig.String("sqlconn")
	if err != nil {
		log.Fatalf("Failed to get database connection string from config: %v", err)
	}

	orm.RegisterDriver("mysql", orm.DRMySQL)
	err = orm.RegisterDataBase("default", "mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to register database: %v", err)
	}

	err = orm.RunSyncdb("default", true, true)
	if err != nil {
		log.Fatalf("Failed to sync database schema: %v", err)
	}

	log.Println("Database initialized successfully")
}
