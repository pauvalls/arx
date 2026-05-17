package domain

import (
	"github.com/example/multi/infrastructure"
)

type OrderService struct {
	Repo *infrastructure.Repository
}
