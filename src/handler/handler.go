package handler

import (
	"github.com/sirupsen/logrus"
	"server/logging"
)

var log *logrus.Logger

func init() {
	log = logging.GetLogger()
}
