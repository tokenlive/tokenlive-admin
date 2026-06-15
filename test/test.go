package test

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/wirex"
)

const (
	baseAPI = "/api/v1"
)

var (
	app          *gin.Engine
	testInjector *wirex.Injector
)

func GetTestInjector() *wirex.Injector {
	return testInjector
}

func init() {
	// Try loading development configuration to enable Redis cache in tests
	if err := config.Load("../configs/dev", "server.toml", "middleware.toml", "logging.toml"); err != nil {
		config.MustLoad("")
	} else {
		// Override DB type and DSN to avoid modifying development MySQL
		config.C.Storage.DB.Type = "sqlite3"
		config.C.Storage.DB.DSN = "data/tokenlive-admin_test.db"
		config.C.General.DenyOperateMenu = false
	}
	config.C.Storage.DB.AutoMigrate = true

	_ = os.RemoveAll(config.C.Storage.DB.DSN)
	ctx := context.Background()
	var err error
	testInjector, _, err = wirex.BuildInjector(ctx)
	if err != nil {
		panic(err)
	}

	if err := testInjector.M.Init(ctx); err != nil {
		panic(err)
	}

	app = gin.New()
	err = testInjector.M.RegisterRouters(ctx, app)
	if err != nil {
		panic(err)
	}
}

func tester(t *testing.T) *httpexpect.Expect {
	return httpexpect.WithConfig(httpexpect.Config{
		Client: &http.Client{
			Transport: httpexpect.NewBinder(app),
			Jar:       httpexpect.NewCookieJar(),
		},
		Reporter: httpexpect.NewAssertReporter(t),
		Printers: []httpexpect.Printer{
			httpexpect.NewDebugPrinter(t, true),
		},
	})
}
