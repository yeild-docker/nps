package routers

import (
	"ehang.io/nps/server/dynmicgateway"
	"ehang.io/nps/web/controllers"
	"github.com/astaxie/beego"
)

func Init() {
	web_base_url := beego.AppConfig.String("web_base_url")
	bridge_over_websocket := dynmicgateway.Bridge_over_websocket
	if len(web_base_url) > 0 {
		ns := beego.NewNamespace(web_base_url,
			beego.NSRouter("/"+bridge_over_websocket, &controllers.WebSocketController{}),
			beego.NSRouter("/", &controllers.IndexController{}, "*:Index"),
			beego.NSAutoRouter(&controllers.IndexController{}),
			beego.NSAutoRouter(&controllers.LoginController{}),
			beego.NSAutoRouter(&controllers.ClientController{}),
			beego.NSAutoRouter(&controllers.AuthController{}),
		)
		if len(bridge_over_websocket) > 0 {
			ns.Router("/"+bridge_over_websocket, &controllers.WebSocketController{})
		}
		beego.AddNamespace(ns)
	} else {
		beego.Router("/", &controllers.IndexController{}, "*:Index")
		beego.AutoRouter(&controllers.IndexController{})
		beego.AutoRouter(&controllers.LoginController{})
		beego.AutoRouter(&controllers.ClientController{})
		beego.AutoRouter(&controllers.AuthController{})
		if len(bridge_over_websocket) > 0 {
			beego.Router("/"+bridge_over_websocket, &controllers.WebSocketController{})
		}
	}
}
