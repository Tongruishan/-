package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"pygHouse/src/models"
	"strconv"
	"github.com/gomodule/redigo/redis"
	"time"
	"strings"
	"github.com/smartwalle/alipay"
)

type OrderControllers struct {
	beego.Controller
}

//展示订单叶
func(this *OrderControllers)ShowOrder(){
	//获取数据
	goodsIds:=this.GetStrings("checkGoods")
	//beego.Info(goodsIds)

	//校验数据
	if len(goodsIds)==0{
		this.Redirect("addCart",302)
		return
	}
	//
	name:=this.GetSession("indexname")

	//获取地址数据
	o:=orm.NewOrm()
	var goodsAddr []models.Address
	o.QueryTable("Address").RelatedSel("Users").Filter("Users__Name",name.(string)).All(&goodsAddr)
	this.Data["goodsAddr"]=goodsAddr


	//获取商品，总价和总数两
	conn,_:=redis.Dial("tcp",":6379")

	//纵容器
	var goods []map[string]interface{}

	var totalPrice,totalCount int


	//遍历数据他
	for _,v:=range goodsIds{
		//行容器
		temp:=make(map[string]interface{})

		id,_:=strconv.Atoi(v)
		var goodsSku models.GoodsSKU
		goodsSku.Id=id
		o.Read(&goodsSku)

		//获取数量
		count,_:=redis.Int(conn.Do("hget","cart_"+name.(string),id))
		//总价
		littleprice := count * goodsSku.Price

		temp["goodsSku"]=goodsSku
		temp["count"]=count
		temp["littleprice"]=littleprice

		totalPrice+=littleprice
		totalCount+=1

		goods=append(goods,temp)
	}


	this.Data["goods"]=goods
	//beego.Info(goods)
	this.Data["totalPrice"]=totalPrice
	this.Data["totalCount"]=totalCount
	this.Data["truePrice"]=totalPrice+10

	this.Data["goodsIds"]=goodsIds

	this.TplName="place_order.html"



}

//提交订单
func(this *OrderControllers)HandlePushOrder(){

	//1.获取数据
	addrId,err:=this.GetInt("addrId")
	payId,err1:=this.GetInt("payId")
	goodsIds:=this.GetString("goodsIds")
	totalCount,err3:=this.GetInt("totalCount")
	totalPrice,err4:=this.GetInt("totalPrice")

	resp:=make(map[string]interface{})
	defer RespFunc(&this.Controller,resp)

	//2.校验数据
	if err!=nil || err1!=nil||err3!=nil ||err4!=nil ||goodsIds==""{
		resp["errnum"]=1
		resp["errmsg"]="获取数据不完全"
		return
	}

	//1.1 获取用户名
	name:=this.GetSession("indexname")
	if name==nil{
		resp["errnum"]=2
		resp["errmsg"]="用户未登录"
		return
	}

	//3.0 操作数据
		//3.1 数据插入OrderInfo表单
	var orderInfo models.OrderInfo

	orderInfo.PayMethod=payId
	orderInfo.TotalCount=totalCount
	orderInfo.TotalPrice=totalPrice
	orderInfo.TransitPrice=10
	orderInfo.OrderId=time.Now().Format("20060102150405")+name.(string)
	//获取orderInfo.Address
	o:=orm.NewOrm()
	var addr models.Address
	addr.Id=addrId
	o.Read(&addr)
	orderInfo.Address=&addr
	//获取orderInfo.User
	var user models.User
	user.Name=name.(string)
	o.Read(&user,"Name")
	//开启事务
	o.Begin()

	orderInfo.User=&user

	//将orderInfo插入数据库
	_,err=o.Insert(&orderInfo)
	if err!=nil {
		resp["errnum"]=3
		resp["errmsg"]="orderInfo插入失败"
		return
	}

		//3.2 将数据插入OrderGoods表单
	goodsIdSlice:=strings.Split(goodsIds[1:len(goodsIds)-1]," ")
	conn,err:=redis.Dial("tcp",":6379")
	if err!=nil {
		resp["errnum"]=4
		resp["errmsg"]="redis链接失败"
		return
	}
	defer conn.Close()


	for _,v:=range goodsIdSlice {

		var orderGoods models.OrderGoods
		orderGoods.OrderInfo=&orderInfo
		orderGoods.Count,err=redis.Int(conn.Do("hget","cart_"+name.(string),v))
		if err!=nil {
			resp["errnum"]=6
			resp["errmsg"]="redis数据获取失败"
			o.Rollback()
			return
		}
		// 获取GoodsSKU
		var goodsSku models.GoodsSKU
		goodsSku.Id,_=strconv.Atoi(v)
		o.Read(&goodsSku)
		orderGoods.GoodsSKU=&goodsSku
		orderGoods.Price=goodsSku.Price*orderGoods.Count
		oldStock:=goodsSku.Stock
		beego.Info("oldStock=",oldStock)
		//插入之前，判断库存是否充足

		if goodsSku.Stock<orderGoods.Count{
			resp["errnum"]=7
			resp["errmsg"]="库存不足"
			o.Rollback()
			return
		}
		//更新库存和销量
		//goodsSku.Stock-=orderGoods.Count
		//goodsSku.Sales+=orderGoods.Count

		//重新读取数据
		o.Read(&goodsSku)
		//高级更新
		qs:=o.QueryTable("GoodsSKU").Filter("Id",goodsSku.Id).Filter("Stock",oldStock)
		_,err=qs.Update(orm.Params{"Stock":goodsSku.Stock-orderGoods.Count,"Sales":goodsSku.Sales+orderGoods.Count})
		if err!=nil {
			resp["errnum"]=8
			resp["errmsg"]="goodsSku更新失败"
			o.Rollback()
			return
		}
		o.Read(&goodsSku)
		beego.Info("Stock=",goodsSku.Stock)


		_,err=o.Insert(&orderGoods)
		if err!=nil {
			resp["errnum"]=6
			resp["errmsg"]="orderGoods插入失败"
			o.Rollback()
			return
		}

		//订单生成成功，清楚购物车该记录
		_,err=conn.Do("hdel","cart_"+name.(string),v)
		if err!=nil {
			resp["errnum"]=7
			resp["errmsg"]="购物车清除失败"
			o.Rollback()
			return
		}
	}
	//事务提交
	o.Commit()
	resp["errnum"]=5
	resp["errmsg"]="OK"

}

//付款
func(this *OrderControllers)Pay(){
	//获取数据
	orderId,err:=this.GetInt("orderId")
	if err!=nil{
		this.Redirect("/user/userOrder",302)
		return
	}

	//操作数据
	o:=orm.NewOrm()
	var orderInfo models.OrderInfo
	orderInfo.Id=orderId
	o.Read(&orderInfo)



	//链接阿里
	publiKey:=`MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAqIY5Qf3fhtvEO1X/syU9
bEyMIdB9MYryWS7Ly4m2UCkIwGGKCN6IPknt8zGjbd/x7r2DiGP5KbywOj28I8Sq
PFmcO4RKHdQtmLXsSX671EC0XGvmO4rY8EOTLxcmWXwSNgtmyJdZX/D694TPUtBw
esHB0IGc8fmK4XH63Aej0i3ZvLTIr9UD2t7NK7akia+5j4ulsmliSxgQ7AkLz92z
CTeXp4Aai03jQk+oGQLuo3BabA8LAFHi3g96jP+BYghGf/ejWp5UHxbgMmZes6BA
L3WFr3U1dLykpVzoi1XmrHja2s2C8SFyhCY8rpwUe0ugzLpx2JxjU/oTeZIDxP/d
pwIDAQAB
`

	privateKey:=`MIIEowIBAAKCAQEAqIY5Qf3fhtvEO1X/syU9bEyMIdB9MYryWS7Ly4m2UCkIwGGK
CN6IPknt8zGjbd/x7r2DiGP5KbywOj28I8SqPFmcO4RKHdQtmLXsSX671EC0XGvm
O4rY8EOTLxcmWXwSNgtmyJdZX/D694TPUtBwesHB0IGc8fmK4XH63Aej0i3ZvLTI
r9UD2t7NK7akia+5j4ulsmliSxgQ7AkLz92zCTeXp4Aai03jQk+oGQLuo3BabA8L
AFHi3g96jP+BYghGf/ejWp5UHxbgMmZes6BAL3WFr3U1dLykpVzoi1XmrHja2s2C
8SFyhCY8rpwUe0ugzLpx2JxjU/oTeZIDxP/dpwIDAQABAoIBADeLrhmrNN69Vut7
yADjGGC2xkVq86i9tG1iuDdJF7dKHyFRxO5xcpSdR3mt9cEEXDilbFIrKXfxQmCf
8oATlUy4+H7BdrBoO/Mmm1AKHpyyCwCfa85aUHC4xS1zQuehtlrr3R/misXNptqo
grE4FKrRbDFuVy2GY5k1OUsGlZ9zhl+cwP6YN/4f9ypmUUNa3+19QDGf3LVvWV+i
6s1FqYd/9S5JYK4h8OWp/yi2lttGL8btPjgqwoUHLY2p+Cpzros5AXA1bzIjDXcJ
gpdjsOFAAV8eJKo7BwJYO3GCj24Q2mCdtKvQXIQDMt/6fcdQx8PH49UpcZOt04pi
drSN/vkCgYEA1n00ihykRuwSNRKRy9OaRePsnkyGiBJMPhIhmIDL1YTrm/rWb+P+
xAKf+inLtsZVTckMMtpjvRKYJWlyLzNGY+i0OY1kPvtrSthhltT/zdTxvPAhrDen
tT48HKLoaU9sPOcNBHwL0mXU5c3rmKUO/NUp3RNr2jCgg4PFWTn53pMCgYEAySO1
SJ7opsRNCP1JoxB1z7oc/QGBQ801mUoh62hg8IY8EhIOTusdZ86gRluEETHLaHKN
B17pXQlGNsEGpRPYSL0G5If+OowQwDd7+URd6QiBT82pCLHnWiq1EhgqKcTQ4ZT8
6JRiJTpTZBb82z39CJoGymWU9HM/HpaM7cZ5HR0CgYAA37O2GFXHADE2zCOR7APF
4x7UqkUmCsUKv7IpV+T8srTvBr+W5xvjmosiTsdIsFgqn/YPwvoDDC9Yf6x6asAP
qiQJ0/yjkQyn2mfTYHzGTubJOUU52WQyhdVi3HsL6snrGZng+cxmiAmtAgDCt3Fv
MEMiZnDbC7Wrs367VVQiRwKBgQCNYMT+q3uaJKaFKAvHSz2T6hyQFau3bZk8Cuh2
oCJrTd8RUJGwFQDRQ2JSsebNczOnUcUhJixfkbJBsEGsbJt83SjoH1Kp8mOfyCAi
4FQmBS1nW4ZxryKPtS8a7dNNEqNDvEZccFfgFPQiiSnoLNUbY7BcOTSO8iauqGGc
jAH7XQKBgBXLpQPNIgju5fCOJn/6XUNjFoGxfZZvqPmL4c0L8lokNQtgFDD+L1n+
mo7E0puG++yjLWZOImsQFkomcT4odicHK7OfJ5UHm4MoPdSbU8P2/qfGGgZyJ6ox
PMmv7MA+rBCyjF46RxtAxdCpTYGVmfZRcxAgXhts8WUDw+jI8laa
`

	client := alipay.New("2016093000629321",publiKey,privateKey,false)
	var p = alipay.TradePagePay{}
	p.NotifyURL = "http://192.168.52.137:8080/payOK"
	p.ReturnURL = "http://192.168.52.137:8080/payOK"
	p.Subject = "品优购超市"
	p.OutTradeNo = orderInfo.OrderId
	p.TotalAmount = strconv.Itoa(orderInfo.TotalPrice)
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"
	url, err := client.TradePagePay(p)
	if err != nil {
		beego.Error("支付失败")
	}
	payUrl := url.String()







	this.Redirect(payUrl,302)


	//返回数据

}
