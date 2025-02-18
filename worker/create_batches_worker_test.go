/*
 * Copyright (c) 2016 TFG Co <backend@tfgco.com>
 * Author: TFG Co <backend@tfgco.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
 * the Software, and to permit persons to whom the Software is furnished to do so,
 * subject to the following conditions:
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
 * FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
 * COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package worker_test

import (
	"encoding/json"
	"strings"
	"time"

	workers "github.com/jrallison/go-workers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/satori/go.uuid"
	"github.com/topfreegames/marathon/extensions"
	"github.com/topfreegames/marathon/model"
	. "github.com/topfreegames/marathon/testing"
	"github.com/topfreegames/marathon/worker"
	"github.com/uber-go/zap"
)

var _ = Describe("CreateBatches Worker", func() {
	var app *model.App
	var template *model.Template
	var context map[string]interface{}

	config := GetConf()
	logger := zap.New(
		zap.NewJSONEncoder(zap.NoTime()),
		zap.FatalLevel,
	)
	w := worker.NewWorker(false, logger, GetConfPath())
	createBatchesWorker := worker.NewCreateBatchesWorker(config, logger, w)

	BeforeEach(func() {
		createBatchesWorker.S3Client = NewFakeS3()
		fakeData1 := []byte(`userids
9e558649-9c23-469d-a11c-59b05813e3d5
57be9009-e616-42c6-9cfe-505508ede2d0
a8e8d2d5-f178-4d90-9b31-683ad3aae920
5c3033c0-24ad-487a-a80d-68432464c8de
4223171e-c665-4612-9edd-485f229240bf
2df5bb01-15d1-4569-bc56-49fa0a33c4c3
67b872de-8ae4-4763-aef8-7c87a7f928a7
3f8732a1-8642-4f22-8d77-a9688dd6a5ae
21854bbf-ea7e-43e3-8f79-9ab2c121b941
843a61f8-45b3-44f9-9ab7-8becb2765653`)
		fakeData2 := []byte(`userids`)
		fakeData3 := []byte(`userids
e78431ca-69a8-4326-af1f-48f817a4a669
ee4455fe-8ff6-4878-8d7c-aec096bd68b4`)
		fakeData4 := []byte("remoteplayeridb00b2bf9-9999-4be9-bdbd-cf0dbbd82cb26ce8a64f-c888-48c4-a040-f24ca7a71714")
		extensions.S3PutObject(createBatchesWorker.Config, createBatchesWorker.S3Client, "test/jobs/obj1.csv", &fakeData1)
		extensions.S3PutObject(createBatchesWorker.Config, createBatchesWorker.S3Client, "test/jobs/obj2.csv", &fakeData2)
		extensions.S3PutObject(createBatchesWorker.Config, createBatchesWorker.S3Client, "test/jobs/obj3.csv", &fakeData3)
		extensions.S3PutObject(createBatchesWorker.Config, createBatchesWorker.S3Client, "test/jobs/obj4.csv", &fakeData4)
		app = CreateTestApp(createBatchesWorker.MarathonDB.DB)
		defaults := map[string]interface{}{
			"user_name":   "Someone",
			"object_name": "village",
		}
		body := map[string]interface{}{
			"alert": "{{user_name}} just liked your {{object_name}}!",
		}
		template = CreateTestTemplate(createBatchesWorker.MarathonDB.DB, app.ID, map[string]interface{}{
			"defaults": defaults,
			"body":     body,
			"locale":   "en",
		})
		context = map[string]interface{}{
			"user_name": "Everyone",
		}
		CreateTestJob(createBatchesWorker.MarathonDB.DB, app.ID, template.Name, map[string]interface{}{
			"context": context,
		})
		users := make([]worker.User, 2)
		for index := range users {
			id := uuid.NewV4().String()
			token := strings.Replace(uuid.NewV4().String(), "-", "", -1)
			users[index] = worker.User{
				UserID: id,
				Token:  token,
				Locale: "en",
			}
		}
		createBatchesWorker.RedisClient.FlushAll()
	})

	Describe("Process", func() {
		It("should panic if jobID is invalid", func() {
			m := map[string]interface{}{
				"jid":  2,
				"args": []string{"8df23db3-b02e-40a0-82b6-4993876c5fc8"},
			}
			smsg, err := json.Marshal(m)
			Expect(err).NotTo(HaveOccurred())
			msg, err := workers.NewMsg(string(smsg))
			Expect(err).NotTo(HaveOccurred())
			Expect(func() { createBatchesWorker.Process(msg) }).Should(Panic())
		})

		It("should panic if csvPath is invalid", func() {
			j := CreateTestJob(createBatchesWorker.MarathonDB.DB, app.ID, template.Name, map[string]interface{}{
				"context": context,
				"filters": map[string]interface{}{},
				"csvPath": "algum",
			})
			m := map[string]interface{}{
				"jid":  2,
				"args": []string{j.ID.String()},
			}
			smsg, err := json.Marshal(m)
			Expect(err).NotTo(HaveOccurred())
			msg, err := workers.NewMsg(string(smsg))
			Expect(err).NotTo(HaveOccurred())
			Expect(func() { createBatchesWorker.Process(msg) }).Should(Panic())
		})

		It("should not panic if csvPath and jobID are valid", func() {
			j := CreateTestJob(createBatchesWorker.MarathonDB.DB, app.ID, template.Name, map[string]interface{}{
				"context": context,
				"filters": map[string]interface{}{},
				"csvPath": "tfg-push-notifications/test/jobs/obj2.csv",
			})
			m := map[string]interface{}{
				"jid":  2,
				"args": []string{j.ID.String()},
			}
			smsg, err := json.Marshal(m)
			Expect(err).NotTo(HaveOccurred())
			msg, err := workers.NewMsg(string(smsg))
			Expect(err).NotTo(HaveOccurred())
			// Expect(func() {
			createBatchesWorker.Process(msg)
			//  }).ShouldNot(Panic())
		})

		It("should work if CSV is from Excel/Windows", func() {
			j := CreateTestJob(createBatchesWorker.MarathonDB.DB, app.ID, template.Name, map[string]interface{}{
				"context": context,
				"filters": map[string]interface{}{},
				"csvPath": "tfg-push-notifications/test/jobs/obj4.csv",
			})
			m := map[string]interface{}{
				"jid":  2,
				"args": []string{j.ID.String()},
			}
			smsg, err := json.Marshal(m)
			Expect(err).NotTo(HaveOccurred())
			msg, err := workers.NewMsg(string(smsg))
			Expect(err).NotTo(HaveOccurred())
			createBatchesWorker.Process(msg)
		})

		It("should do nothing if job status is stopped", func() {
			a := CreateTestApp(createBatchesWorker.MarathonDB.DB, map[string]interface{}{"name": "testapp"})
			j := CreateTestJob(createBatchesWorker.MarathonDB.DB, a.ID, template.Name, map[string]interface{}{
				"context": context,
				"filters": map[string]interface{}{},
				"csvPath": "tfg-push-notifications/test/jobs/obj1.csv",
			})
			_, err := createBatchesWorker.MarathonDB.DB.Model(&model.Job{}).Set("status = 'stopped'").Where("id = ?", j.ID).Update()
			Expect(err).NotTo(HaveOccurred())
			m := map[string]interface{}{
				"jid":  2,
				"args": []string{j.ID.String()},
			}
			smsg, err := json.Marshal(m)
			Expect(err).NotTo(HaveOccurred())
			msg, err := workers.NewMsg(string(smsg))
			Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
			res, err := createBatchesWorker.RedisClient.LLen("queue:process_batch_worker").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(BeEquivalentTo(0))
		})

		It("should create batches with the right tokens and tz and send to process_batches_worker", func() {
			a := CreateTestApp(createBatchesWorker.MarathonDB.DB, map[string]interface{}{"name": "testapp"})
			j := CreateTestJob(createBatchesWorker.MarathonDB.DB, a.ID, template.Name, map[string]interface{}{
				"context": context,
				"filters": map[string]interface{}{},
				"csvPath": "tfg-push-notifications/test/jobs/obj1.csv",
			})
			m := map[string]interface{}{
				"jid":  2,
				"args": []string{j.ID.String()},
			}
			smsg, err := json.Marshal(m)
			Expect(err).NotTo(HaveOccurred())
			msg, err := workers.NewMsg(string(smsg))
			Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
			res, err := createBatchesWorker.RedisClient.LLen("queue:process_batch_worker").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(BeEquivalentTo(2))
			job1, err := createBatchesWorker.RedisClient.LPop("queue:process_batch_worker").Result()
			Expect(err).NotTo(HaveOccurred())
			job2, err := createBatchesWorker.RedisClient.LPop("queue:process_batch_worker").Result()
			Expect(err).NotTo(HaveOccurred())
			j1 := map[string]interface{}{}
			j2 := map[string]interface{}{}
			err = json.Unmarshal([]byte(job1), &j1)
			Expect(err).NotTo(HaveOccurred())
			err = json.Unmarshal([]byte(job2), &j2)
			Expect(err).NotTo(HaveOccurred())
			Expect(j1["queue"].(string)).To(Equal("process_batch_worker"))
			Expect(j2["queue"].(string)).To(Equal("process_batch_worker"))
			Expect(len((j1["args"].([]interface{}))[2].([]interface{})) + len((j2["args"].([]interface{}))[2].([]interface{}))).To(BeEquivalentTo(10))
		})

		It("should create batches with the right tokens and tz and send to process_batches_worker if numPushes < dbPageSize", func() {
			createBatchesWorker.DBPageSize = 500
			a := CreateTestApp(createBatchesWorker.MarathonDB.DB, map[string]interface{}{"name": "testapp"})
			j := CreateTestJob(createBatchesWorker.MarathonDB.DB, a.ID, template.Name, map[string]interface{}{
				"context": context,
				"filters": map[string]interface{}{},
				"csvPath": "tfg-push-notifications/test/jobs/obj1.csv",
			})
			m := map[string]interface{}{
				"jid":  2,
				"args": []string{j.ID.String()},
			}
			smsg, err := json.Marshal(m)
			Expect(err).NotTo(HaveOccurred())
			msg, err := workers.NewMsg(string(smsg))
			createBatchesWorker.Process(msg)
			//Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
			res, err := createBatchesWorker.RedisClient.LLen("queue:process_batch_worker").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(BeEquivalentTo(2))
			job1, err := createBatchesWorker.RedisClient.LPop("queue:process_batch_worker").Result()
			Expect(err).NotTo(HaveOccurred())
			job2, err := createBatchesWorker.RedisClient.LPop("queue:process_batch_worker").Result()
			Expect(err).NotTo(HaveOccurred())
			j1 := map[string]interface{}{}
			j2 := map[string]interface{}{}
			err = json.Unmarshal([]byte(job1), &j1)
			Expect(err).NotTo(HaveOccurred())
			err = json.Unmarshal([]byte(job2), &j2)
			Expect(err).NotTo(HaveOccurred())
			Expect(j1["queue"].(string)).To(Equal("process_batch_worker"))
			Expect(j2["queue"].(string)).To(Equal("process_batch_worker"))
			Expect(len((j1["args"].([]interface{}))[2].([]interface{})) + len((j2["args"].([]interface{}))[2].([]interface{}))).To(BeEquivalentTo(10))
		})

		It("should skip batches if startsAt is past and pastTimeStrategy is skip", func() {
			a := CreateTestApp(createBatchesWorker.MarathonDB.DB, map[string]interface{}{"name": "testapp"})
			j := CreateTestJob(createBatchesWorker.MarathonDB.DB, a.ID, template.Name, map[string]interface{}{
				"context":          context,
				"filters":          map[string]interface{}{},
				"csvPath":          "tfg-push-notifications/test/jobs/obj1.csv",
				"localized":        true,
				"startsAt":         time.Now().UTC().Add(-12 * time.Hour).UnixNano(),
				"pastTimeStrategy": "skip",
			})
			m := map[string]interface{}{
				"jid":  2,
				"args": []string{j.ID.String()},
			}
			smsg, err := json.Marshal(m)
			Expect(err).NotTo(HaveOccurred())
			msg, err := workers.NewMsg(string(smsg))
			Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
			res, err := createBatchesWorker.RedisClient.ZCard("schedule").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(BeEquivalentTo(0))
		})

		It("should delay batches to next day if startsAt is past and pastTimeStrategy is nextDay", func() {
			a := CreateTestApp(createBatchesWorker.MarathonDB.DB, map[string]interface{}{"name": "testapp"})
			j := CreateTestJob(createBatchesWorker.MarathonDB.DB, a.ID, template.Name, map[string]interface{}{
				"context":          context,
				"filters":          map[string]interface{}{},
				"csvPath":          "tfg-push-notifications/test/jobs/obj1.csv",
				"localized":        true,
				"startsAt":         time.Now().UTC().Add(time.Duration(-6) * time.Hour).UnixNano(),
				"pastTimeStrategy": "nextDay",
			})
			m := map[string]interface{}{
				"jid":  2,
				"args": []string{j.ID.String()},
			}
			smsg, err := json.Marshal(m)
			Expect(err).NotTo(HaveOccurred())
			msg, err := workers.NewMsg(string(smsg))
			Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
			res, err := createBatchesWorker.RedisClient.ZCard("schedule").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(BeEquivalentTo(2))
			var data workers.EnqueueData
			jobs, err := createBatchesWorker.RedisClient.ZRange("schedule", 0, 2).Result()
			bytes, err := RedisReplyToBytes(jobs[0], err)
			Expect(err).NotTo(HaveOccurred())
			json.Unmarshal(bytes, &data)
			pushTime := time.Unix(0, int64(data.At*workers.NanoSecondPrecision))
			Expect(pushTime.After(time.Now())).To(Equal(true))
		})

		It("should schedule process_batches_worker if push is localized and starts in future", func() {
			a := CreateTestApp(createBatchesWorker.MarathonDB.DB, map[string]interface{}{"name": "testapp"})
			j := CreateTestJob(createBatchesWorker.MarathonDB.DB, a.ID, template.Name, map[string]interface{}{
				"context":   context,
				"filters":   map[string]interface{}{},
				"csvPath":   "tfg-push-notifications/test/jobs/obj1.csv",
				"localized": true,
				"startsAt":  time.Now().UTC().Add(12 * time.Hour).UnixNano(),
			})
			m := map[string]interface{}{
				"jid":  2,
				"args": []string{j.ID.String()},
			}
			smsg, err := json.Marshal(m)
			Expect(err).NotTo(HaveOccurred())
			msg, err := workers.NewMsg(string(smsg))
			Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
			res, err := createBatchesWorker.RedisClient.ZCard("schedule").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(BeEquivalentTo(2))
			var data workers.EnqueueData
			jobs, err := createBatchesWorker.RedisClient.ZRange("schedule", 0, 2).Result()
			bytes, err := RedisReplyToBytes(jobs[0], err)
			Expect(err).NotTo(HaveOccurred())
			json.Unmarshal(bytes, &data)
			pushTime := time.Unix(0, int64(data.At*workers.NanoSecondPrecision))
			Expect(pushTime.After(time.Now())).To(Equal(true))
		})

		It("should create batches with the right tokens and tz and send to process_batches_worker if a filter has multiple values separated bt comma", func() {
			a := CreateTestApp(createBatchesWorker.MarathonDB.DB, map[string]interface{}{"name": "testapp"})
			j := CreateTestJob(createBatchesWorker.MarathonDB.DB, a.ID, template.Name, map[string]interface{}{
				"context": context,
				"filters": map[string]interface{}{
					"locale": "pt,en",
				},
				"csvPath": "tfg-push-notifications/test/jobs/obj1.csv",
			})
			m := map[string]interface{}{
				"jid":  2,
				"args": []string{j.ID.String()},
			}
			smsg, err := json.Marshal(m)
			Expect(err).NotTo(HaveOccurred())
			msg, err := workers.NewMsg(string(smsg))
			Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
			res, err := createBatchesWorker.RedisClient.LLen("queue:process_batch_worker").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(BeEquivalentTo(2))
			job1, err := createBatchesWorker.RedisClient.LPop("queue:process_batch_worker").Result()
			Expect(err).NotTo(HaveOccurred())
			job2, err := createBatchesWorker.RedisClient.LPop("queue:process_batch_worker").Result()
			Expect(err).NotTo(HaveOccurred())
			j1 := map[string]interface{}{}
			j2 := map[string]interface{}{}
			err = json.Unmarshal([]byte(job1), &j1)
			Expect(err).NotTo(HaveOccurred())
			err = json.Unmarshal([]byte(job2), &j2)
			Expect(err).NotTo(HaveOccurred())
			Expect(j1["queue"].(string)).To(Equal("process_batch_worker"))
			Expect(j2["queue"].(string)).To(Equal("process_batch_worker"))
			Expect(len((j1["args"].([]interface{}))[2].([]interface{})) + len((j2["args"].([]interface{}))[2].([]interface{}))).To(BeEquivalentTo(10))
		})

		It("should create batches with the right tokens and tz and send to process_batches_worker if service is gcm", func() {
			a := CreateTestApp(createBatchesWorker.MarathonDB.DB, map[string]interface{}{"name": "testapp"})
			j := CreateTestJob(createBatchesWorker.MarathonDB.DB, a.ID, template.Name, map[string]interface{}{
				"context": context,
				"filters": map[string]interface{}{},
				"csvPath": "tfg-push-notifications/test/jobs/obj1.csv",
				"service": "gcm",
			})
			m := map[string]interface{}{
				"jid":  2,
				"args": []string{j.ID.String()},
			}
			smsg, err := json.Marshal(m)
			Expect(err).NotTo(HaveOccurred())
			msg, err := workers.NewMsg(string(smsg))
			Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
			res, err := createBatchesWorker.RedisClient.LLen("queue:process_batch_worker").Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(BeEquivalentTo(2))
			job1, err := createBatchesWorker.RedisClient.LPop("queue:process_batch_worker").Result()
			Expect(err).NotTo(HaveOccurred())
			job2, err := createBatchesWorker.RedisClient.LPop("queue:process_batch_worker").Result()
			Expect(err).NotTo(HaveOccurred())
			j1 := map[string]interface{}{}
			j2 := map[string]interface{}{}
			err = json.Unmarshal([]byte(job1), &j1)
			Expect(err).NotTo(HaveOccurred())
			err = json.Unmarshal([]byte(job2), &j2)
			Expect(err).NotTo(HaveOccurred())
			Expect(j1["queue"].(string)).To(Equal("process_batch_worker"))
			Expect(j2["queue"].(string)).To(Equal("process_batch_worker"))
			Expect(len((j1["args"].([]interface{}))[2].([]interface{})) + len((j2["args"].([]interface{}))[2].([]interface{}))).To(BeEquivalentTo(9))
		})
	})

	It("should not panic if job is a reexecution", func() {
		a := CreateTestApp(createBatchesWorker.MarathonDB.DB, map[string]interface{}{"name": "testapp"})
		j := CreateTestJob(createBatchesWorker.MarathonDB.DB, a.ID, template.Name, map[string]interface{}{
			"context": context,
			"filters": map[string]interface{}{},
			"csvPath": "tfg-push-notifications/test/jobs/obj1.csv",
			"service": "gcm",
		})
		m := map[string]interface{}{
			"jid":  2,
			"args": []string{j.ID.String()},
		}
		smsg, err := json.Marshal(m)
		Expect(err).NotTo(HaveOccurred())
		msg, err := workers.NewMsg(string(smsg))
		Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
		res, err := createBatchesWorker.RedisClient.LLen("queue:process_batch_worker").Result()
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(BeEquivalentTo(2))
		job1, err := createBatchesWorker.RedisClient.LPop("queue:process_batch_worker").Result()
		Expect(err).NotTo(HaveOccurred())
		job2, err := createBatchesWorker.RedisClient.LPop("queue:process_batch_worker").Result()
		Expect(err).NotTo(HaveOccurred())
		j1 := map[string]interface{}{}
		j2 := map[string]interface{}{}
		err = json.Unmarshal([]byte(job1), &j1)
		Expect(err).NotTo(HaveOccurred())
		err = json.Unmarshal([]byte(job2), &j2)
		Expect(err).NotTo(HaveOccurred())
		Expect(j1["queue"].(string)).To(Equal("process_batch_worker"))
		Expect(j2["queue"].(string)).To(Equal("process_batch_worker"))
		Expect(len((j1["args"].([]interface{}))[2].([]interface{})) + len((j2["args"].([]interface{}))[2].([]interface{}))).To(BeEquivalentTo(9))
		Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
	})

	It("should update job DBPageSize if no previous size", func() {
		createBatchesWorker.DBPageSize = config.GetInt("workers.createBatches.dbPageSize")
		a := CreateTestApp(createBatchesWorker.MarathonDB.DB, map[string]interface{}{"name": "testapp"})
		j := CreateTestJob(createBatchesWorker.MarathonDB.DB, a.ID, template.Name, map[string]interface{}{
			"context": context,
			"filters": map[string]interface{}{},
			"csvPath": "tfg-push-notifications/test/jobs/obj1.csv",
		})
		m := map[string]interface{}{
			"jid":  2,
			"args": []string{j.ID.String()},
		}
		smsg, err := json.Marshal(m)
		Expect(err).NotTo(HaveOccurred())
		msg, err := workers.NewMsg(string(smsg))
		Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
		err = createBatchesWorker.MarathonDB.DB.Model(j).Column("job.*", "App").Where("job.id = ?", j.ID).Select()
		Expect(err).NotTo(HaveOccurred())
		Expect(j.DBPageSize).To(Equal(config.GetInt("workers.createBatches.dbPageSize")))
	})

	It("should use job DBPageSize if specified", func() {
		a := CreateTestApp(createBatchesWorker.MarathonDB.DB, map[string]interface{}{"name": "testapp"})
		j := CreateTestJob(createBatchesWorker.MarathonDB.DB, a.ID, template.Name, map[string]interface{}{
			"context": context,
			"filters": map[string]interface{}{},
			"csvPath": "tfg-push-notifications/test/jobs/obj1.csv",
		})
		createBatchesWorker.MarathonDB.DB.Model(j).Set("db_page_size = ?", 500).Returning("*").Update()
		m := map[string]interface{}{
			"jid":  2,
			"args": []string{j.ID.String()},
		}
		smsg, err := json.Marshal(m)
		Expect(err).NotTo(HaveOccurred())
		msg, err := workers.NewMsg(string(smsg))
		Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
		err = createBatchesWorker.MarathonDB.DB.Model(j).Column("job.*", "App").Where("job.id = ?", j.ID).Select()
		Expect(err).NotTo(HaveOccurred())
		Expect(j.DBPageSize).To(Equal(500))
	})

	It("should increment job totalBatches when no previous totalBatches", func() {
		a := CreateTestApp(createBatchesWorker.MarathonDB.DB, map[string]interface{}{"name": "testapp"})
		j := CreateTestJob(createBatchesWorker.MarathonDB.DB, a.ID, template.Name, map[string]interface{}{
			"context": context,
			"filters": map[string]interface{}{},
			"csvPath": "tfg-push-notifications/test/jobs/obj1.csv",
		})
		m := map[string]interface{}{
			"jid":  2,
			"args": []string{j.ID.String()},
		}
		smsg, err := json.Marshal(m)
		Expect(err).NotTo(HaveOccurred())
		msg, err := workers.NewMsg(string(smsg))
		Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
		job := &model.Job{}
		err = createBatchesWorker.MarathonDB.DB.Model(job).Where("id = ?", j.ID).Select()
		Expect(err).NotTo(HaveOccurred())
		Expect(job.TotalBatches).To(BeEquivalentTo(2))
	})

	It("should increment job totalBatches when previous totalBatches", func() {
		a := CreateTestApp(createBatchesWorker.MarathonDB.DB, map[string]interface{}{"name": "testapp"})
		j := CreateTestJob(createBatchesWorker.MarathonDB.DB, a.ID, template.Name, map[string]interface{}{
			"context": context,
			"filters": map[string]interface{}{},
			"csvPath": "tfg-push-notifications/test/jobs/obj1.csv",
		})
		_, err := createBatchesWorker.MarathonDB.DB.Model(j).Set("total_batches = 4").Where("id = ?", j.ID).Update()
		Expect(err).NotTo(HaveOccurred())
		m := map[string]interface{}{
			"jid":  2,
			"args": []string{j.ID.String()},
		}
		smsg, err := json.Marshal(m)
		Expect(err).NotTo(HaveOccurred())
		msg, err := workers.NewMsg(string(smsg))
		Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
		job := &model.Job{}
		err = createBatchesWorker.MarathonDB.DB.Model(job).Where("id = ?", j.ID).Select()
		Expect(err).NotTo(HaveOccurred())
		Expect(job.TotalBatches).To(BeEquivalentTo(6))
	})

	It("should increment job totalUsers when no previous totalUsers", func() {
		a := CreateTestApp(createBatchesWorker.MarathonDB.DB, map[string]interface{}{"name": "testapp"})
		j := CreateTestJob(createBatchesWorker.MarathonDB.DB, a.ID, template.Name, map[string]interface{}{
			"context": context,
			"filters": map[string]interface{}{},
			"csvPath": "tfg-push-notifications/test/jobs/obj1.csv",
		})
		m := map[string]interface{}{
			"jid":  2,
			"args": []string{j.ID.String()},
		}
		smsg, err := json.Marshal(m)
		Expect(err).NotTo(HaveOccurred())
		msg, err := workers.NewMsg(string(smsg))
		Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
		job := &model.Job{}
		err = createBatchesWorker.MarathonDB.DB.Model(job).Where("id = ?", j.ID).Select()
		Expect(err).NotTo(HaveOccurred())
		Expect(job.TotalUsers).To(BeEquivalentTo(10))
	})

	It("should increment job totalUsers when previous totalUsers", func() {
		a := CreateTestApp(createBatchesWorker.MarathonDB.DB, map[string]interface{}{"name": "testapp"})
		j := CreateTestJob(createBatchesWorker.MarathonDB.DB, a.ID, template.Name, map[string]interface{}{
			"context": context,
			"filters": map[string]interface{}{},
			"csvPath": "tfg-push-notifications/test/jobs/obj3.csv",
		})
		_, err := createBatchesWorker.MarathonDB.DB.Model(j).Set("total_users = 4").Where("id = ?", j.ID).Update()
		Expect(err).NotTo(HaveOccurred())
		m := map[string]interface{}{
			"jid":  3,
			"args": []string{j.ID.String()},
		}
		smsg, err := json.Marshal(m)
		Expect(err).NotTo(HaveOccurred())
		msg, err := workers.NewMsg(string(smsg))
		Expect(func() { createBatchesWorker.Process(msg) }).ShouldNot(Panic())
		job := &model.Job{}
		err = createBatchesWorker.MarathonDB.DB.Model(job).Where("id = ?", j.ID).Select()
		Expect(err).NotTo(HaveOccurred())
		Expect(job.TotalUsers).To(BeEquivalentTo(8))
	})

	Describe("Read CSV from S3", func() {
		It("should return correct array from Unix csv data", func() {
			res := createBatchesWorker.ReadCSVFromS3("tfg-push-notifications/test/jobs/obj3.csv")
			Expect(*res).To(HaveLen(2))
		})

		It("should return correct array from DOS csv data", func() {
			res := createBatchesWorker.ReadCSVFromS3("tfg-push-notifications/test/jobs/obj4.csv")
			Expect(*res).To(HaveLen(2))
		})
	})
})
