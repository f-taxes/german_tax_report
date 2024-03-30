package web

import (
	"embed"
	"net/http"

	"github.com/f-taxes/german_tax_report/conf"
	"github.com/f-taxes/german_tax_report/global"
	iu "github.com/f-taxes/german_tax_report/irisutils"
	"github.com/kataras/golog"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/view"
)

func Start(address string, webAssets embed.FS) {
	if conf.App.Bool("debug") {
		global.SetGoLogDebugFormat()
		golog.SetLevel("debug")
		golog.Info("Debug logging is enabled!")
	}

	app := iris.New()
	app.Use(iris.Compression)
	app.SetRoutesNoLog(true)

	registerFrontend(app, webAssets)

	app.Post("/report", func(ctx iris.Context) {

		ctx.JSON(iu.Resp{
			Result: true,
		})
	})

	if err := app.Listen(address); err != nil {
		golog.Fatal(err)
	}
}

func registerFrontend(app *iris.Application, webAssets embed.FS) {
	var frontendTpl *view.HTMLEngine
	useEmbedded := conf.App.Bool("embedded")

	if useEmbedded {
		golog.Debug("Using embedded web sources")
		embeddedFs := iris.PrefixDir("frontend-dist", http.FS(webAssets))
		frontendTpl = iris.HTML(embeddedFs, ".html")
		app.HandleDir("/assets", embeddedFs)
	} else {
		golog.Debug("Using external web sources")
		frontendTpl = iris.HTML("./frontend-dist", ".html")
		app.HandleDir("/assets", "frontend-dist")
	}

	golog.Debug("Automatic reload of web sources is enabled")
	frontendTpl.Reload(conf.App.Bool("debug"))
	app.RegisterView(frontendTpl)
	app.OnAnyErrorCode(index)

	app.Get("/", index)
	app.Get("/{p:path}", index)
}

func index(ctx iris.Context) {
	ctx.View("index.html")
}
