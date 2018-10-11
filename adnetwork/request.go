package adnetwork

import "github.com/econnelly/myrevenue"

type Request interface {
	Initialize() error
	Fetch() ([]myrevenue.Model, error)
	GetName() string
	GetReport() interface{}
}
