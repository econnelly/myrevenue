package adnetwork

import (
	"github.com/econnelly/myrevenue"
	"io"
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

type DirectlyParsable interface {
	ParseRevenue(reader io.Reader) ([]myrevenue.Model, error)
}
