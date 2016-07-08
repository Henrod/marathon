package kafka_test

import (
	"fmt"

	"git.topfreegames.com/topfreegames/marathon/kafka"
	"git.topfreegames.com/topfreegames/marathon/messages"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Producer", func() {
	It("Should send messages received in the inChan to kafka", func() {
		topic := "test-producer-1"
		topics := []string{topic}
		brokers := []string{"localhost:3536"}
		consumerGroup := "consumer-group-test-producer-1"
		message := "message%d"

		producerConfig := kafka.ProducerConfig{Brokers: brokers}
		inChan := make(chan *messages.KafkaMessage)
		defer close(inChan)

		go kafka.Producer(&producerConfig, inChan)
		message1 := fmt.Sprintf(message, 1)
		message2 := fmt.Sprintf(message, 1)
		msg1 := &messages.KafkaMessage{Message: message1, Topic: topic}
		msg2 := &messages.KafkaMessage{Message: message2, Topic: topic}
		inChan <- msg1
		inChan <- msg2

		// Consuming
		consumerConfig := kafka.ConsumerConfig{
			ConsumerGroup: consumerGroup,
			Topics:        topics,
			Brokers:       brokers,
		}
		outChan := make(chan string, 10)
		defer close(outChan)
		done := make(chan struct{}, 1)
		defer close(done)
		go kafka.Consumer(&consumerConfig, outChan, done)

		consumedMessage1 := <-outChan
		consumedMessage2 := <-outChan
		Expect(consumedMessage1).To(Equal(message1))
		Expect(consumedMessage2).To(Equal(message2))
	})

	It("Should not create a producer if no broker found", func() {
		brokers := []string{"localhost:3555"}

		producerConfig := kafka.ProducerConfig{Brokers: brokers}
		inChan := make(chan *messages.KafkaMessage)
		defer close(inChan)

		// Producer returns here and don't get blocked
		kafka.Producer(&producerConfig, inChan)
	})
})
