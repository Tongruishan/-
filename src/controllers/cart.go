package controllers

import (
	"github.com/gomodule/redigo/redis"
	"github.com/astaxie/beego"
	"pygHouse/src/models"
	"github.com/astaxie/beego/orm"
)

type CartControllers struct {
	beego.Controller
}

//展示添加购物车页面
func (this *CartControllers)HandleAddCart() {

	//获取数据
	goodsId,err:=this.GetInt("goodsId")
	beego.Info(goodsId)
	num,err2:=this.GetInt("num")

	resp:=make(map[string]interface{})

	defer RespFunc(&this.Controller,resp)

	//校验数据
	if err!=nil || err2 != nil {
		beego.Error("数据不完全")
		resp["errnum"]=1
		resp["errmsg"]="数据不完全"
		return
	}

	name := this.GetSession("indexname")

	if name == nil {
		beego.Error("用户未登录")
		resp["errnum"]=2
		resp["errmsg"]="用户未登录"
		return
	}
	//操作数据
	//链接redis
	conn,err:=redis.Dial("tcp",":6379")
	if err!=nil{
		beego.Error("redis链接错误")
		resp["errnum"]=3
		resp["errmsg"]="redis链接错误"
		return
	}
	defer conn.Close()

	//查询数量
	oldNum,_:=redis.Int( conn.Do("hget","cart_"+name.(string),goodsId))
	//if err!=nil{
	//	beego.Error("redis查询错误",err)
	//	resp["errnum"]=6
	//	resp["errmsg"]="redis查询错误"
	//	return
	//}
	//操作redis,添加数量
	_,err=conn.Do("hset","cart_"+name.(string),goodsId,oldNum+num)

	if err!=nil{
		beego.Error("添加商品到购物车错误")
		resp["errnum"]=4
		resp["errmsg"]="添加商品到购物车错误"
		return
	}
	//返回数据
	resp["errnum"]=5
	resp["errmsg"]="OK"


}

//展示购物车页面
func (this *CartControllers)ShowCartName()  {
	//beego.Info("111111111")
	//链接redis
	conn,err:=redis.Dial("tcp","127.0.0.1:6379")
	if err!=nil{
		beego.Error("redis链接错误",err)
		return
	}
	//redis查询数据
	name:=this.GetSession("indexname")
	result,_:=redis.Ints(conn.Do("hgetall","cart_"+name.(string)))

	//创建大容器
	var cartR []map[string]interface{}

	totalPrice:=0
	totalcount:=0

	//便利结果
	for i:=0;i<len(result);i+=2{

		//result[i]==商品id
		//result[i+1]==shangpin shuliang
		//创建行容器
		temp:=make(map[string]interface{})
		//var litCount
		//向mysql查询数据
		o:=orm.NewOrm()
		var goodsSku models.GoodsSKU
		goodsSku.Id=result[i]
		o.Read(&goodsSku)
		//行容器赋值
		temp["goodsSku"]=goodsSku
		temp["count"]=result[i+1]

		litPrice:=result[i+1]*goodsSku.Price
		temp["litPrice"]=litPrice

		//
		totalPrice+=litPrice
		totalcount++
		//打容器赋值
		cartR=append(cartR,temp)


	}
	this.Data["totalPrice"]=totalPrice
	this.Data["totalcount"]=totalcount

	this.Data["cartR"]=cartR

	this.TplName="cart.html"


}

//处理购物车添加按钮
func(this *CartControllers)HandleAddCartCount(){
	//获取数据
	id,err:=this.GetInt("cartGoodsId")
	num,err1:=this.GetInt("count")

	resp:=make(map[string]interface{})

	defer RespFunc(&this.Controller,resp)

	//校验数据
	if err!=nil || err1 !=nil{
		resp["errnum"]=1
		resp["errmsg"]="获取数据错误"
		return
	}

	//处理数据
	name :=this.GetSession("indexname")
	if name==nil{
		resp["errnum"]=4
		resp["errmsg"]="用户未登录，请先登录"
		return
	}

	conn,err:=redis.Dial("tcp",":6379")
	if err!=nil{
		resp["errnum"]=2
		resp["errmsg"]="redis链接错误"
		return
	}
	defer conn.Close()

	_,err=conn.Do("hset","cart_"+name.(string),id,num)
	if err!=nil{
		resp["errnum"]=3
		resp["errmsg"]="redis写入错误"
		return
	}

	//返回数据
	resp["errnum"]=5
	resp["errmsg"]="OK"



}

//删除购物车条目
func (this *CartControllers)HandleDeleteCart()  {
	//获取数据
	id,err:=this.GetInt("goodsId")

	beego.Info(id)

	resp:=make(map[string]interface{})
	defer RespFunc(&this.Controller,resp)

	if err!=nil{
		resp["errnum"]=1
		resp["errmsg"]="数据获取失败"
		return
	}

	name:=this.GetSession("indexname")
	if err!=nil{
		resp["errnum"]=2
		resp["errmsg"]="用户未登录"
		return
	}

	conn,err:=redis.Dial("tcp",":6379")
	if err!=nil{
		resp["errnum"]=3
		resp["errmsg"]="redis链接错误"
		return
	}

	_,err=conn.Do("hdel","cart_"+name.(string),id)
	if err!=nil{
		resp["errnum"]=4
		resp["errmsg"]="删除数据错误"
		return
	}

	resp["errnum"]=5
	resp["errmsg"]="OK"


}
