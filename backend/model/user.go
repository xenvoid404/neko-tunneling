package model

import "time"

type User struct {
	ID         int64
	Protocol   string
	Username   string
	Password   string
	LimitIP    int
	LimitQuota int
	Status     string
	ExpiredAt  time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
