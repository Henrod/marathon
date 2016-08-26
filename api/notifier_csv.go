package api

import (
	"time"

	"git.topfreegames.com/topfreegames/marathon/messages"
	"git.topfreegames.com/topfreegames/marathon/models"
	"git.topfreegames.com/topfreegames/marathon/workers"

	"github.com/kataras/iris"
	"github.com/satori/go.uuid"
	"github.com/uber-go/zap"
)

type csvNotificationPayload struct {
	App      string  `json:"app"`
	Service  string  `json:"service"`
	PageSize int     `json:"pageSize"`
	Message  message `json:"message"`
	Bucket   string  `json:"bucket"`
	Key      string  `json:"key"`
}

// SendCsvNotificationHandler is the handler responsible for creating new pushes
func SendCsvNotificationHandler(application *Application) func(c *iris.Context) {
	return func(c *iris.Context) {
		start := time.Now()
		notifierID := c.Param("notifierID")
		l := application.Logger.With(
			zap.String("source", "csvNotificationHandler"),
			zap.String("operation", "sendNotification"),
		)

		notifierIDUuid, err := uuid.FromString(notifierID)
		if err != nil {
			l.Error(
				"Could not convert notifierID into UUID.",
				zap.Error(err),
				zap.Duration("duration", time.Now().Sub(start)),
			)
			FailWith(400, err.Error(), c)
			return
		}

		notifier, err := models.GetNotifierByID(application.Db, notifierIDUuid)
		if err != nil {
			l.Error("Could not find notifier.", zap.Error(err), zap.Duration("duration", time.Now().Sub(start)))
			FailWith(400, err.Error(), c)
			return
		}

		var payload csvNotificationPayload
		if err := LoadJSONPayload(&payload, c, l); err != nil {
			l.Error(
				"Failed to parse json payload.",
				zap.Error(err),
				zap.Duration("duration", time.Now().Sub(start)),
			)
			FailWith(400, err.Error(), c)
			return
		}

		modifiers := [][]interface{}{{"LIMIT", payload.PageSize}}

		message := &messages.InputMessage{
			App:     payload.App,
			Service: payload.Service,
		}

		if payload.Message.Template != "" {
			message.Template = payload.Message.Template
		}
		if payload.Message.Params != nil {
			message.Params = payload.Message.Params
		}
		if payload.Message.Message != nil {
			message.Message = payload.Message.Message
		}
		if payload.Message.Metadata != nil {
			message.Metadata = payload.Message.Metadata
		}

		workerConfig := &workers.BatchCsvWorker{
			ConfigPath: application.ConfigPath,
			Logger:     l,
			Notifier:   notifier,
			Message:    message,
			Modifiers:  modifiers,
			Bucket:     payload.Bucket,
			Key:        payload.Key,
		}
		worker, err := workers.GetBatchCsvWorker(workerConfig)
		if err != nil {
			l.Error("Invalid worker config,", zap.Error(err), zap.Duration("duration", time.Now().Sub(start)))
			FailWith(400, err.Error(), c)
		}

		worker.Start()

		SucceedWith(map[string]interface{}{
			"id": worker.ID.String(),
		}, c)
	}
}
