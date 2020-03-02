package parser

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type report struct {
	Id              int
	CheckId         int
	ServiceId       int
	ServiceName     string
	Url             string
	UrlParsed       int
	UrlAvailable    int
	UrlRedirected   int
	UrlBlockedByRkn int
	StopPhrases     []string
	Descriptions    []string
	Contacts        []string
	RegDocs         []string
	PayButtons      []string
	PayRules        []string
	Offers          []string
	PayLimits       []string
	Comments        []string
	ServiceTurnover float32
	WordsCount      int
	ImagesCount     int
	ChildPages      []string
}

func Parser(urlsData []map[string]string, checkId int, references map[string][]string, minWords int, minImgs int) (mainPageResult []*report) {
	// получаем набор урлов на проверку
	var urlSlice []string
	for _, value := range urlsData {
		urlSlice = append(urlSlice, value["WebSite"])
	}

	// результат анализа главной страницы
	mainPageResult = startWorkers(urlSlice, references, 50)
	// парсим дочерние страницы
	for i, mainPageData := range mainPageResult {
		childPageResult := startWorkers(mainPageData.ChildPages, references, 50)
		for _, childPageData := range childPageResult {
			// проверяем данные
			if childPageData.UrlParsed == 0 {
				mainPageData.UrlParsed = 0
			}
			if childPageData.UrlAvailable == 0 {
				mainPageData.UrlAvailable = 0
			}
			if childPageData.UrlRedirected == 1 {
				mainPageData.UrlRedirected = 1
			}
			if childPageData.UrlBlockedByRkn == 1 {
				mainPageData.UrlBlockedByRkn = 1
			}
			// объединяем результаты анализа всех страниц
			mainPageData.WordsCount += childPageData.WordsCount
			mainPageData.ImagesCount += childPageData.ImagesCount
			mainPageData.Comments = append(mainPageData.Comments, childPageData.Comments...)
			mainPageData.Contacts = append(mainPageData.Contacts, childPageData.Contacts...)
			mainPageData.Descriptions = append(mainPageData.Descriptions, childPageData.Descriptions...)
			mainPageData.RegDocs = append(mainPageData.RegDocs, childPageData.RegDocs...)
			mainPageData.PayButtons = append(mainPageData.PayButtons, childPageData.PayButtons...)
			mainPageData.PayRules = append(mainPageData.PayRules, childPageData.PayRules...)
			mainPageData.Offers = append(mainPageData.Offers, childPageData.Offers...)
			mainPageData.PayLimits = append(mainPageData.PayLimits, childPageData.PayLimits...)
			mainPageData.StopPhrases = append(mainPageData.StopPhrases, childPageData.StopPhrases...)
		}
		// добавляем необходимые поля
		mainPageData.Id = i + 1
		mainPageData.CheckId = checkId
		serviceId, err := strconv.Atoi(urlsData[i]["Id"])
		if err != nil {
			serviceId = 0
		}
		mainPageData.ServiceId = serviceId
		mainPageData.ServiceName = urlsData[i]["Name"]

		// если сайт доступен, проверяем на мин. к-во слов и картинок
		if mainPageData.UrlAvailable == 1 && mainPageData.WordsCount < minWords {
			mainPageData.Comments = append(mainPageData.Comments, "мало слов: "+strconv.Itoa(mainPageData.WordsCount))
		}
		if mainPageData.UrlAvailable == 1 && mainPageData.ImagesCount < minImgs {
			mainPageData.Comments = append(mainPageData.Comments, "мало картинок: "+strconv.Itoa(mainPageData.ImagesCount))
		}

		// TODO ServiceTurnover
		if serviceId != 0 {

		}

		// убираем ненужное поле
		mainPageResult[i].ChildPages = nil
		// убираем неуникальные ключи
		mainPageData.Comments = uniquelyzer(mainPageData.Comments)
		mainPageData.Contacts = uniquelyzer(mainPageData.Contacts)
		mainPageData.Descriptions = uniquelyzer(mainPageData.Descriptions)
		mainPageData.RegDocs = uniquelyzer(mainPageData.RegDocs)
		mainPageData.PayButtons = uniquelyzer(mainPageData.PayButtons)
		mainPageData.PayRules = uniquelyzer(mainPageData.PayRules)
		mainPageData.Offers = uniquelyzer(mainPageData.Offers)
		mainPageData.PayLimits = uniquelyzer(mainPageData.PayLimits)
		mainPageData.StopPhrases = uniquelyzer(mainPageData.StopPhrases)

		// считаем процент выполнения
		fmt.Println((i + 1) * 100 / len(urlSlice))
	}

	return mainPageResult
}

func startWorkers(urlsList []string, references map[string][]string, maxWorkers int) (result []*report) {
	var numJobs = len(urlsList)
	jobs := make(chan string, numJobs)
	pageAnalyzis := make(chan *report, numJobs)

	var workersQuantity = numJobs
	if workersQuantity > maxWorkers {
		workersQuantity = maxWorkers
	}
	for w := 0; w < workersQuantity; w++ {
		go workerLogic(w+1, references, jobs, pageAnalyzis)
	}

	for i := 0; i < len(urlsList); i++ {
		jobs <- urlsList[i]
	}
	close(jobs)

	for j := 0; j < numJobs; j++ {
		//fmt.Println(j, numJobs)
		result = append(result, <-pageAnalyzis)
	}

	return result
}

func workerLogic(id int, references map[string][]string, jobs <-chan string, results chan<- *report) {
	var index = 0
	for urlData := range jobs {
		// бланк ответа
		var reportData = new(report)
		reportData.Id = index + 1
		reportData.Url = urlData
		// проводим валидацию урла
		var re = regexp.MustCompile(`^(https?:\/\/)?[-а-яё\w\d:.]*(\.)[-а-яё\w\d]{2,}.*`)
		if re.Match([]byte(urlData)) == false {
			reportData.Comments = append(reportData.Comments, urlData+" - невалидный url")
			results <- reportData
			return
		}

		// проверяем протокол
		if strings.HasPrefix(urlData, "http") == false {
			resp, err := http.Head("https://" + urlData)
			if err != nil || resp.StatusCode >= 400 {
				urlData = "http://" + urlData
			} else {
				urlData = "https://" + urlData
			}
		}

		// TODO проверяем, что url отсутствует в представлении ViewBlockedDomain
		/*
			reportData.UrlBlockedByRkn = 1
			results <- reportData
			return
		*/

		// делаем запрос
		client := &http.Client{
			Timeout: 5 * time.Second,
		}
		resp, err := client.Get(urlData)
		if err != nil {
			// обрабатываем ошибки
			if strings.Contains(err.Error(), "certificate") {
				reportData.Comments = append(reportData.Comments, "Не удалось распознать сертификат")
			} else if strings.Contains(err.Error(), "no such host") {
				reportData.Comments = append(reportData.Comments, "Неразрешенный домен")
			} else if strings.Contains(err.Error(), "request canceled") ||
				strings.Contains(err.Error(), "connection was forcibly closed") {
				reportData.Comments = append(reportData.Comments, "Соединение сброшено или таймаут")
			} else {
				fmt.Println("Ошибка воркера ", id, urlData, err)
				reportData.Comments = append(reportData.Comments, "Требуется уточнение")
			}
			results <- reportData
			return
		}

		var finalURL = urlData
		// узнаем статус
		var statusCode = responseStatusCode(urlData)
		reportData.UrlAvailable = 1
		reportData.UrlParsed = 1
		// находим редиректы
		if statusCode > 300 && statusCode < 400 {
			finalURL = resp.Request.URL.String()
			reportData.UrlRedirected = 1
			reportData.Comments = append(reportData.Comments, strconv.Itoa(statusCode)+" "+finalURL)
			// TODO проверяем, что url отсутствует в представлении ViewBlockedDomain
			/*
					reportData.UrlBlockedByRkn = 1
					results <- urlData
				    return
			*/
		} else if statusCode >= 400 {
			reportData.UrlAvailable = 0
			reportData.UrlParsed = 0
			reportData.Comments = append(reportData.Comments, "Код ошибки: "+strconv.Itoa(statusCode))
			results <- reportData
			return
		}

		// Анализируем сайт
		reportData.ChildPages, // определяем список адресов дочерних страниц
			reportData.ImagesCount, // подсчитываем картинки
			reportData.WordsCount,  // Подсчитываем слова
			reportData.StopPhrases,
			reportData.Descriptions,
			reportData.Contacts,
			reportData.RegDocs,
			reportData.PayButtons,
			reportData.PayRules,
			reportData.Offers,
			reportData.PayLimits = getSiteAnalysis(resp.Body, finalURL, references)

		// закрываем тело ответа
		resp.Body.Close()
		// возвращаем результат
		results <- reportData
	}
}

/**
Узнаем http статус ответа
*/
func responseStatusCode(urlStr string) int {
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(urlStr)
	if err != nil {
		return http.StatusInternalServerError
	}
	return resp.StatusCode
}

/**
Проводим анализ сайта
*/
func getSiteAnalysis(body io.Reader, finalURL string, references map[string][]string) (
	childPages []string,
	imgsCount int,
	wordsCount int,
	stopPhrases []string,
	descriptions []string,
	contacts []string,
	regDocs []string,
	payButtons []string,
	payRules []string,
	offers []string,
	payLimits []string) {
	mainUrlData, err := url.Parse(finalURL)
	if err != nil {
		return
	}
	document, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return
	}

	// Получаем список дочерних страниц
	childPages = func(body io.Reader, finalURL string) (result []string) {
		document.Find("a").Each(func(index int, element *goquery.Selection) {
			href, exists := element.Attr("href")
			if exists && len(href) > 1 {
				if href[0:2] == "//" {
					// разбираем урлы с //
					href = mainUrlData.Scheme + ":" + href
					result = append(result, href)
				} else if href[0:1] == "/" {
					// разбираем урлы, которые начинаются с /
					href = mainUrlData.Scheme + "://" + mainUrlData.Host + href
					result = append(result, href)
				} else {
					// разбираем оставшиеся урлы
					childUrlData, err := url.Parse(href)
					if err == nil && mainUrlData.Host == childUrlData.Host {
						result = append(result, href)
					}
				}
			}
		})
		return uniquelyzer(result)
	}(body, finalURL)

	// получаем количество картинок
	document.Find("img").Each(func(index int, element *goquery.Selection) {
		_, exists := element.Attr("src")
		if exists {
			imgsCount++
		}
	})

	// получаем количество слов
	var siteText = document.Find("body").Text()
	siteText = strings.ToLower(siteText)
	var textWords = strings.Fields(siteText)
	wordsCount = len(textWords)

	// делаем map для простоты поиска
	var siteTextMap = make(map[string]bool)
	for _, word := range textWords {
		siteTextMap[word] = true
	}

	// ищем фразы по справочникам
	stopPhrases = searchKeywords(siteTextMap, references["stopPhrases"])
	descriptions = searchKeywords(siteTextMap, references["descriptions"])
	contacts = searchKeywords(siteTextMap, references["contacts"])
	regDocs = searchKeywords(siteTextMap, references["regDocs"])
	payButtons = searchKeywords(siteTextMap, references["payButtons"])
	payRules = searchKeywords(siteTextMap, references["payRules"])
	offers = searchKeywords(siteTextMap, references["offers"])
	payLimits = searchKeywords(siteTextMap, references["payLimits"])

	return childPages, imgsCount, wordsCount, stopPhrases, descriptions,
		contacts, regDocs, payButtons, payRules, offers, payLimits
}

/**
Ищем ключевые слова в тексте
*/
func searchKeywords(textMap map[string]bool, keywords []string) (result []string) {
	for _, keyword := range keywords {
		if _, ok := textMap[keyword]; ok {
			result = append(result, keyword)
		}
	}
	return result
}

/**
Убираем дубли в строковом массиве
*/
func uniquelyzer(strSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range strSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
