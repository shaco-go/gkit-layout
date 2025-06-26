package gkit_gorm

import (
	"fmt"
)

var DefaultDSN = func() *DSN {
	return &DSN{
		Username:  "root",
		Password:  "123456",
		Host:      "127.0.0.1",
		Port:      3306,
		Char:      "utf8mb4",
		ParseTime: true,
		Loc:       "Local",
	}
}

type DSN struct {
	Username  string
	Password  string
	Host      string
	Port      int
	DBName    string
	Char      string
	ParseTime bool
	Loc       string
}

func (d *DSN) String() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%v&loc=%s",
		d.Username, d.Password, d.Host, d.Port, d.DBName, d.Char, d.ParseTime, d.Loc)
}
