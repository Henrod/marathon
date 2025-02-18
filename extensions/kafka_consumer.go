/*
 * Copyright (c) 2017 TFG Co <backend@tfgco.com>
 * Author: TFG Co <backend@tfgco.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
 * the Software, and to permit persons to whom the Software is furnished to do so,
 * subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
 * FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
 * COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
 * IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package extensions

import (
	"fmt"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	raven "github.com/getsentry/raven-go"
	"github.com/spf13/viper"
	"github.com/topfreegames/marathon/interfaces"
	"github.com/uber-go/zap"
)

// KafkaConsumer for getting push requests
type KafkaConsumer struct {
	Brokers                        string
	Config                         *viper.Viper
	Consumer                       interfaces.KafkaConsumerClient
	ConsumerGroup                  string
	Logger                         zap.Logger
	messagesReceived               int64
	msgChan                        chan []byte
	OffsetResetStrategy            string
	run                            bool
	SessionTimeout                 int
	Topics                         []string
	pendingMessagesWG              *sync.WaitGroup
	HandleAllMessagesBeforeExiting bool
}

// NewKafkaConsumer for creating a new KafkaConsumer instance
func NewKafkaConsumer(
	config *viper.Viper,
	logger zap.Logger,
	clientOrNil ...interfaces.KafkaConsumerClient,
) (*KafkaConsumer, error) {
	q := &KafkaConsumer{
		Config:            config,
		Logger:            logger,
		messagesReceived:  0,
		msgChan:           make(chan []byte),
		pendingMessagesWG: nil,
	}
	var client interfaces.KafkaConsumerClient
	if len(clientOrNil) == 1 {
		client = clientOrNil[0]
	}
	err := q.configure(client)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (q *KafkaConsumer) loadConfigurationDefaults() {
	q.Config.SetDefault("feedbackListener.kafka.topics", []string{"com.games.test"})
	q.Config.SetDefault("feedbackListener.kafka.brokers", "localhost:9092")
	q.Config.SetDefault("feedbackListener.kafka.group", "test")
	q.Config.SetDefault("feedbackListener.kafka.sessionTimeout", 6000)
	q.Config.SetDefault("feedbackListener.kafka.offsetResetStrategy", "latest")
	q.Config.SetDefault("feedbackListener.kafka.handleAllMessagesBeforeExiting", true)
}

func (q *KafkaConsumer) configure(client interfaces.KafkaConsumerClient) error {
	q.loadConfigurationDefaults()
	q.OffsetResetStrategy = q.Config.GetString("feedbackListener.kafka.offsetResetStrategy")
	q.Brokers = q.Config.GetString("feedbackListener.kafka.brokers")
	q.ConsumerGroup = q.Config.GetString("feedbackListener.kafka.group")
	q.SessionTimeout = q.Config.GetInt("feedbackListener.kafka.sessionTimeout")
	q.Topics = q.Config.GetStringSlice("feedbackListener.kafka.topics")
	q.HandleAllMessagesBeforeExiting = q.Config.GetBool("feedbackListener.kafka.handleAllMessagesBeforeExiting")

	if q.HandleAllMessagesBeforeExiting {
		var wg sync.WaitGroup
		q.pendingMessagesWG = &wg
	}

	err := q.configureConsumer(client)
	if err != nil {
		return err
	}
	return nil
}

func (q *KafkaConsumer) configureConsumer(client interfaces.KafkaConsumerClient) error {
	l := q.Logger.With(
		zap.String("method", "configureConsumer"),
		zap.String("bootstrap.servers", q.Brokers),
		zap.String("group.id", q.ConsumerGroup),
		zap.Int("session.timeout.ms", q.SessionTimeout),
		zap.Bool("go.events.channel.enable", true),
		zap.Bool("go.application.rebalance.enable", true),
		zap.Bool("enable.auto.commit", true),
		zap.Object("default.topic.config", map[string]interface{}{
			"auto.offset.reset":  q.OffsetResetStrategy,
			"auto.commit.enable": true,
		}),
		zap.Object("topics", q.Topics),
	)
	l.Debug("configuring kafka queue extension")

	if client == nil {
		c, err := kafka.NewConsumer(&kafka.ConfigMap{
			"bootstrap.servers":               q.Brokers,
			"group.id":                        q.ConsumerGroup,
			"session.timeout.ms":              q.SessionTimeout,
			"go.events.channel.enable":        true,
			"go.application.rebalance.enable": true,
			"enable.auto.commit":              true,
			"default.topic.config": kafka.ConfigMap{
				"auto.offset.reset":  q.OffsetResetStrategy,
				"auto.commit.enable": true,
			},
		})
		if err != nil {
			l.Error("error configuring kafka queue", zap.Error(err))
			return err
		}
		q.Consumer = c
	} else {
		q.Consumer = client
	}
	l.Info("kafka queue configured")
	return nil
}

// PendingMessagesWaitGroup returns the waitGroup that is incremented every time a push is consumed
func (q *KafkaConsumer) PendingMessagesWaitGroup() *sync.WaitGroup {
	return q.pendingMessagesWG
}

// StopConsuming stops consuming messages from the queue
func (q *KafkaConsumer) StopConsuming() {
	q.run = false
}

// MessagesChannel returns the channel that will receive all messages got from kafka
func (q *KafkaConsumer) MessagesChannel() *chan []byte {
	return &q.msgChan
}

// ConsumeLoop consume messages from the queue and put in messages to send channel
func (q *KafkaConsumer) ConsumeLoop() error {
	q.run = true
	l := q.Logger.With(
		zap.String("method", "ConsumeLoop"),
		zap.Object("topics", q.Topics),
	)

	err := q.Consumer.SubscribeTopics(q.Topics, nil)
	if err != nil {
		l.Error("error subscribing to topics", zap.Error(err))
		return err
	}

	l.Info("successfully subscribed to topics")

	for q.run == true {
		select {
		case ev := <-q.Consumer.Events():
			switch e := ev.(type) {
			case kafka.AssignedPartitions:
				err = q.assignPartitions(e.Partitions)
				if err != nil {
					l.Error("error assigning partitions", zap.Error(err))
				}
			case kafka.RevokedPartitions:
				err = q.unassignPartitions()
				if err != nil {
					l.Error("error revoking partitions", zap.Error(err))
				}
			case *kafka.Message:
				q.receiveMessage(e.TopicPartition, e.Value)
			case kafka.PartitionEOF:
				q.handlePartitionEOF(ev)
			case kafka.OffsetsCommitted:
				q.handleOffsetsCommitted(ev)
			case kafka.Error:
				q.handleError(ev)
				q.StopConsuming()
				return e
			default:
				q.handleUnrecognized(e)
			}
		}
	}

	return nil
}

func (q *KafkaConsumer) assignPartitions(partitions []kafka.TopicPartition) error {
	l := q.Logger.With(
		zap.String("method", "assignPartitions"),
		zap.String("partitions", fmt.Sprintf("%v", partitions)),
	)

	l.Debug("Assigning partitions...")
	err := q.Consumer.Assign(partitions)
	if err != nil {
		l.Error("Failed to assign partitions.", zap.Error(err))
		return err
	}
	l.Info("Partitions assigned.")
	return nil
}

func (q *KafkaConsumer) unassignPartitions() error {
	l := q.Logger.With(
		zap.String("method", "unassignPartitions"),
	)

	l.Debug("Unassigning partitions...")
	err := q.Consumer.Unassign()
	if err != nil {
		l.Error("Failed to unassign partitions.", zap.Error(err))
		return err
	}
	l.Info("Partitions unassigned.")
	return nil
}

func (q *KafkaConsumer) receiveMessage(topicPartition kafka.TopicPartition, value []byte) {
	l := q.Logger.With(
		zap.String("method", "receiveMessage"),
	)

	l.Debug("Processing received message...")

	q.messagesReceived++
	if q.messagesReceived%1000 == 0 {
		l.Info("received from kafka", zap.Int64("numMessages", q.messagesReceived))
	}
	l.Debug("new kafka message", zap.Int("partition", int(topicPartition.Partition)), zap.String("message", string(value)))
	if q.pendingMessagesWG != nil {
		q.pendingMessagesWG.Add(1)
	}
	q.msgChan <- value

	l.Debug("Received message processed.")
}

func (q *KafkaConsumer) handlePartitionEOF(ev kafka.Event) {
	l := q.Logger.With(
		zap.String("method", "handlePartitionEOF"),
		zap.String("partition", fmt.Sprintf("%v", ev)),
	)

	l.Debug("Reached partition EOF.")
}

func (q *KafkaConsumer) handleOffsetsCommitted(ev kafka.Event) {
	l := q.Logger.With(
		zap.String("method", "handleOffsetsCommitted"),
		zap.String("partition", fmt.Sprintf("%v", ev)),
	)

	l.Debug("Offsets committed successfully.")
}

func (q *KafkaConsumer) handleError(ev kafka.Event) {
	l := q.Logger.With(
		zap.String("method", "handleError"),
	)
	err := ev.(error)
	raven.CaptureError(err, nil)
	l.Error("Error in Kafka connection.", zap.Error(err))
}

func (q *KafkaConsumer) handleUnrecognized(ev kafka.Event) {
	l := q.Logger.With(
		zap.String("method", "handleUnrecognized"),
		zap.String("event", fmt.Sprintf("%v", ev)),
	)
	l.Warn("Kafka event not recognized.")
}

//Cleanup closes kafka consumer connection
func (q *KafkaConsumer) Cleanup() error {
	if q.run {
		q.StopConsuming()
	}
	if q.Consumer != nil {
		err := q.Consumer.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
