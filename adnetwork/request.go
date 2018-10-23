package adnetwork

import (
	"github.com/econnelly/myrevenue"
	"time"
)

type Request interface {
	Initialize() error
	Fetch() ([]myrevenue.Model, error)
	GetStartDate() time.Time
	GetEndDate() time.Time
	GetName() string
	GetReport() interface{}
}
