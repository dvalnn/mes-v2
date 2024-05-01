package test

import (
	"context"
	mes "mes/internal"
	"time"
)

func getHttpTestContext(url string, timeout time.Duration) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, mes.KEY_HTTP_TIMEOUT, timeout)
	ctx = context.WithValue(ctx, mes.KEY_ERP_URL, url)
	return ctx
}
