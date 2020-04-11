package main

import (
	"compress/gzip"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	jsoniter "github.com/json-iterator/go"
	"io/ioutil"
	"jd-auto/config"
	"jd-auto/util"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	loginHome     = "https://passport.jd.com/new/login.aspx"
	qrShow        = "https://qr.m.jd.com/show"
	loginCheck    = "https://qr.m.jd.com/check"
	validationUrl = "https://passport.jd.com/uc/qrCodeTicketValidation"
	stockUrl      = "https://c0.3.cn/stocks"
	goodsItem     = "https://item.jd.com/"
	priceUrl      = "http://p.3.cn/prices/mgets"
	selectAllItem = "https://cart.jd.com/selectAllItem.action"
	cartDetail    = "https://cart.jd.com/cart.action"
	orderUrl      = "http://trade.jd.com/shopping/order/getOrderInfo.action"
	cookies       []*http.Cookie
)

//var(
//	headerMap map[string]string
//)
//
//func init() {
//	headerMap = make(map[string]string)
//	headerMap["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.132 Safari/537.36"
//	headerMap["ContentType"]
//	headerMap[""]
//	headerMap[""]
//	headerMap[""]
//	headerMap[""]
//}

var (
	p3p string
)

func GetRequest(method string, url string) *http.Request {
	request, _ := http.NewRequest(method, url, nil)
	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.142 Safari/537.36")
	request.Header.Add("ContentType", "text/html; charset=utf-8")
	request.Header.Add("Accept-Encoding", "gzip, deflate, sdch")
	request.Header.Add("Accept-Language", "zh-CN,zh;q=0.8")
	request.Header.Add("Connection", "keep-alive")
	return request
}

// 扫码登录
func LoginByQr() {
	fmt.Println("1. 打开京东手机客户端，扫码进行登录...")
	client := http.Client{}
	request := GetRequest("GET", loginHome)
	response, _ := client.Do(request)
	if response.StatusCode != http.StatusOK {
		respBody, _ := ioutil.ReadAll(response.Body)
		fmt.Println("登录首页打开失败..." + string(respBody))
		return
	}
	cookies = response.Cookies()
	fmt.Println("2. 登录首页打开成功...")

	// 获取登录二维码
	reqQrCodeUrl := qrShow + "?appid=133&size=147&t="
	request = GetRequest("GET", reqQrCodeUrl)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response, _ = client.Do(request)
	if response.StatusCode != http.StatusOK {
		fmt.Println("登录二维码获取失败...")
		return
	}
	for _, cookie := range response.Cookies() {
		cookies = append(cookies, cookie)
	}
	fmt.Println("3. 登录二维码获取成功...")

	// 保存二维码
	qrCodeImageName := "qr_code.png"
	file, err := os.Create(qrCodeImageName)
	if err != nil {
		fmt.Println("二维码图片创建失败...")
		return
	}
	bytes, err := ioutil.ReadAll(response.Body)
	file.Write(bytes)

	// 使用当前系统打开二维码
	sysType := runtime.GOOS
	if sysType == "linux" {
		// TODO
	} else if sysType == "windows" {
		// TODO
	}

	// 检查二维码扫描结果
	token := ""
	for _, cookie := range cookies {
		if cookie.Name == "wlfstk_smdl" {
			token = cookie.Value
		}
	}
	randValue := strconv.FormatInt(util.RandInt64(1000000, 9999999), 10)
	cur := strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	params := fmt.Sprintf("?callback=jQuery%s&appid=133&token=%s&_=%s", randValue, token, cur)
	request = GetRequest("GET", loginCheck+params)
	request.Header.Add("referer", loginHome)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	loginTicket := ""
	// 尝试 100次
	for i := 0; i < 100; i++ {
		response, _ = client.Do(request)
		if response.StatusCode != http.StatusOK {
			continue
		}
		if response.Header.Get("Content-Encoding") == "gzip" {
			reader, _ := gzip.NewReader(response.Body)
			body, _ := ioutil.ReadAll(reader)
			compile := regexp.MustCompile(`({(.+)})`)
			match := compile.FindAllStringSubmatch(strings.ReplaceAll(string(body), "\n", ""), -1)
			respCode := jsoniter.Get([]byte(match[0][1]), "code").ToString()
			loginTicket = jsoniter.Get([]byte(match[0][1]), "ticket").ToString()
			fmt.Println(match[0][1])
			if strings.Compare(respCode, "200") == 0 {
				for _, cookie := range response.Cookies() {
					cookies = append(cookies, cookie)
				}
				fmt.Println("登录成功...")
				break
			}
		}
		time.Sleep(1 * time.Second)
	}
	if loginTicket == "" {
		fmt.Println("二维码登录失败...")
		return
	}
	// 验证扫描登录结果
	request = GetRequest("GET", validationUrl+"?t="+loginTicket)
	request.Header.Add("referer", loginHome)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response, _ = client.Do(request)
	if response.StatusCode != http.StatusOK {
		fmt.Println("登录验证失败...")
		return
	}
	// 判断是否要手动安全验证
	// TODO
	p3p = response.Header.Get("p3p")

	//
	readAll, err := ioutil.ReadAll(response.Body)
	fmt.Println(string(readAll))

	// 登录成功
	for _, cookie := range response.Cookies() {
		cookies = append(cookies, cookie)
	}
}

// 库存查询
func StockQuery(skuId string, area string) (int, string) {
	params := fmt.Sprintf("?skuIds=%s&area=%s&type=getstocks", skuId, area)
	response, _ := http.Get(stockUrl + params)
	contentType := response.Header.Get("Content-Type")
	readAll, _ := ioutil.ReadAll(response.Body)
	if contentType == "application/json;charset=gbk" {
		toByte := util.ConvertToByte(string(readAll), "gbk", "utf-8")
		stockJSON := jsoniter.Get(toByte, skuId).ToString()
		// 33现货  34无货 40可配货
		stockState := jsoniter.Get([]byte(stockJSON), "StockState").ToInt()
		stockStateName := jsoniter.Get([]byte(stockJSON), "StockStateName").ToString()
		return stockState, stockStateName
	}
	return -1, ""
}

// 价格查询
func PriceQuery(skuId string, area string) float64 {
	response, _ := http.Get(priceUrl + "?type=1&area=" + area + "&skuIds=J_" + skuId)
	body := response.Body
	all, _ := ioutil.ReadAll(body)
	s2 := string(all)[1 : len(string(all))-2]
	return jsoniter.Get([]byte(s2), "p").ToFloat64()
}

// 商品详情查询
func GoodsDetailQuery(skuId string, area string) map[string]interface{} {
	goodsDetail := make(map[string]interface{})
	client := http.Client{}
	detailURL := goodsItem + skuId + ".html"
	request := GetRequest("GET", detailURL)
	response, _ := client.Do(request)
	if response.Header.Get("Content-Encoding") == "gzip" {
		reader, _ := gzip.NewReader(response.Body)
		document, err := goquery.NewDocumentFromReader(reader)
		if err != nil {
			fmt.Println("html 文档加载失败..." + err.Error())
			return nil
		}
		// 商品名称
		document.Find(".sku-name").Each(func(i int, selection *goquery.Selection) {
			text := selection.Text()
			toByte := util.ConvertToByte(text, "gbk", "utf8")
			goodsName := strings.ReplaceAll(strings.ReplaceAll(string(toByte), " ", ""), "\n", "")
			goodsDetail["goodsName"] = goodsName
		})
		// 购物车链接
		document.Find("#InitCartUrl").Each(func(i int, selection *goquery.Selection) {
			href, exists := selection.Attr("href")
			if exists {
				goodsDetail["cart"] = "http:" + href
			} else {
				goodsDetail["cart"] = ""
			}
		})
		// 库存
		goodsDetail["stock"], goodsDetail["stockName"] = StockQuery(skuId, area)
		goodsDetail["price"] = PriceQuery(skuId, area)
	}
	return goodsDetail
}

// 添加商品到购物车
func GoodsAddCart(skuIds string, areaId string) {
	// 先查询商品的信息
	for {
		detailQuery := GoodsDetailQuery(skuIds, areaId)
		if detailQuery["stock"].(int) != 33 {
			// 没有现货
			fmt.Printf("[当前商品]: %s\n[库存状态]:%s\n", detailQuery["goodsName"].(string), detailQuery["stockName"].(string))
			time.Sleep(1 * time.Second)
			continue
		}
		// 现货
		fmt.Printf("[当前商品]: %s\n[库存状态]:%s\n[购物车链接]：%s\n", detailQuery["goodsName"], detailQuery["stockName"], detailQuery["cart"])
		if detailQuery["cart"] == "" {
			// 如果没有购物车链接
			return
		}
		// 取消勾选所有商品
		client := http.Client{}
		requestParam := struct {
			T          string `json:"t"`
			OutSkus    string `json:"outSkus"`
			Random     string `json:"random"`
			LocationId string `json:"locationId"`
		}{
			T:          "0",
			OutSkus:    "",
			Random:     strconv.FormatFloat(rand.Float64(), 'f', -1, 64),
			LocationId: areaId,
		}
		bytes, _ := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(requestParam)
		request, _ := http.NewRequest("POST", selectAllItem, strings.NewReader(string(bytes)))
		request.Header.Add("Referer", "https://cart.jd.com/cart.action?r="+strconv.FormatFloat(rand.Float64(), 'f', -1, 64))
		for _, cookie := range cookies {
			request.AddCookie(cookie)
		}
		response, _ := client.Do(request)
		if response.StatusCode == 200 {
			fmt.Println("预先购物车清空成功")
		}
		// 将当前商品加入到购物车
		cartUrl := detailQuery["cart"]
		request = GetRequest("GET", cartUrl.(string))
		for _, cookie := range cookies {
			request.AddCookie(cookie)
		}
		response, _ = client.Do(request)
		if response.Header.Get("Content-Encoding") == "gzip" {
			reader, _ := gzip.NewReader(response.Body)
			document, _ := goquery.NewDocumentFromReader(reader)
			document.Find("div.success-top").Each(func(i int, selection *goquery.Selection) {
				text := selection.Find("h3.ftx-02").Text()
				fmt.Println(text)
			})
		}
		fmt.Println("-----------------购物车详情-------------------")
		// 查看购物车详情
		request = GetRequest("GET", cartDetail)
		for _, cookie := range cookies {
			request.AddCookie(cookie)
		}
		response, _ = client.Do(request)
		if response.Header.Get("Content-Encoding") == "gzip" {
			reader, _ := gzip.NewReader(response.Body)
			document, _ := goquery.NewDocumentFromReader(reader)
			document.Find(".item-item").Each(func(i int, selection *goquery.Selection) {
				skuId, _ := selection.Attr("skuid")
				venderId, _ := selection.Attr("venderid")
				num, _ := selection.Attr("num")
				name := selection.Find("div.p-name").Find("a").Text()
				fmt.Printf("[商品Id]: %s [商家编号]： %s [商品数量]：%s [商品名称]： %s\n", skuId, venderId, num, strings.ReplaceAll(strings.ReplaceAll(name, "\t", ""), "\n", ""))
			})
			fmt.Println("-------------------------------------------")
		}
		break
	}

}

// 预下单
func OrderCreate() {
	client := http.Client{}
	request := GetRequest("GET", orderUrl)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response, _ := client.Do(request)
	readAll, _ := ioutil.ReadAll(response.Body)
	fmt.Println(string(readAll))

}

// 真正下单

// 邮件发送通知

func main() {
	configFile := config.LoadConfigFile()
	if configFile == nil {
		return
	}
	//skuIds, _ := configFile.GetValue("config", "sku_ids")
	//mail, _ := configFile.GetValue("config", "mail")
	//areaId, _ := configFile.GetValue("config", "area_id")
	LoginByQr()
	//GoodsAddCart(skuIds, areaId)
	//fmt.Println(detailQuery)
	request := GetRequest("GET", cartDetail)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	request.Header.Add("P3P", p3p)
	client := http.Client{}
	response, _ := client.Do(request)
	if response.Header.Get("Content-Encoding") == "gzip" {
		reader, _ := gzip.NewReader(response.Body)
		all, _ := ioutil.ReadAll(reader)
		fmt.Println(string(all))
		fmt.Println("-------------------------------------------")
	}
}
