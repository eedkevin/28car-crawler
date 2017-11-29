package parser

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"bitbucket.org/eedkevin/28car-crawler/database"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

var (
	// regex
	regexVid      = regexp.MustCompile(`\d+`)
	regexNextPage = regexp.MustCompile(`goPage\((?P<num>\d+?)\)`)
	regexMaxPage  = regexp.MustCompile(`genPage\((?P<max>\d+?),\s?(?P<curr>\d+?)\)`)
	regexPrice    = regexp.MustCompile(`HKD\$\s?(?P<num>[+-]?[0-9]{1,3}(?:,?[0-9])*(?:\.[0-9]{1,2})?)`)
	regexOrgPrice = regexp.MustCompile(`原價.?\$\s?(?P<num>[+-]?[0-9]{1,3}(?:,?[0-9])*(?:\.[0-9]{1,2})?)`)
)

func ParseLink(res *http.Response) (string, error) {
	doc, _ := goquery.NewDocumentFromResponse(res)

	maxPageSelector := "select#h_page > script"

	blockTestSelection := doc.Find(maxPageSelector)
	if blockTestSelection.Index() < 0 {
		fmt.Println("Error on parsing page, it's probably been blocked. Page - " + res.Request.URL.String())
		return "", errors.New("Error on parsing page, it's probably been blocked. Page - " + res.Request.URL.String())
	}

	maxPageHtml, _ := doc.Find(maxPageSelector).First().Html()
	fmt.Println(maxPageSelector)
	pages := regexMaxPage.FindStringSubmatch(maxPageHtml)
	if len(pages) > 0 {
		// return the max page number
		return pages[1], nil
	} else {
		return "EOF", nil
	}
}

func ParsePage(res *http.Response) ([]string, error) {
	vidArr := make([]string, 0)

	doc, _ := goquery.NewDocumentFromResponse(res)

	itemSelector := "div#tch_box"
	itemUrlSelector := "td[onclick^='goDsp']"

	blockTestSelection := doc.Find(itemSelector)
	if blockTestSelection.Index() < 0 {
		fmt.Println("Error on parsing page, it's probably been blocked. Page - " + res.Request.URL.String())
		return vidArr, errors.New("Error on parsing page, it's probably been blocked. Page - " + res.Request.URL.String())
	}

	doc.Find(itemSelector).Each(func(i int, s *goquery.Selection) {
		vidHtml, exists := s.Find(itemUrlSelector).First().Attr("onclick")
		if !exists {
			fmt.Println("Warning, page item contains no vid. Page - " + res.Request.URL.String())
			return
		}
		vid := regexVid.FindAllString(vidHtml, -1)
		if len(vid) > 1 {
			vidArr = append(vidArr, vid[1])
		}
	})
	return vidArr, nil
}

func ParseItem(res *http.Response) (*database.Car, error) {
	doc, _ := goquery.NewDocumentFromResponse(res)

	vid := regexVid.FindAllString(res.Request.URL.RawQuery, -1)[0]
	fmt.Println("vid:" + vid)

	sidSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(1) > td.formt"

	blockTestSelection := doc.Find(sidSelector)
	if blockTestSelection.Index() < 0 {
		fmt.Println("Error on parsing page, it's probably been blocked. Page - " + res.Request.URL.String())
		return &database.Car{}, errors.New("Error on parsing page, it's probably been blocked. Page - " + res.Request.URL.String())
	}

	sid := doc.Find(sidSelector).First().Text()
	fmt.Println("sid:" + sid)

	typeSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(2) > td.formt"
	typeText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(typeSelector).First().Text()))
	fmt.Println("車類:" + typeText)

	brandSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(3) > td.formt"
	brandText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(brandSelector).First().Text()))
	fmt.Println("車廠:" + brandText)

	modelSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(4) > td.formt > a"
	modelText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(modelSelector).First().Text()))
	fmt.Println("型號:" + modelText)

	seatSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(5) > td.formt"
	seatText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(seatSelector).First().Text()))
	fmt.Println("座位:" + seatText)

	engineSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(6) > td.formt"
	engineText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(engineSelector).First().Text()))
	fmt.Println("容積:" + engineText)

	shiftSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(7) > td.formt"
	shiftText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(shiftSelector).First().Text()))
	fmt.Println("傳動:" + shiftText)

	productionYearSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(8) > td.formt"
	productionYearText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(productionYearSelector).First().Text()))
	fmt.Println("年份:" + productionYearText)

	descSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(9) > td.formt"
	descText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(descSelector).First().Text()))
	fmt.Println("簡評:" + descText)

	priceSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(10) > td.formt > table > tbody > tr > td:nth-child(1)"
	priceStr, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(priceSelector).First().Text()))
	priceText := regexPrice.FindStringSubmatch(priceStr)[1]
	currPrice, _ := strconv.Atoi(strings.Replace(priceText, ",", "", -1))
	fmt.Printf("售價:%d\n", currPrice)

	origPriceArr := regexOrgPrice.FindStringSubmatch(priceStr)
	var origPrice int
	if len(origPriceArr) != 0 {
		origPrice, _ = strconv.Atoi(strings.Replace(origPriceArr[1], ",", "", -1))
		fmt.Printf("原價:%d\n", origPrice)
	}

	contactSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(11) > td.formt"
	contactText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(contactSelector).First().Text()))
	fmt.Println("聯絡人:" + contactText)

	updateTimeSelector := "body > table:nth-child(10) > tbody > tr > td > table > tbody > tr > td > table > tbody > tr > td > table:nth-child(4) > tbody > tr > td > table > tbody > tr:nth-child(12) > td.formt"
	updateTimeText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(doc.Find(updateTimeSelector).First().Text()))
	fmt.Println("更新日期:" + updateTimeText)

	commentSelector := "tr.formt"
	comments := []database.Comment{}

	doc.Find(commentSelector).Each(func(i int, s *goquery.Selection) {
		replierSelector := "span[id^=reply_na_]"
		replierText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(s.Find(replierSelector).First().Text()))
		replyMsgSelector := "span[id^=reply_ms_]"
		replyMsgText, _, _ := transform.String(traditionalchinese.Big5.NewDecoder(), strings.TrimSpace(s.Find(replyMsgSelector).First().Text()))

		comment := database.Comment{
			Replier: replierText,
			Msg:     replyMsgText,
		}
		comments = append(comments, comment)
	})

	hasher := md5.New()
	hasher.Write([]byte(vid + updateTimeText))
	hash := hex.EncodeToString(hasher.Sum(nil))

	car := database.Car{
		Vid:            vid,
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
		Hash:           hash,
	}
	return &car, nil
}
