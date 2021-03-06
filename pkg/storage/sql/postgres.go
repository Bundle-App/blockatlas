package sql

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/Bundle-App/blockatlas/pkg/logger"
	"time"
)

type PgSql struct {
	sql
}

func (db *PgSql) Init(host string) {
	client, err := gorm.Open("postgres", host)
	if err != nil {
		logger.Fatal(err, "postgress connection failed")
	}
	client.DB().SetMaxIdleConns(20)
	client.DB().SetMaxOpenConns(100)
	client.DB().SetConnMaxLifetime(time.Minute)
	client.LogMode(true)
	db.Client = client
}

func (db *PgSql) IsReady() bool {
	return db.Client != nil
}

func (db *PgSql) Close() error {
	return db.Client.Close()
}
