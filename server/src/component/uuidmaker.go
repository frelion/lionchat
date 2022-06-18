package component

import "github.com/google/uuid"

func Uuidmaker() string {
	u4 := uuid.New()
	return u4.String()
}
