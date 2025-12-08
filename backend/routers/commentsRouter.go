package routers

import (
	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context/param"
)

func init() {

    beego.GlobalControllerRouter["backend/controllers:AuthController"] = append(beego.GlobalControllerRouter["backend/controllers:AuthController"],
        beego.ControllerComments{
            Method: "GetAll",
            Router: `/`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:AuthController"] = append(beego.GlobalControllerRouter["backend/controllers:AuthController"],
        beego.ControllerComments{
            Method: "GetOne",
            Router: `/:id`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:AuthController"] = append(beego.GlobalControllerRouter["backend/controllers:AuthController"],
        beego.ControllerComments{
            Method: "Put",
            Router: `/:id`,
            AllowHTTPMethods: []string{"put"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:AuthController"] = append(beego.GlobalControllerRouter["backend/controllers:AuthController"],
        beego.ControllerComments{
            Method: "Delete",
            Router: `/:id`,
            AllowHTTPMethods: []string{"delete"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:AuthController"] = append(beego.GlobalControllerRouter["backend/controllers:AuthController"],
        beego.ControllerComments{
            Method: "Login",
            Router: `/login`,
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:AuthController"] = append(beego.GlobalControllerRouter["backend/controllers:AuthController"],
        beego.ControllerComments{
            Method: "Registration",
            Router: `/registration`,
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:LeverageController"] = append(beego.GlobalControllerRouter["backend/controllers:LeverageController"],
        beego.ControllerComments{
            Method: "GetPositionDetail",
            Router: `/position/:id`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:LeverageController"] = append(beego.GlobalControllerRouter["backend/controllers:LeverageController"],
        beego.ControllerComments{
            Method: "ClosePosition",
            Router: `/position/:id/close`,
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:LeverageController"] = append(beego.GlobalControllerRouter["backend/controllers:LeverageController"],
        beego.ControllerComments{
            Method: "OpenPosition",
            Router: `/position/open`,
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:LeverageController"] = append(beego.GlobalControllerRouter["backend/controllers:LeverageController"],
        beego.ControllerComments{
            Method: "GetPositionHistory",
            Router: `/positions/history`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:LeverageController"] = append(beego.GlobalControllerRouter["backend/controllers:LeverageController"],
        beego.ControllerComments{
            Method: "GetOpenPositions",
            Router: `/positions/open`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:MarketController"] = append(beego.GlobalControllerRouter["backend/controllers:MarketController"],
        beego.ControllerComments{
            Method: "GetKLines",
            Router: `/klines`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:TradingController"] = append(beego.GlobalControllerRouter["backend/controllers:TradingController"],
        beego.ControllerComments{
            Method: "PlaceOrder",
            Router: `/order`,
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:TradingController"] = append(beego.GlobalControllerRouter["backend/controllers:TradingController"],
        beego.ControllerComments{
            Method: "CancelOrder",
            Router: `/order/:id/cancel`,
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:TradingController"] = append(beego.GlobalControllerRouter["backend/controllers:TradingController"],
        beego.ControllerComments{
            Method: "GetOrders",
            Router: `/orders`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:TradingController"] = append(beego.GlobalControllerRouter["backend/controllers:TradingController"],
        beego.ControllerComments{
            Method: "GetPrices",
            Router: `/prices`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:TradingController"] = append(beego.GlobalControllerRouter["backend/controllers:TradingController"],
        beego.ControllerComments{
            Method: "GetTransactions",
            Router: `/transactions`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:TradingController"] = append(beego.GlobalControllerRouter["backend/controllers:TradingController"],
        beego.ControllerComments{
            Method: "GetWallets",
            Router: `/wallets`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:UserController"] = append(beego.GlobalControllerRouter["backend/controllers:UserController"],
        beego.ControllerComments{
            Method: "Post",
            Router: `/`,
            AllowHTTPMethods: []string{"post"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:UserController"] = append(beego.GlobalControllerRouter["backend/controllers:UserController"],
        beego.ControllerComments{
            Method: "GetAll",
            Router: `/`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:UserController"] = append(beego.GlobalControllerRouter["backend/controllers:UserController"],
        beego.ControllerComments{
            Method: "GetOne",
            Router: `/:id`,
            AllowHTTPMethods: []string{"get"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:UserController"] = append(beego.GlobalControllerRouter["backend/controllers:UserController"],
        beego.ControllerComments{
            Method: "Put",
            Router: `/:id`,
            AllowHTTPMethods: []string{"put"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

    beego.GlobalControllerRouter["backend/controllers:UserController"] = append(beego.GlobalControllerRouter["backend/controllers:UserController"],
        beego.ControllerComments{
            Method: "Delete",
            Router: `/:id`,
            AllowHTTPMethods: []string{"delete"},
            MethodParams: param.Make(),
            Filters: nil,
            Params: nil})

}
