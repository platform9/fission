package main

import (
	"fmt"
	"net/http"

	sarama "github.com/Shopify/sarama"
)

// Handler posts a message to Kafka Topic
func Handler(w http.ResponseWriter, r *http.Request) {
	brokers := []string{"broker.kafka.svc.cluster.local:9092"}
	producerConfig := sarama.NewConfig()
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll
	producerConfig.Producer.Retry.Max = 10
	producerConfig.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(brokers, producerConfig)
	fmt.Println("Created a new producer ", producer)
	if err != nil {
		panic(err)
	}
	_, _, err = producer.SendMessage(&sarama.ProducerMessage{
		Topic: "testtopic",
		Value: sarama.StringEncoder("{\"name\": \"testvalue\"}"),
	})

	if err != nil {
		w.Write([]byte(fmt.Sprintf("Failed to publish message to topic %s: %v", "testtopic", err)))
	}
	w.Write([]byte("Successfully sent to testtopic"))
}
