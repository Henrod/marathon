package api

import (
	"fmt"
	"net/url"
	"time"

	"github.com/kataras/iris"
	"github.com/minio/minio-go"
	"github.com/uber-go/zap"
)

func UploadHandler(application *Application) func(c *iris.Context) {
	return func(c *iris.Context) {
		start := time.Now()
		l := application.Logger.With(
			zap.String("source", "uploadHandler"),
			zap.String("operation", "getUploadUrl"),
		)

		s3Bucket := application.Config.GetString("s3.bucket")
		s3Folder := application.Config.GetString("s3.folder")
		s3DaysExpiry := time.Second * 10 * 60 * 60
		s3AccessKeyID := application.Config.GetString("s3.accessKey")
		s3SecretAccessKey := application.Config.GetString("s3.secretAccessKey")
		enableSSL := true

		s3Key := fmt.Sprintf("%s/upload%v.csv", s3Folder, start.Unix())
		s3Client, err := minio.New("s3.amazonaws.com", s3AccessKeyID, s3SecretAccessKey, enableSSL)
		if err != nil {
			l.Error(
				"Failed to create S3 client.",
				zap.Error(err),
				zap.Duration("duration", time.Now().Sub(start)),
			)
			FailWith(400, err.Error(), c)
			return
		}

		u, err := s3Client.PresignedPutObject(s3Bucket, s3Key, s3DaysExpiry)
		if err != nil {
			l.Error(
				"Failed to create presigned PUT policy.",
				zap.Error(err),
				zap.Duration("duration", time.Now().Sub(start)),
			)
			FailWith(400, err.Error(), c)
			return
		}
		l.Info(
			"Upload handler retrieved url and params successfully.",
			zap.Duration("duration", time.Now().Sub(start)),
		)
		m := make(map[string]interface{})
		// For some reason we need to unescape it
		uURL, err := url.QueryUnescape(u.String())
		if err != nil {
			l.Error(
				"Failed to unescape presigned URL.",
				zap.Error(err),
				zap.Duration("duration", time.Now().Sub(start)),
			)
			FailWith(500, err.Error(), c)
			return
		}
		m["url"] = uURL
		SucceedWith(m, c)
	}
}
