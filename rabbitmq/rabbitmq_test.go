package rabbitmq

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var amqpUri string

func init() {
	amqpUri = os.Getenv("AMQP_URI")
	if amqpUri == "" {
		log.Fatal("AMQP_URI is required to run rabbitmq tests")
	}
}

// Integration tests. Expect a AMQP_URI env params
func TestNewRabbitMQ(t *testing.T) {
	queueName := "test_new_rabbitmq"

	rmq := NewRabbitMQ(amqpUri, queueName)
	defer rmq.Close()

	// Даем время на подключение
	time.Sleep(5 * time.Second)

	assert.NotNil(t, rmq.connection, "rmq.Connection is nil")
	assert.NotNil(t, rmq.channel, "rmq.Channel is nil")
}

func TestRabbitMQ_Close(t *testing.T) {
	a := assert.New(t)
	queueName := "test_rabbitmq_close"

	rmq := NewRabbitMQ(amqpUri, queueName)

	// Даем время на подключение
	time.Sleep(5 * time.Second)

	a.NotNil(rmq.connection, "rmq.Connection is nil")
	a.NotNil(rmq.channel, "rmq.Channel is nil")

	rmq.Close()
	a.Error(rmq.connection.Close(), "rmq.Connection is not closed")
	a.Error(rmq.channel.Close(), "rmq.Channel is not closed")
}

func TestRabbitMQ_DeclareQueue(t *testing.T) {
	a := assert.New(t)
	queueName := "test_rabbitmq_declare_queue"

	rmq := NewRabbitMQ(amqpUri, queueName)
	defer rmq.Close()

	// Даем время на подключение
	time.Sleep(5 * time.Second)

	q, err := rmq.DeclareQueue(queueName)
	a.NoError(err, "Unexpected error from DeclareQueue")
	a.Equal(queueName, q.Name, "Unexpected queue name")
	a.Equal(0, q.Consumers, "Unexpected queue consumers count")
	a.Equal(0, q.Messages, "Unexpected queue messages count")
}

func TestRabbitMQ_PublishWithPriority(t *testing.T) {
	a := assert.New(t)
	queueName := "test_rabbitmq_publish_with_priority"
	consumerName := fmt.Sprintf("consumer_%s", queueName)

	rmq := NewRabbitMQ(amqpUri, queueName)
	defer rmq.Close()

	// Даем время на подключение
	time.Sleep(5 * time.Second)

	expectedMsg := map[string]string{
		"key": "value",
	}
	priority := uint8(10)
	err := rmq.publishToExchange(expectedMsg, priority)
	a.NoError(err, "Unexpected error from PublishWithPriority")

	msgs, err := rmq.channel.Consume(queueName, consumerName, true, true, false, false, nil)
	d := <-msgs
	a.Equal(priority, d.Priority, "Unexpected msg priority value")

	var actualMsg map[string]string
	json.Unmarshal(d.Body, &actualMsg)

	a.Equal(expectedMsg, actualMsg, "Send and recieved msgs are not equal")

	brokenMsg := make(chan int)
	err = rmq.publishToExchange(brokenMsg, priority)
	a.Error(err, "Expected error from PublishWithPriority doesn't exist")
}

func TestRabbitMQ_Consume(t *testing.T) {
	a := assert.New(t)
	queueName := "test_rabbitmq_consume"
	workerName := fmt.Sprintf("worker_%s", queueName)

	rmq := NewRabbitMQ(amqpUri, queueName)
	defer rmq.Close()

	// Даем время на подключение
	time.Sleep(5 * time.Second)

	msgs, err := rmq.GetConsumer(workerName)
	a.NoError(err, "Unexpected error from Consume")

	// Публикуем в очередь сообщение
	expectedMsg := map[string]string{
		"key": "value",
	}
	priority := uint8(10)
	rmq.publishToExchange(expectedMsg, priority)

	// Читаем консьюмером сообщение и проверяем, что смогли это сделать корректно
	d := <-msgs
	defer d.Ack()
	expectedBody, _ := json.Marshal(expectedMsg)
	a.Equal(expectedBody, d.Body(), "Unexpected body of recieved msg")
	a.Equal(priority, d.Priority(), "Unexpected priority of recieved msg")
}
