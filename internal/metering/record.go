package metering

import (
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

var logger = logrus.StandardLogger().WithField("metering", true)

func RecordLogin(loginType string, userID uuid.UUID) {
	logger.WithFields(logrus.Fields{
		"action":       "login",
		"login_method": loginType,
		"instance_id":  uuid.Nil.String(),
		"user_id":      userID.String(),
	}).Info("Login")
}
