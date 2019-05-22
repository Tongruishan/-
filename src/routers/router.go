package routers

import (
	"pygHouse/src/controllers"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
)

func init() {
	//路由过滤器
	beego.InsertFilter("/user/*",beego.BeforeExec,userFilter)

    beego.Router("/", &controllers.MainController{})
    //用户注册
    beego.Router("/register",&controllers.UserControllers{},"get:ShowRegister;post:HandleRegister")
    //发送短信
    beego.Router("/sendMsg",&controllers.UserControllers{},"post:HandleSendMsg")
    //邮箱激活
    beego.Router("/register-email",&controllers.UserControllers{},"get:ShowRegEmail;post:HandleRegEmail")
    //激活
    beego.Router("/active",&controllers.UserControllers{},"get:Active")
    //登陆页面
    beego.Router("/login",&controllers.UserControllers{},"get:ShowLogin;post:HandleLogin")
    //展示首页
    beego.Router("/index",&controllers.ProductControllers{},"get:ShowIndex")
    //退出登陆
    beego.Router("/user/logout",&controllers.UserControllers{},"get:Logout")
    //展示会员中心
    beego.Router("/user/userCenterInfo",&controllers.UserControllers{},"get:ShowUserCenterInfo")
    //展示收获地址
    beego.Router("/user/site",&controllers.UserControllers{},"get:ShowSite;post:HandleAddSite")
    //展示生鲜商品首页
    beego.Router("/index_sx",&controllers.ProductControllers{},"get:ShowIndexSx")
    //展示商品详情
    beego.Router("/goodsDetail",&controllers.ProductControllers{},"get:ShowGoodsDetail")
    //展示商品列表
    beego.Router("/goodsList",&controllers.ProductControllers{},"get:ShowList")
    //展示搜索结果
    beego.Router("/searchGoods",&controllers.ProductControllers{},"post:HandleSearch")
    //添加购物车
    beego.Router("/addCart",&controllers.CartControllers{},"post:HandleAddCart")
    //展示购物车页面
    beego.Router("/user/cartname",&controllers.CartControllers{},"get:ShowCartName")
    //购物车窗口添加数量按钮
    beego.Router("/addCartCount",&controllers.CartControllers{},"post:HandleAddCartCount")
    //删除购物车条目
    beego.Router("/deleteCart",&controllers.CartControllers{},"post:HandleDeleteCart")
    //添加到商品订单
    beego.Router("/user/settleAccunt",&controllers.OrderControllers{},"post:ShowOrder")
    //提交订单
    beego.Router("/pushOrder",&controllers.OrderControllers{},"post:HandlePushOrder")
    //订单展示
    beego.Router("/user/userOrder",&controllers.UserControllers{},"get:ShowUserOrder")
    //付款
    beego.Router("pay",&controllers.OrderControllers{},"get:Pay")

}

func userFilter(ctx *context.Context) {
	indexname:=ctx.Input.Session("indexname")

	if indexname==nil{
		ctx.Redirect(302,"/login")
		return
	}

}
