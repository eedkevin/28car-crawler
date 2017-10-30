package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"strconv"

	"github.com/PuerkitoBio/fetchbot"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/transform"
	"golang.org/x/text/encoding/traditionalchinese"
	mgo "gopkg.in/mgo.v2"
)

var (
	seed     = "http://28car.com/sell_lst.php"
	base     = "http://28car.com/sell_dsp.php?h_vid="
	pageUrl  = "http://28car.com/sell_lst.php?h_page="

	jobQueue  *fetchbot.Queue
	pageQueue *fetchbot.Queue

	// regex
	regexVid = regexp.MustCompile(`\d+`)
	regexNextPage = regexp.MustCompile(`goPage\((?P<num>\d+?)\)`)
	regexPrice = regexp.MustCompile(`HKD\$\ ?(?P<num>[+-]?[0-9]{1,3}(?:,?[0-9])*(?:\.[0-9]{1,2})?)`)
	regexOrgPrice = regexp.MustCompile(`原價.?\$\ ?(?P<num>[+-]?[0-9]{1,3}(?:,?[0-9])*(?:\.[0-9]{1,2})?)`)
)

type Car struct {
	Vid string
	Sid string
	Type string
	Brand string
	Model string
	Seat string
	Engine string
	Shift string
	ProductionYear string
	Description string
	OrigPrice int
	CurrPrice int
	Contact string
	UploadTime string
}

func main() {
	linkFetcher := fetchbot.New(fetchbot.HandlerFunc(jobHandler))
	jobQueue = linkFetcher.Start()
	pageFetcher := fetchbot.New(fetchbot.HandlerFunc(pageHandler))
	pageQueue = pageFetcher.Start()

	jobQueue.SendStringGet(seed)

	jobQueue.Block()
	pageQueue.Block()
}

func jobHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())

	nextPageSelector := "input#btn_nxt"
	doc, _ := goquery.NewDocumentFromResponse(res)

	nxtHtml, exists := doc.Find(nextPageSelector).First().Attr("onclick")
	if !exists {
		fmt.Println("At the end of the site. Last page - " + ctx.Cmd.URL().String())
		return
	}
	pageNum := regexNextPage.FindStringSubmatch(nxtHtml)
	nextPageUrl := pageUrl + pageNum[1]
	jobQueue.SendStringGet(nextPageUrl)

	doc.Find("div#tch_box").Each(func(i int, s *goquery.Selection) {
		rows := s.Find("td[onclick^='goDsp']")

		vidHtml, exists := rows.First().Attr("onclick")
		if !exists {
			fmt.Println("Page item contains no vid. Page - " + ctx.Cmd.URL().String())
		}
		vid := regexVid.FindAllString(vidHtml, -1)
		itemUrl := base + vid[1]
		pageQueue.SendStringGet(itemUrl)
	})
}
func fakePageHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())
}

func pageHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())

	doc, _ := goquery.NewDocumentFromResponse(res)

	vid := regexVid.FindAllString(ctx.Cmd.URL().RawQuery, -1)[0]
	fmt.Println("vid:" + vid)

	sidSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(1) > td.formt"
	sid := doc.Find(sidSelector).First().Text()
	fmt.Println("sid:" + sid)

	typeSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(2) > td.formt"
	typeText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(typeSelector).First().Text()))
	fmt.Println("车类:" + typeText)

	brandSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(3) > td.formt"
	brandText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(brandSelector).First().Text()))
	fmt.Println("车场:" + brandText)

	modelSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(4) > td.formt > a"
	modelText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(modelSelector).First().Text()))
	fmt.Println("型号:" + modelText)

	seatSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(6) > td.formt"
	seatText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(seatSelector).First().Text()))
	fmt.Println("座位:" + seatText)

	engineSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(6) > td.formt"
	engineText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(engineSelector).First().Text()))
	fmt.Println("容积:" + engineText)

	shiftSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(7) > td.formt"
	shiftText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(shiftSelector).First().Text()))
	fmt.Println("传动:" + shiftText)

	productionYearSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(8) > td.formt"
	productionYearText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(productionYearSelector).First().Text()))
	fmt.Println("年份:" + productionYearText)

	descSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(9) > td.formt"
	descText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(descSelector).First().Text()))
	fmt.Println("简评:" + descText)

	priceSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(10) > td.formt > table > tbody > tr > td:nth-child(1)"
	priceStr, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(priceSelector).First().Text()))
	priceText := regexPrice.FindStringSubmatch(priceStr)[1]
	currPrice, _ := strconv.Atoi(strings.Replace(priceText, ",", "", -1))
	fmt.Printf("售价:%d\n", currPrice)

	origPriceArr := regexOrgPrice.FindStringSubmatch(priceStr)
	var origPrice int
	if len(origPriceArr) != 0 {
		origPrice, _ = strconv.Atoi(strings.Replace(origPriceArr[1], ",", "", -1))
		fmt.Printf("原价:%d\n", origPrice)
	}

	contactSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(11) > td.formt"
	contactText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(contactSelector).First().Text()))
	fmt.Println("联络人:" + contactText)

	updateTimeSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(12) > td.formt"
	updateTimeText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(updateTimeSelector).First().Text()))
	fmt.Println("更新日期:" + updateTimeText)

	car := Car{
		Vid: 			vid,
		Sid:            sid,
		Type:           typeText,
		Brand:          brandText,
		Model:          modelText,
		Seat:           seatText,
		Engine:         engineText,
		Shift:          shiftText,
		ProductionYear: productionYearText,
		Description:    descText,
		OrigPrice:      origPrice,
		CurrPrice:      currPrice,
		Contact:        contactText,
		UploadTime:     updateTimeText,
	}

	errPersist := persist(&car)
	if errPersist != nil {
		fmt.Println("error on persisting data")
	}
}

func persist(car *Car) error {
	session, err := mgo.Dial("localhost:27017")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("28car").C("cars")
	err = c.Insert(car)
	return err

	//result := Car{}
	//err = c.Find(bson.M{"sid": car.Sid}).One(&result)
	//if err != nil {
	//	fmt.Println("error on getting database entity")
	//}
	//fmt.Println("Car:", result.Model)
}