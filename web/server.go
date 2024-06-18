package web

import (
	"embed"
	"net/http"
	"time"

	"github.com/f-taxes/german_tax_report/conf"
	"github.com/f-taxes/german_tax_report/global"
	"github.com/f-taxes/german_tax_report/reporting"
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

	app.Post("/report/generate", func(ctx iris.Context) {
		reqData := struct {
			Year int `json:"year"`
		}{}

		if !global.ReadJSON(ctx, &reqData) {
			return
		}

		gerTZ, err := time.LoadLocation("Europe/Berlin")
		if err != nil {
			golog.Error(err)
			ctx.JSON(global.Resp{
				Result: false,
			})
			return
		}

		from := time.Date(2000, time.January, 1, 0, 0, 0, 0, gerTZ).In(time.UTC)
		to := time.Date(reqData.Year, time.December, 31, 23, 59, 59, 0, gerTZ).In(time.UTC)
		generator := reporting.NewGenerator()

		generator.Start(from, to)

		ctx.JSON(global.Resp{
			Result: true,
			Data:   generator.Recs,
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

	// app.Get("/", index)
	app.Get("/{p:path}", index)
}

func index(ctx iris.Context) {
	ctx.View("index.html")
}
