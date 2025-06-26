package global

import (
	"github.com/rs/zerolog"
	"github.com/shaco-go/gkit-layout/configs"
	"gorm.io/gorm"
)

var (
	Conf *configs.Config
	DB   *gorm.DB
	Log  zerolog.Logger
)
