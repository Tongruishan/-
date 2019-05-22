package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"pygHouse/src/models"
	"math"
	"github.com/gomodule/redigo/redis"
)

type ProductControllers struct {
	beego.Controller
}

//首页展示，三级liandong
func(this *ProductControllers)ShowIndex(){
	//获取用户名
	indexname:=this.GetSession("indexname")

	if indexname!=nil{
		this.Data["indexname"]=indexname.(string)
	}else {
		this.Data["indexname"]=""
	}

	//sanji
	//first stage
	//get first stage class
	o:=orm.NewOrm()
	var firstClass []models.TpshopCategory
	o.QueryTable("TpshopCategory").Filter("Pid",0).All(&firstClass)
	//beego.Info(firstClass)

	//get second stage class
	var types []map[string]interface{}	//chuang jian rong qi jie shou cha xun jie guo
	for _,k:=range firstClass{	//cha xun di yi ji
		t:=make(map[string]interface{})
		var secondStage []models.TpshopCategory
		o.QueryTable("TpshopCategory").Filter("Pid",k.Id).All(&secondStage)
		t["t1"]=k
		//beego.Info(k)
		t["t2"]=secondStage
		//beego.Info(secondStage)
		types=append(types,t)

	}

	//查询第三
	//huo qu suo you da rong qi shuju
	for _,k1:=range types{
		var erji []map[string]interface{}
		//huo qu suo you er ji cai dan
		for _,k2:=range k1["t2"].([]models.TpshopCategory){
			t:=make(map[string]interface{})
			var thirdStage []models.TpshopCategory
			o.QueryTable("TpshopCategory").Filter("Pid",k2.Id).All(&thirdStage)
			t["t22"]=k2
			t["t23"]=thirdStage
			erji=append(erji,t)

		}
		k1["t3"]=erji
	}

	this.Data["types"]=types
	this.TplName="index.html"
}

//展示生鲜首页展示
func (this *ProductControllers)ShowIndexSx() {

	o:=orm.NewOrm()

	//get goodsTypes
	var goodsTypes []models.GoodsType
	o.QueryTable("GoodsType").All(&goodsTypes)
	this.Data["goodsTypes"]=goodsTypes

	//get lunbo leixingg
	var lunboGoodsBanner []models.IndexGoodsBanner
	o.QueryTable("IndexGoodsBanner").OrderBy("index").All(&lunboGoodsBanner)
	this.Data["lunboGoodsBanner"]=lunboGoodsBanner

	//get huodong leixing
	var indexPromotion []models.IndexPromotionBanner
	o.QueryTable("IndexPromotionBanner").OrderBy("index").All(&indexPromotion)
	this.Data["indexPromotion"]=indexPromotion

	//index-sx zhanshi
	var goods[]map[string]interface{}

	for _,v:=range goodsTypes{

		//beego.Info(v)

		var textGoods []models.IndexTypeGoodsBanner
		var imagGoods []models.IndexTypeGoodsBanner

		qs:=o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsType","GoodsSKU").Filter("GoodsType__Id",v.Id).OrderBy("Index")
		qs.Filter("DisplayType",0).All(&textGoods)
		qs.Filter("DisplayType",1).All(&imagGoods)

		temp:=make(map[string]interface{})
		temp["goodstype"]=v
		temp["textGoods"]=textGoods
		temp["imagGoods"]=imagGoods

		goods=append(goods,temp)

	}
	this.Data["goods"]=goods

	this.Layout="sx_layout.html"
	this.TplName="index_sx.html"

}

//展示商品详情页
func(this *ProductControllers)ShowGoodsDetail(){

	//获取数据
	id,err:=this.GetInt("Id")

	if err!=nil{
		beego.Error("连接错误")
		this.Redirect("/index_sx",302)
		return
	}
	//详情
	o:=orm.NewOrm()
	var goodsSku models.GoodsSKU
	//goodsSku.Id=id
	//o.Read(&goodsSku)
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType","Goods").Filter("Id",id).One(&goodsSku)
	this.Data["goodsSku"]=goodsSku

	//新品推荐
	var newGoods []models.GoodsSKU
	qs:=o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Name",goodsSku.GoodsType.Name)
	qs.OrderBy("-Time").Limit(2,0).All(&newGoods)

	this.Data["newGoods"]=newGoods

	//历史浏览数据添加
		//获取用户名
	name:=this.GetSession("indexname")
	if name!=nil{
		conn,err:=redis.Dial("tcp",":6379")
		defer conn.Close()
		if err==nil{
			//先删除，去重
			conn.Do("lrem","history_"+name.(string),0,id)
			//再添加到redis
			conn.Do("lpush","history_"+name.(string),id)
		}



	}

	this.Layout="sx_layout.html"
	this.TplName="detail.html"


}

//分页算法，不依赖于beego框架
func PageCala(pageCount int,pageNow int)[]int{

	var pages []int
	if pageCount<5{
		for i:=1;i<=pageCount;i++{
			pages=append(pages,i)
		}
	} else if pageNow<=3{
		for i:=1;i<=5;i++{
			pages=append(pages,i)
		}
	} else if pageNow>=pageCount-2{
		for i:=pageCount-4;i<=pageCount;i++{
			pages=append(pages,i)
		}
	}	else {
		for i:=pageNow-2;i<=pageNow+2;i++{
			pages=append(pages,i)
		}
	}
	return pages
}

//分类列表展示
func(this *ProductControllers)ShowList(){
	//获取数据
	id,err:=this.GetInt("Id")

	//校验数据
	if err!=nil{
		beego.Error("类型错误")
		return
	}


	//操作数据
	o:=orm.NewOrm()
	var goodsType []models.GoodsSKU


	qs:=o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",id)
	//fen ye xian shi
	//xin xi de zong shu liang
	count,_:=qs.Count()
	//mei ye rong liang
	pageSize:=1
	//获取总页数
	pageCount:=int(math.Ceil(float64(count)/float64(pageSize)))
	//获取当前所在
	pageNow,_:=this.GetInt("pageNow")
	pages:=PageCala(pageCount,pageNow)
	this.Data["pages"]=pages
	//prePages nextPages
	var prePage,nextPage int

	//prePages
	if pageNow-1<=0{
		prePage=1
	}else {
		prePage=pageNow-1
	}
	//nextPages
	if pageNow+1>pageCount{
		nextPage=pageCount
	}else {
		nextPage=pageNow+1
	}

	this.Data["prePage"]=prePage
	this.Data["nextPage"]=nextPage
	//kong zhi mei ye xian shi shu liang
	qs=qs.Limit(pageSize,pageSize*(pageNow-1))

	//排序
	sort:=this.GetString("sort")
	if sort==""{
		qs.All(&goodsType)
	}else if sort == "price"{
		qs.OrderBy("Price").All(&goodsType)
	}else {
		qs.OrderBy("Sales").All(&goodsType)
	}

	//yongyu xuanze kuang xiann shi
	this.Data["sort"]=sort
	this.Data["goodsType"]=goodsType

	//
	this.Data["id"]=id
	this.Layout="sx_layout.html"
	this.TplName="list.html"
}

//商品搜索展示
func (this *ProductControllers)HandleSearch()  {

	//huo qu
	searchWords:=this.GetString("search")

	//jiaoyann
	if searchWords==""{
		beego.Error("bu neng wei kong ")
		this.Redirect("/index_sx",302)
		return
	}

	//caozuo
	o:=orm.NewOrm()
	var goodsSku []models.GoodsSKU
	o.QueryTable("GoodsSKU").Filter("Name__icontains",searchWords).All(&goodsSku)

	//fanhui
	this.Data["goodsSku"]=goodsSku
	this.TplName="search.html"
}