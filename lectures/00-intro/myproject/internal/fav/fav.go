package fav

import (
	"fmt"

	"github.com/google/uuid"
)

func Print() {
	fmt.Printf("my favourite uuid is %s\n", uuid.New())
}
