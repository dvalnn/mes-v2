package main

import (
	"context"
	"time"

	mes "mes/internal"
)

func main() {
	simTime := 1 * time.Second
	mes.Run(context.Background(), simTime)
}
