package test

import (
	"context"
	utils "mes/internal/utils"
	"time"
)

func getHttpTestContext(url string, timeout time.Duration) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.KEY_HTTP_TIMEOUT, timeout)
	ctx = context.WithValue(ctx, utils.KEY_ERP_URL, url)
	return ctx
}
