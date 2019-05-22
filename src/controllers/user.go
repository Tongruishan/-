package controllers

import (
	"github.com/astaxie/beego"
	"regexp"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"encoding/json"
	"math/rand"
	"time"
	"fmt"
	"github.com/astaxie/beego/orm"
	"pygHouse/src/models"
	"github.com/astaxie/beego/utils"

	"github.com/gomodule/redigo/redis"
	"math"
)

type UserControllers struct {
	beego.Controller
}

type Message struct {
	Message string
	RequestId string
	BizId string
	Code string
} 
//注册页面展示
func (this *UserControllers)ShowRegister(){
	this.TplName="register.html"
}

//跳转页面
func RespFunc(this *beego.Controller,resp map[string]interface{}){

	//传递给前
	this.Data["json"]=resp
	//指定传递方式，为json
	this.ServeJSON()
}

//展示注册页，发送短信
func (this*UserControllers)HandleSendMsg(){
	//1.获取电话号码
	phone:=this.GetString("phone")
	fmt.Println(phone)
	//2.返回为json格式，必须为
	resp:=make(map[string]interface{})
	defer RespFunc(&this.Controller,resp)
	//3.验证电话
	if phone==""{
		beego.Error("获取电话号码失败")
		resp["errno"]=1
		resp["errmsg"]="获取电话号码错误"

		return
	}
	
	//4.验证电话格式
	//正则匹配
	reg,_:=regexp.Compile(`^1[3-9][0-9]{9}$`)
	result:=reg.FindString(phone)
	if result==""{//==""相当于没找到，如果找到就是电话号码
		beego.Error("电话号码格式错误")
		resp["errno"]=2
		resp["errmsg"]="电话号码格式错误"
		return
	}

	//5.短信验证   SDK调用
	client, err := sdk.NewClientWithAccessKey("cn-hangzhou", "LTAIu4sh9mfgqjjr", "sTPSi0Ybj0oFyqDTjQyQNqdq9I9akE")
	if err != nil {
		beego.Error("电话号码格式错误")
		//2.给容器赋值
		resp["errno"] = 3
		resp["errmsg"] = "初始化短信错误"
		return
	}
	//生成6位数随机数
	rand.Seed(time.Now().UnixNano())
	//num:=rand.Intn(100000)
	vCode:=fmt.Sprintf("%06d",rand.Intn(1000000))

	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.Scheme = "https" // https | http
	request.Domain = "dysmsapi.aliyuncs.com"
	request.Version = "2017-05-25"
	request.ApiName = "SendSms"
	request.QueryParams["RegionId"] = "cn-hangzhou"
	request.QueryParams["PhoneNumbers"] = phone
	request.QueryParams["SignName"] = "品优购"
	request.QueryParams["TemplateCode"] = "SMS_164275022"
	request.QueryParams["TemplateParam"] = "{\"code\":"+vCode+"}"

	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		beego.Error("电话号码格式错误")
		//2.给容器赋值
		resp["errno"] = 4
		resp["errmsg"] = "短信发送失败"
		return
	}
	//6.创建结构体，接受短信调用返回结构图，用json数据解析
	var message Message
	json.Unmarshal(response.GetHttpContentBytes(),&message)
	if message.Message != "OK"{
		beego.Error("电话号码格式错误")
		//2.给容器赋值
		resp["errno"] = 6
		resp["errmsg"] = message.Message
		resp["code"]=vCode
		return
	}
	//传递数据
	resp["errno"] = 5
	resp["errmsg"] = "发送成功"
}

//实现注册业务
func(this *UserControllers)HandleRegister(){
	//获取数据
	name:=this.GetString("phone")
	pwd:=this.GetString("password")
	repwd:=this.GetString("repassword")
	phone:=this.GetString("phone")

	//验证数据
	if name=="" || pwd=="" || repwd=="" || phone=="" {
		beego.Error("注册信息错误")
		this.Data["errmsg"]="注册信息错误"
		this.TplName="registerhtml"
		return
	}
	if pwd!=repwd{
		beego.Error("密码不一致")
		this.Data["errmsg"]="密码不一致"
		this.TplName="registerhtml"
		return
	}


	//操作数据
	o:=orm.NewOrm()
	var user models.User
	user.Name=name
	user.Pwd=pwd
	user.Phone=phone
	o.Insert(&user)

	//返回数据,将用户名保存到cookie
	this.Ctx.SetCookie("name",user.Name,600)
	this.Redirect("/register-email",302)
}

//邮箱激活页面展示
func(this *UserControllers)ShowRegEmail(){
	this.TplName="register-email.html"
}

//邮箱激活业务实现
func (this *UserControllers)HandleRegEmail() {
	//获取数据
	email:=this.GetString("email")
	pwd:=this.GetString("password")
	repwd:=this.GetString("repassword")

	//校验数据
	if email=="" || pwd=="" || repwd==""{
		beego.Error("邮箱验证数据不完全")
		this.Data["error"]="邮箱验证数据不完全"
		this.TplName="register-email.html"
		return
	}
	if pwd!=repwd {
		beego.Error("邮箱验证数据,密码不一致")
		this.Data["error"]="邮箱验证，密码不一致"
		this.TplName="register-email.html"
		return
	}
	reg,_:=regexp.Compile(`^\w[\w\.-]*@[0-9a-z][0-9a-z-]*(\.[a-z]+)*\.[a-z]{2,6}$`)
	result:=reg.FindString(email)
	if result==""{
		beego.Error("邮箱格式不正确")
		this.Data["error"]="邮箱格式不正确"
		this.TplName="register-email.html"
		return
	}
	//邮箱配置设置
	config := `{"username":"russian11@163.com","password":"t12345","host":"smtp.163.com","port":25}`

	beego.Info(config)

	emailReg:=utils.NewEMail(config)

	emailReg.Subject="品优购邮箱激活"
	emailReg.From="russian11@163.com"
	emailReg.To=[]string{email}
	beego.Info([]string{email})

	name1:=this.Ctx.GetCookie("name")
	beego.Info(name1)
	emailReg.HTML=`<a href="http://192.168.52.134:8080/active?userName=`+name1+`"> 点击激活该用户</a>`

	err := emailReg.Send()
	if err != nil {
		beego.Error(err)
		this.Data["error"] = "发送激活邮件失败，请重新注册！"
		this.TplName = "register-email.html"
		return
	}

	////插入邮箱
	//o:=orm.NewOrm()
	//var user models.User
	//user.Name=name1
	//o.Read(&user,"Name")



	this.Ctx.WriteString("邮件已发送，请去用户邮箱激活")

	//返回数据

}

//激活
func(this *UserControllers)Active(){
	//获取数据
	userName:=this.GetString("userName")

	//验证数据
	if userName==""{
		beego.Error("激活获取用户名错误")
		this.Redirect("/register-email",302)
		return
	}

	//操作数据
	o:=orm.NewOrm()
	var user models.User
	user.Name=userName
	err:=o.Read(&user,"Name")//查询
	if err!=nil{
		beego.Error("激活该用户不存在")
		this.Redirect("/register-email",302)
		return
	}
	user.Active=true
	o.Update(&user,"Active")//更新


	//返回数据
	this.Redirect("/login",302)
}

//登陆页面展示
func(this *UserControllers)ShowLogin(){
	//显示账号密码
	username:=this.Ctx.GetCookie("userName")
	//password:=this.Ctx.GetCookie("psw")

	beego.Info(username)
	this.Data["userName"]=username

	if username=="" {
			this.Data["checked"]=""
	} else {
		this.Data["checked"]="checked"
	}
	//渲染页面

	this.TplName="login.html"

}

//实现登陆业务
func(this *UserControllers)HandleLogin(){
	//获取数据
	userName:=this.GetString("userName")
	pwd:=this.GetString("password")

	//验证数据
	if userName=="" || pwd==""{
		beego.Error("用户名或密码为空")
		this.TplName="login.html"
		return
	}

	//操作数据
	o:=orm.NewOrm()
	var user models.User

	reg,_:=regexp.Compile(`^\w[\w\.-]*@[0-9a-z][0-9a-z-]*(\.[a-z]+)*\.[a-z]{2,6}$`)
	result:=reg.FindString(userName)

	if result!=""{
		user.Email=userName
		err:=o.Read(&user,"userName")
		if err!=nil{
			beego.Error("用户不存在")
			this.TplName="login.html"
			return
		}
		if user.Pwd!=pwd{
			beego.Error("密码错误")
			this.TplName="login.html"
			return
		}

	}else {
		user.Name=userName
		err:=o.Read(&user,"Name")
		if err!=nil{
			beego.Error("用户不存在")
			this.TplName="login.html"
			return
		}
		if user.Pwd!=pwd{
			beego.Error("密码错误")
			this.TplName="login.html"
			return
		}
	}

	if user.Active==false{
		beego.Error("用户未激活，请到邮箱激活")
		this.TplName="register-email.html"
		return
	}


	//记住用户名个
	m1:=this.GetString("m1")
	beego.Info(m1)
	if m1=="2"{
		this.Ctx.SetCookie("userName",user.Name,600)
		//this.Ctx.SetCookie("psw",user.Pwd,600)
	}else {
		this.Ctx.SetCookie("userName",user.Name,-1)
		//this.Ctx.SetCookie("psw",user.Pwd,-1)
	}


	//返回数据,将数据存在session
	this.SetSession("indexname",user.Name)
	this.Redirect("/index",302)

}

//退出登陆
func (this *UserControllers)Logout()  {
	this.DelSession("indexname")
	this.Redirect("/login",302)
}

//展示用户中心
func (this *UserControllers)ShowUserCenterInfo()  {
	//获取对象
	o:=orm.NewOrm()
	var addr models.Address
	//获取用户名
	name:=this.GetSession("indexname")
	//查询
	qs:=o.QueryTable("Address").RelatedSel("Users").Filter("Users__Name",name.(string))
	qs.Filter("IsDefault",true).One(&addr)

	////手机号加密
	//qian := addr.Phone[:3]
	//hou := addr.Phone[7:]
	//addr.Phone = qian + "****" + hou

	this.Data["addr"]=addr

	//获取历史浏览记录
	conn,err:=redis.Dial("tcp",":6379")
	if err!=nil{
		beego.Info("redis 数据库链接失败",err)
	}
	goodsIds,err:=redis.Ints(conn.Do("lrange","history_"+name.(string),0,4))
	if err!=nil{
		beego.Info("redis 数据库读取失败",err)
	}

	//创建大容器
	var goods []map[string]interface{}

	for _,v:=range goodsIds{
		//创建行容器
		temp:=make(map[string]interface{})
		o:=orm.NewOrm()
		var goodsSku models.GoodsSKU
		goodsSku.Id=v
		o.Read(&goodsSku)
		temp["goodsSku"]=goodsSku
		goods=append(goods, temp)
	}
	//传递大容器
	this.Data["goods"]=goods

	//分页


	//
	this.Layout="layout.html"
	this.TplName="user_center_info.html"

}

//展示收获地址
func (this *UserControllers)ShowSite(){

	o:=orm.NewOrm()
	var addr models.Address
	name:=this.GetSession("indexname")
	qs:=o.QueryTable("Address").RelatedSel("Users").Filter("Users__Name",name.(string))
	qs.Filter("IsDefault",true).One(&addr)
	//beego.Info(addr)



	this.Data["addr"]=addr

	this.Layout="layout.html"
	this.TplName="user_center_site.html"
}

//实现地址添加业务
func(this *UserControllers)HandleAddSite(){
	//获取数据
	revName:=this.GetString("revName")
	revAddress:=this.GetString("address")
	postCode:=this.GetString("postCode")
	revPhone:=this.GetString("revPhone")
	//验证数据
	if revName=="" || revAddress=="" || postCode==""||revPhone==""{
		beego.Error("收货地址不完整")
		this.Data["errmsg"]="收货地址不完整"
		return
	}

	//操作数据
	o:=orm.NewOrm()
	var addr models.Address
	addr.Phone=revPhone
	addr.Addr=revAddress
	addr.PostCode=postCode
	addr.Receiver=revName
	//由sessionn获取用户名
	indexname:=this.GetSession("indexname")

	//插入用户
	var user models.User

	user.Name=indexname.(string)
	o.Read(&user,"Name")
	addr.Users=&user

	//addr.IsDefault赋值
	var filteraddr models.Address
	qs:=o.QueryTable("Address").RelatedSel("Users").Filter("Users__Name",indexname.(string))
	err:=qs.Filter("IsDefault",true).One(&filteraddr)
	//beego.Info(filteraddr)
	if err==nil{
		filteraddr.IsDefault=false
		o.Update(&filteraddr,"IsDefault")
	}
	addr.IsDefault=true

	o.Insert(&addr)


	this.Data["addr"]=addr
	//返回数据
	this.Redirect("/user/site",302)
}

//订单展示
func(this *UserControllers)ShowUserOrder(){
	//获取数据
	name:=this.GetSession("indexname")
	//获取的订单信息
	o:=orm.NewOrm()
	var orderInfos []models.OrderInfo
	qs:=o.QueryTable("OrderInfo").RelatedSel("User").Filter("User__Name",name.(string))
	qs.OrderBy("-Time")

	//分页展示
	//获取订单总数两
	ordercount,_:=qs.Count()

	//设置每页显示数量
	pageCount:=2

	//总页数
	totalPage:=int(math.Ceil(float64(ordercount)/float64(pageCount)))
	//beego.Info(totalPage)
	//获取当前页
	pageNow,err:=this.GetInt("pageNow")
	if err!=nil{
		beego.Info("pageNow获取失败")
		pageNow=1
	}
	//调用分页算法，获取页码集合
	pageSlice:=PageCala(totalPage,pageNow)
	qs=qs.Limit(pageCount,pageCount*(pageNow-1))
	qs.All(&orderInfos)

	//上一页和下一页
	prePage:=1
	if pageNow-1<=0{
		prePage=1
	}else {
		prePage=pageNow-1
	}

	nextPage:=1
	if pageNow+1>totalPage{
		nextPage=totalPage
	}else {
		nextPage=pageNow+1
	}

	this.Data["prePage"]=prePage
	this.Data["nextPage"]=nextPage

	//二级容器展示订单信心
	//创建大容器
	var orders []map[string]interface{}

	//遍历orderInfos
	for _,v:=range orderInfos{
		//
		temp:=make(map[string]interface{})

		var orderGoods []models.OrderGoods
		o.QueryTable("OrderGoods").RelatedSel("OrderInfo","GoodsSKU").Filter("OrderInfo__Id",v.Id).All(&orderGoods)

		temp["orderInfo"]=v
		temp["orderGoods"]=orderGoods

		orders=append(orders,temp)
	}

	this.Data["orders"]=orders
	//beego.Info(orders)



	this.Data["pageSlice"]=pageSlice
	//this.Data[""]
	this.Layout="layout.html"
	this.TplName="user_center_order.html"

}
