package example

import (
	"time"

	"github.com/rusl222/scada/analyzer/testdata/src/scada"
)

var api scada.Api

var alg struct {
	q1 *scada.Var[int]
	q2 *scada.Var[float64]
	q3 *scada.Var[time.Time]
}

const (
	validReper   = "ABC"
	unicodeReper = "1TF QСТ КУШ"
	invalidReper = "UNKNOWN"
)

func valid() {
	alg.q1.Bind("ABC", api)
	alg.q1.Bind(validReper, api)
	alg.q1.Bind(unicodeReper, api)

	alg.q2.Bind("DEF", api)
	alg.q3.Bind("ABC", api)
}

func invalid() {
	alg.q1.Bind("UNKNOWN", api) // want `unknown reper "UNKNOWN"`

	alg.q1.Bind(invalidReper, api) // want `unknown reper "UNKNOWN"`
}

func dynamic() {
	value := "ABC"

	alg.q1.Bind(value, api) // want `reper must be a string constant`

	alg.q1.Bind(getReper(), api) // want `reper must be a string constant`
}

func constantExpression() {
	const prefix = "ABC"
	const suffix = "_DEF"
	const combined = prefix + suffix

	alg.q1.Bind(combined, api) // want `unknown reper "ABC_DEF"`
}

func otherType() {
	var other scada.Other

	other.Bind("UNKNOWN", api)
}

func getReper() string {
	return "ABC"
}
