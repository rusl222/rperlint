package rabbit

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var connected bool ///Смотрим подключились ли к Rabbit
var inputMap safeMap
var outputMap safeMap
var shareQueue = false

type Out struct {
	MEK_Address int
	Raper       string
	Value       float32
	TypeParam   string
	OldValue    float32
	Reliability bool
	TimeOld     time.Time
	Time        time.Time
}

type OutToRabbitMQ struct {
	MEK_Address int
	Raper       string
	Value       float32
	TypeParam   string
	Reliability bool
	Time        time.Time
}

type safeMap struct {
	Mu  sync.RWMutex
	Out map[string]Out
}

var connRabbitMQPublish *amqp.Connection
var connRabbitMQConsume *amqp.Connection
var nameAlg = ""

type SafeChange struct {
	Mu      sync.Mutex
	Changed bool
}

// InitializeRabbitMQConn создает  соединение для rabbit MQ «один процесс — одно соединение»
func InitializeRabbitMQConn(forWhat string, connStr string) error {
	c := make(chan *amqp.Error)
	go func() {
		err := <-c
		fmt.Println("Переподключение к Очереди: " + err.Error())
		_ = InitializeRabbitMQConn(forWhat, connStr)
	}()

	conn, err := amqp.Dial(connStr)
	//conn, err := amqp.Dial(CONNECTRABBITPC)
	if err != nil {
		fmt.Println("Не могу подключиться к Rabbit для ", forWhat, err)
		return err
	}
	conn.NotifyClose(c)
	switch forWhat {
	case "Publish":
		connRabbitMQPublish = conn
	case "Consume":
		connRabbitMQConsume = conn
	}
	return nil

}

// sendToRabbitMQ отправка Структуры в очередь по названию (для мэк)
func sendToRabbitMQ(OutputMap *safeMap) {

	if shareQueue {
		for {
			OutputMap.Mu.Lock()
			output := OutputMap.Out
			var outToRabbit = make([]OutToRabbitMQ, 0)
			for key := range output {
				value := output[key]
				if value.TimeOld != value.Time {
					outToRabbit = append(outToRabbit, OutToRabbitMQ{value.MEK_Address, value.Raper, value.Value, value.TypeParam, value.Reliability, value.Time})
					outVal, exist := OutputMap.Out[key]
					if exist {
						outVal.TimeOld = outVal.Time
						OutputMap.Out[key] = outVal
					}
				}
			}
			OutputMap.Mu.Unlock()
			if len(outToRabbit) > 0 {
				body, err := json.Marshal(outToRabbit)
				if err != nil {
					fmt.Println("Ошибка При формировании JSON ", err)
				}
				ch, err := connRabbitMQPublish.Channel()
				if err != nil {
					fmt.Println("Ошибка открытия канала RabbitMQ ", err)
				}
				args := amqp.Table{
					"x-max-length": 100,
					"x-overflow":   "reject-publish",
				}
				q, err := ch.QueueDeclare(
					"AllAlgOut", // name
					false,       // durable
					false,       // delete when unused
					false,       // exclusive
					false,       // no-wait
					args,        // arguments
				)
				if err != nil {
					fmt.Println("Failed to declare a queue ", err)

				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

				err = ch.PublishWithContext(ctx,
					"",     // exchange
					q.Name, // routing key
					false,  // mandatory
					false,  // immediate
					amqp.Publishing{
						ContentType: "application/json",
						Body:        body,
					})
				if err != nil {
					fmt.Println("Ошибка отправки в очередь", err)
				}
				ch.Close()
				cancel()
				//fmt.Println(" [x] Отправил в очередь ", outToRabbit)
			}
			time.Sleep(100 * time.Millisecond)
		}
	} else {
		for {
			OutputMap.Mu.Lock()
			output := OutputMap.Out
			var outToRabbit = make([]OutToRabbitMQ, 0)
			for key := range output {
				value := output[key]
				if value.TimeOld != value.Time {
					outToRabbit = append(outToRabbit, OutToRabbitMQ{value.MEK_Address, value.Raper, value.Value, value.TypeParam, value.Reliability, value.Time})
					outVal, exist := OutputMap.Out[key]
					if exist {
						outVal.TimeOld = outVal.Time
						OutputMap.Out[key] = outVal
					}
				}
			}
			OutputMap.Mu.Unlock()
			if len(outToRabbit) > 0 {
				body, err := json.Marshal(outToRabbit)
				if err != nil {
					fmt.Println("Ошибка При формировании JSON ", err)
				}
				ch, err := connRabbitMQPublish.Channel()
				if err != nil {
					fmt.Println("Ошибка открытия канала RabbitMQ ", err)
				}
				args := amqp.Table{
					"x-max-length": 1,
					"x-overflow":   "reject-publish",
				}
				q, err := ch.QueueDeclare(
					nameAlg+"Out", // name
					false,         // durable
					false,         // delete when unused
					false,         // exclusive
					false,         // no-wait
					args,          // arguments
				)
				if err != nil {
					fmt.Println("Failed to declare a queue ", err)

				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

				err = ch.PublishWithContext(ctx,
					"",     // exchange
					q.Name, // routing key
					false,  // mandatory
					false,  // immediate
					amqp.Publishing{
						ContentType: "application/json",
						Body:        body,
					})
				if err != nil {
					fmt.Println("Ошибка отправки в очередь", err)
				}
				ch.Close()
				cancel()
				//fmt.Println(" [x] Отправил в очередь ", outToRabbit)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// consumeFromRabbitMq получаем сообщения rabbit по названию очереди
func consumeFromRabbitMq(Out *safeMap) {

	if shareQueue {

		//Conn := ConnRabbitMQConsume
		ch, err := connRabbitMQConsume.Channel()
		if err != nil {
			fmt.Println("Ошибка открытия канала RabbitMQ ", err)
		}

		defer ch.Close()
		args := amqp.Table{
			"x-max-length": 1,
			"x-overflow":   "drop-head",
		}
		q, err := ch.QueueDeclare(
			nameAlg, // name
			true,    // durable
			false,   // delete when unused
			false,   // exclusive
			false,   // no-wait
			args,    // arguments
		)
		if err != nil {
			fmt.Println("Consumer Ошибка декларирования очереди RabbitMQ ", nameAlg+"Out", err)
		}

		err = ch.Qos(
			1,     // prefetch count
			0,     // prefetch size
			false, // global
		)
		if err != nil {
			fmt.Println("Consumer Ошибка Qos RabbitMQ ", err)
		}
		// Привязываем очередь клиента к fanout exchange
		err = ch.QueueBind(
			q.Name,      // имя очереди
			"",          // routing key (не используется для fanout)
			"FanoutAlg", // имя exchange
			false,
			args,
		)
		if err != nil {
			fmt.Printf("Не удалось привязать очередь %s к exchange: %s", q.Name, err)
		}
		msgs, err := ch.Consume(
			q.Name, // queue
			"",     // consumer
			false,  // auto-ack
			false,  // exclusive
			false,  // no-local
			false,  // no-wait
			args,   // args
		)
		if err != nil {
			fmt.Println("Consumer Ошибка создания Consumer ", err)
		}

		var forever chan struct{}

		if err == nil {
			MessageHandler(msgs, Out)
		}
		fmt.Println(" [*] Waiting for messages.")
		<-forever
	} else {
		//Conn := ConnRabbitMQConsume
		ch, err := connRabbitMQConsume.Channel()
		if err != nil {
			fmt.Println("Ошибка открытия канала RabbitMQ ", err)
		}

		defer ch.Close()
		args := amqp.Table{
			"x-max-length": 1,
			"x-overflow":   "reject-publish",
		}
		q, err := ch.QueueDeclare(
			nameAlg, // name
			false,   // durable
			false,   // delete when unused
			false,   // exclusive
			false,   // no-wait
			args,    // arguments
		)
		if err != nil {
			fmt.Println("Consumer Ошибка декларирования очереди RabbitMQ ", nameAlg+"Out", err)
		}

		err = ch.Qos(
			1,     // prefetch count
			0,     // prefetch size
			false, // global
		)
		if err != nil {
			fmt.Println("Consumer Ошибка Qos RabbitMQ ", err)
		}

		msgs, err := ch.Consume(
			q.Name, // queue
			"",     // consumer
			false,  // auto-ack
			false,  // exclusive
			false,  // no-local
			false,  // no-wait
			args,   // args
		)
		if err != nil {
			fmt.Println("Consumer Ошибка создания Consumer ", err)
		}

		var forever chan struct{}

		if err == nil {
			MessageHandler(msgs, Out)
		}
		fmt.Println(" [*] Waiting for messages.")
		<-forever
	}
}

// MessageHandler записывает изменения пришедшие с очереди мека в общую выходную структуру
func MessageHandler(msgs <-chan amqp.Delivery, OutArr *safeMap) {
	var data []OutToRabbitMQ
	for d := range msgs {
		err := json.Unmarshal(d.Body, &data)
		if err != nil {
			fmt.Println("Ошибка разбора JSON:", err)
			continue
		}
		//  ************ ЗАПИСЬ В ОБЩУЮ СТРУКТУРУ**********
		OutArr.Mu.Lock()
		connected = true
		for _, inputVal := range data {

			outVal, exist := OutArr.Out[inputVal.Raper]
			if exist {
				//fmt.Println("Добавляю в массив: ", inputVal.Raper)
				outVal.Value = inputVal.Value
				outVal.Time = inputVal.Time
				outVal.Reliability = inputVal.Reliability
				OutArr.Out[inputVal.Raper] = outVal
			} else {
				//fmt.Println("Добавляю в массив: ", inputVal.Raper)
				OutArr.Out[inputVal.Raper] = Out{
					Value:       inputVal.Value,
					Time:        inputVal.Time,
					Raper:       inputVal.Raper,
					MEK_Address: inputVal.MEK_Address,
					TypeParam:   inputVal.TypeParam,
					Reliability: inputVal.Reliability,
				}
			}
		}
		OutArr.Mu.Unlock()
		d.Ack(false)
	}
}

func UpdateVal(Reaper string, val float32, Reliability bool) {
	outputMap.Mu.Lock()
	outVal, exist := outputMap.Out[Reaper]
	if exist {
		outVal.OldValue = outVal.Value
		outVal.Value = val
		outVal.TimeOld = outVal.Time
		outVal.Time = time.Now()
		outVal.Reliability = Reliability
		outputMap.Out[Reaper] = outVal
	} else {
		outputMap.Out[Reaper] = Out{
			Value:       val,
			Reliability: Reliability,
			Raper:       Reaper,
			Time:        time.Now(),
		}
	}
	outputMap.Mu.Unlock()
}

func declareArrays() {
	inputMap.Out = make(map[string]Out)
	outputMap.Out = make(map[string]Out)
}

func declareRabbit(connStr string) {

	// Иницилизация rabbitMQ
	for {
		err := InitializeRabbitMQConn("Publish", connStr)
		InitializeRabbitMQConn("Consume", connStr)
		if err != nil {
			fmt.Println("Пытаюсь подключиться к Rabbit")
			time.Sleep(4 * time.Second)
		} else {
			break
		}
	}
	//InitializeRabbitMQConn("Publish")
	///InitializeRabbitMQConn("Consume")
}

func BoolTfl32(val bool) float32 {
	switch val {
	case true:
		return 1.0
	default:
		return 0.0
	}
}

func rValue(Raper string) (float64, error) {
	inputMap.Mu.RLock()
	val, exist := inputMap.Out[Raper]
	inputMap.Mu.RUnlock()
	if !exist {
		//fmt.Println("Не получен репер:", Raper)
		return math.NaN(), fmt.Errorf("не получен репер %s", Raper)
	}
	return float64(val.Value), nil
}

func Dost(Raper string) bool {
	inputMap.Mu.RLock()
	value := inputMap.Out[Raper].Reliability
	inputMap.Mu.RUnlock()
	return value
}

// превращает float32 в bool
func F32ToBol(param float32) bool {
	if param == 1 {
		return true
	}
	return false
}

// превращает float32 в bool
func InversF32(param float32) float32 {
	return BoolTfl32(!F32ToBol(param))
}
