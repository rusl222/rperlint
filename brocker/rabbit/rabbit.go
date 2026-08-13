package rabbit

import (
	"fmt"
	"log/slog"
	"reflect"
	"time"
)

type Rabbit struct {
	connString string
	log        *slog.Logger
}

type Point struct {
	Value any
	Valid bool
	Time  time.Time
}

func (r *Rabbit) Get(reper string, dst any) error {
	val, err := rValue(reper)

	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("dst must be a non-nil pointer")
	}
	elem := rv.Elem()

	switch elem.Kind() {
	case reflect.Float32, reflect.Float64:
		elem.SetFloat(float64(val))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if err != nil {
			elem.SetInt(-1)
		} else {
			elem.SetInt(int64(val))
		}
	case reflect.Struct:
		if elem.Type() == reflect.TypeOf(time.Time{}) || elem.Type().ConvertibleTo(reflect.TypeOf(time.Time{})) {
			timeVal := reflect.ValueOf(time.Unix(int64(val), 0))
			if elem.Type() != timeVal.Type() {
				timeVal = timeVal.Convert(elem.Type())
			}
			elem.Set(timeVal)
			return nil
		}
		fallthrough
	default:
		return fmt.Errorf("тип не реализован")
	}

	return nil
}

func (r *Rabbit) Set(reper string, value any) error {

	UpdateVal(reper, r.toFloat32(value), true)
	return nil
}

func (r *Rabbit) toFloat32(value any) float32 {
	switch v := value.(type) {
	case float32:
		return v
	case float64:
		return float32(v)
	case int:
		return float32(v)
	case time.Time:
		return float32(v.Unix())
	default:
		return 0
	}
}

func (r *Rabbit) typeParam(value any) string {
	switch value.(type) {
	case float32, float64:
		return "float"
	case int:
		return "int"
	case time.Time:
		return "time"
	default:
		return "unknown"
	}
}

const DefaultConnString = "amqp://admin:admin@127.0.0.1:5672/"

func New(connString string, logger *slog.Logger) *Rabbit {
	if connString == "" {
		connString = DefaultConnString
	}
	r := Rabbit{
		connString: connString,
		log:        logger,
	}

	return &r
}

func (r *Rabbit) Connect() error {
	nameAlg = "algo"

	//Объявление входных и выходных массивов
	declareArrays()

	//Подключаемся к RabbitMQ
	declareRabbit(r.connString)

	//Определяем брать данные из общей очереди или из отдельной// если true - общая очередь
	shareQueue = true

	//Запрашиваем и отправляем данные
	go consumeFromRabbitMq(&inputMap)
	go sendToRabbitMQ(&outputMap)
	return nil
}

func (r *Rabbit) Connected() bool {
	return connected
}
