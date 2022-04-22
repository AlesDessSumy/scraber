package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/jasonwinn/geocoder"
)

// структура сотрудника филиала
type Employee struct {
	Name     string
	Position string
	Phone    string
	Mail     string
}

// структура филиала
type Filial struct {
	Name        string
	Link        string
	Geoposition [2]float64
	Employ      []Employee
}

// массив структур филиалов
var result = make([]Filial, 0)

// канал для передачи запросов
// в виде строк
// данные между запросами через
// request.Ctx не передаю, т. в Go
// это признано плохой практикой
var channal = make(chan string)

// число запросов
var zapros int = 0

// основная программа
func main() {

	t := time.Now()

	// test()
	// time.Sleep(1000 * time.Second)

	go treatment()

	visit()

	for {
		if zapros != len(result) {
			fmt.Println("no complete", zapros, len(result))
			time.Sleep(1000 * time.Millisecond)
		} else {
			break
		}
	}

	rec(result, "res.txt")

	fmt.Println(time.Since(t))
}

// обработка запроса
// получает запрос из канала
// и запускает его обработку в горутине
func treatment() {
	for {
		list := <-channal
		go parse_body(list)
		time.Sleep(200 * time.Millisecond)
	}
}

// осуществляет запрос
// и осуществляет разбор
// тела запроса
func parse_body(link string) {

	res := Filial{}

	resp, err := http.Get(link)
	if err != nil {
		log.Fatalln(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	//org := string(search_tag(body, []byte("content=")))
	org := string(parse_tag(body, []byte("<title>"), []byte("</title>")))
	org = strings.Replace(org, "About ", "", 1)

	res.Name = org
	res.Link = link

	lat, lng, err := geocoder.Geocode(org)
	if err != nil {
		panic("THERE WAS SOME ERROR!!!!!")
	}

	res.Geoposition[0] = lat
	res.Geoposition[1] = lng

	r := parse_html(body, []byte("<p><strong>"), []byte("</p>"))

	for _, i := range r {

		if len(i) > 500 {
			continue
		}

		rs := Employee{}

		rs.Name = string(parse_tag(i, []byte("><strong>"), []byte("</strong><br />")))
		rs.Phone = string(parse_tag(i, []byte("</a><br />"), []byte("</p>")))
		if strings.Contains(rs.Phone, "<") || strings.Contains(rs.Phone, ">") {
			rs.Phone = ""
		}
		rs.Mail = strings.Replace(string(search_tag(i, []byte("href="))), "mailto: ", "", 1)
		rs.Position = strings.Replace(string(parse_tag(i, []byte("</strong><br />"), []byte("<br />"))), "amp;", "", 1)

		if rs.Mail != "" && rs.Name != "" {
			res.Employ = append(res.Employ, rs)
		}
	}

	result = append(result, res)
}

// проходим по всем ссылкам
// на выбраном сайте
func visit() {

	n := 0

	c := colly.NewCollector(
		// Visit only domains: hackerspaces.org, wiki.hackerspaces.org
		colly.AllowedDomains("hackerspaces.org", "wiki.hackerspaces.org", "ymcanyc.org"),
	)

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		c.Visit(e.Request.AbsoluteURL(link))
	})

	c.OnRequest(func(r *colly.Request) {
		s := r.URL.String()
		if strings.HasSuffix(s, "-ymca/about") {
			fmt.Println(n, len(result), "Visiting", r.URL.String())
			n++
			zapros++
			channal <- s
		}

	})

	c.Visit("https://ymcanyc.org/")
}

// разбор html страницы
func parse_html(body, nc, nk []byte) (res [][]byte) {
	type nck struct {
		a [2]int
	}
	rs := make([]nck, 0)
	for i := 0; i < len(body)-len(nc); i++ {
		a := nck{}
		if compaire_byte(body[i:i+len(nc)], nc) {
			a.a[0] = i
			for j := i + len(nc); j < len(body)-len(nk); j++ {
				if compaire_byte(body[j:j+len(nk)], nk) {
					a.a[1] = j + len(nk)

					rs = append(rs, a)
					break
				}
			}
		}
	}
	for _, i := range rs {
		res = append(res, body[i.a[0]:i.a[1]])
	}
	return
}

// поиск содержимого в срезе байт
func parse_tag(body, nc, nk []byte) (res []byte) {
	for i := 0; i < len(body)-len(nc); i++ {
		a := [2]int{}
		if compaire_byte(body[i:i+len(nc)], nc) {
			a[0] = i + len(nc)
			for j := i + len(nc); j < len(body)-len(nk)+1; j++ {
				//fmt.Println(string(body[j:j+len(nk)]), string(nk))
				if compaire_byte(body[j:j+len(nk)], nk) {
					a[1] = j
				}
				if a[1] > a[0] {
					res = body[a[0]:a[1]]
					res = del_enter(res)
					return
				}
			}
		}
	}
	return
}

// поиск содержимого в срезе байт
func search_tag(body, tg []byte) []byte {
	for i := 0; i < len(body)-len(tg); i++ {
		a := [2]int{}
		if compaire_byte(body[i:i+len(tg)], tg) {
			a[0] = i + len(tg)
			flag := 0
			n1 := 0
			n2 := 0
			for j := a[0]; j < len(body); j++ {
				if body[j] == 34 {
					if flag == 0 {
						n1 = j
						flag = 1
						continue
					} else if flag == 1 {
						n2 = j
					}
					return del_enter(body[n1+1 : n2])
				}
			}
		}
	}
	return []byte{}
}

// сравнивает два среза байт
func compaire_byte(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for n, i := range a {
		if b[n] != i {
			return false
		}
	}
	return true
}

// удаляет спец символы из строки
func del_enter(a []byte) []byte {
	r := make([]byte, 0)
	for _, i := range a {
		if i < 31 {
			continue
		}
		r = append(r, i)
	}
	return r
}

// записывает структуру
// в файл в формате json
func rec(a []Filial, s string) {

	str := ""

	for _, i := range a {

		fmt.Println(i.Name, i.Geoposition)
		for _, j := range i.Employ {
			fmt.Println("\t", j)
		}
		b, err := json.Marshal(i)
		if err != nil {
			fmt.Println(err)
		}
		str = str + string(b) + "\n"
	}

	file, err := os.Create("./" + s)

	if err != nil {
		fmt.Println("Unable to create file:", err)
		os.Exit(1)
	}
	defer file.Close()

	file.WriteString(str)
}

// осуществляет запрос
// и осуществляет разбор
// тела запроса
func test() {

	s := ""

	lst := make([]string, 0)

	lst = append(lst, "https://ymcanyc.org/locations/west-side-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/northeast-bronx-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/bedford-stuyvesant-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/castle-hill-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/broadway-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/chinatown-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/coney-island-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/dodge-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/cross-island-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/flushing-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/flatbush-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/prospect-park-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/park-slope-armory-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/greenpoint-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/harlem-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/jamaica-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/long-island-city-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/mcburney-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/north-brooklyn-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/ridgewood-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/rockaway-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/south-shore-ymca/about")
	lst = append(lst, "https://ymcanyc.org/locations/vanderbilt-ymca/about")

	for _, link := range lst {
		res := Filial{}

		resp, err := http.Get(link)
		if err != nil {
			log.Fatalln(err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
		}

		//org := string(search_tag(body, []byte("content=")))
		org := string(parse_tag(body, []byte("<title>"), []byte("</title>")))
		org = strings.Replace(org, "About the ", "", 1)

		res.Name = org
		res.Link = link

		lat, lng, err := geocoder.Geocode(org)
		if err != nil {
			panic("THERE WAS SOME ERROR!!!!!")
		}

		res.Geoposition[0] = lat
		res.Geoposition[1] = lng

		fmt.Println()
		fmt.Println("-------------------")

		fmt.Println(org)
		fmt.Println(res.Geoposition)

		r := parse_html(body, []byte("><strong>"), []byte("</a>"))

		for _, i := range r {

			if len(i) > 500 {
				continue
			}

			rs := Employee{}

			rs.Name = string(parse_tag(i, []byte("><strong>"), []byte("</strong><br />")))
			rs.Phone = string(parse_tag(i, []byte("</a><br />"), []byte("</p>")))
			if strings.Contains(rs.Phone, "<") || strings.Contains(rs.Phone, ">") {
				rs.Phone = ""
			}
			rs.Mail = strings.Replace(string(search_tag(i, []byte("href="))), "mailto: ", "", 1)
			rs.Position = strings.Replace(string(parse_tag(i, []byte("</strong><br />"), []byte("<br />"))), "amp;", "", 1)

			if rs.Mail != "" && rs.Name != "" {
				res.Employ = append(res.Employ, rs)
				fmt.Println(rs.Name)
				fmt.Println(rs.Phone)
				fmt.Println(rs.Mail)
				fmt.Println(rs.Position)
				fmt.Println("*****************")
			}

			s = s + string(body) + "\n\n"
		}
	}

	file, err := os.Create("./rrr.txt")

	if err != nil {
		fmt.Println("Unable to create file:", err)
		os.Exit(1)
	}
	defer file.Close()

	file.WriteString(s)

}
