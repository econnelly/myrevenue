package adnetwork

import (
	".."
)

type Request interface {
	Initialize() error
	Fetch() ([]myrevenue.Model, error)
	GetName() string
	GetReport() interface{}
}
