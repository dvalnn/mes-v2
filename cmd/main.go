package main

import (
	"context"
	"time"

	mes "mes/internal"
)

func main() {
	simTime := 1 * time.Minute
	mes.Run(context.Background(), simTime)
}
