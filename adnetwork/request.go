package adnetwork

import "../src/myrevenue"

type Request interface {
	Initialize() error
	Fetch() ([]myrevenue.Model, error)
	GetName() string
	GetReport() interface{}
}