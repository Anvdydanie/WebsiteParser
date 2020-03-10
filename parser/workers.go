package parser

import (
	"RobotChecker/logger"
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
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

var nodePort = os.Getenv("NODE_PORT")
var nodeCheckStatus = "http://localhost:" + nodePort + "/api/check-status/"
var nodeCheckStatusPages = "http://localhost:" + nodePort + "/api/check-status-pages/"

const fileWithBlockedDomains = "listOfBlockedDomains.txt"
const childPagesQuantity = 50

var iteratee = 0
var samePercent = 0

func Parser(
	urlsData []map[string]string,
	checkId int,
	references map[string][]string,
	minWords int, minImgs int,
	duplicUrls map[string]string,
	serviceTurnover map[string]float32,
) (mainPageResult []*report) {
	iteratee = 0
	samePercent = 0
	// получаем набор урлов на проверку
	var urlSlice []string
	for _, value := range urlsData {
		urlSlice = append(urlSlice, value["WebSite"])
	}

	// получаем список заблокированных сайтов
	var blockedUrlsMap = getBlockedUrlsMap()

	// оповещение ноды о проверке страницы
	var nodeUrlPages = nodeCheckStatusPages + strconv.Itoa(checkId) + "?message=" + url.QueryEscape("Идет проверка главных страниц сайтов")
	_, _ = http.Get(nodeUrlPages)

	// логируем статус
	logger.Logger("Начинается проверка " + strconv.Itoa(checkId) + " на список сайтов: " + strconv.Itoa(len(urlSlice)))

	// результат анализа главной страницы
	mainPageResult = startWorkers(urlSlice, references, blockedUrlsMap, true, checkId, 50)
	// парсим дочерние страницы
	for i, mainPageData := range mainPageResult {
		var needToCache = false
		// останавливаем воркеры
		if RedisGetBool("stop_"+strconv.Itoa(checkId)) == true {
			break
		}

		// проверяем является ли сайт дубликатом
		_, exists := duplicUrls[mainPageData.Url]
		if exists {
			siteData := RedisGet(mainPageData.Url)
			// Если в кэше есть данные
			if len(siteData.Url) > 0 {
				*mainPageData = *siteData
			} else {
				needToCache = true
			}
		}

		// оповещение ноды о проверке страницы
		var nodeUrlPages = nodeCheckStatusPages + strconv.Itoa(checkId) + "?message=" + url.QueryEscape("Идет проверка дочерних страниц сайта "+mainPageData.Url)
		_, _ = http.Get(nodeUrlPages)

		if exists == false || exists && needToCache {
			// сокращаем количество дочерних страниц до 50
			if len(mainPageData.ChildPages) > childPagesQuantity {
				mainPageData.ChildPages = mainPageData.ChildPages[0:childPagesQuantity]
			}
			childPageResult := startWorkers(mainPageData.ChildPages, references, blockedUrlsMap, false, checkId, 50)
			for _, childPageData := range childPageResult {
				// останавливаем воркеры
				if RedisGetBool("stop_"+strconv.Itoa(checkId)) == true {
					break
				}
				// проверяем данные
				//if childPageData.UrlParsed == 0 {
				//	mainPageData.UrlParsed = 0
				//}
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
			// записываем результаты в кэш
			if needToCache {
				RedisSet(mainPageData.Url, mainPageData)
			}
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

		// Считаем оборот за неделю
		if serviceId != 0 {
			if value, exist := serviceTurnover[strconv.Itoa(serviceId)]; exist {
				mainPageData.ServiceTurnover = value
			}
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
		iteratee += 1
		// оповещаем node о ходе проверки
		var percent = iteratee * 50 / len(urlSlice)
		if percent%2 == 0 && percent != samePercent {
			samePercent = percent
			var nodeUrl = nodeCheckStatus + strconv.Itoa(checkId) + "?percent=" + strconv.Itoa(percent)
			_, _ = http.Get(nodeUrl)
			// логируем проверку
			logger.Logger("Ход проверки " + strconv.Itoa(checkId) + " составляет " + strconv.Itoa(percent) + "%")
		}

		//fmt.Println("дочерние", iteratee*50/len(urlSlice))
	}

	// если проверка была остановлена
	if RedisGetBool("stop_"+strconv.Itoa(checkId)) == true {
		// логируем остановку
		logger.Logger("Проверка " + strconv.Itoa(checkId) + " остановлена пользователем")
		return nil
	}

	return mainPageResult
}

func startWorkers(urlsList []string, references map[string][]string, blockedUrls map[string]bool, mainPage bool, checkId int, maxWorkers int) (result []*report) {
	var numJobs = len(urlsList)
	jobs := make(chan string, numJobs)
	pageAnalyzis := make(chan *report, numJobs)

	var workersQuantity = numJobs
	if workersQuantity > maxWorkers {
		workersQuantity = maxWorkers
	}
	for w := 0; w < workersQuantity; w++ {
		go workerLogic(w+1, references, blockedUrls, jobs, pageAnalyzis)
	}

	for i := 0; i < len(urlsList); i++ {
		jobs <- urlsList[i]
	}
	close(jobs)

	for j := 0; j < numJobs; j++ {
		if mainPage == true {
			iteratee += 1
			var percent = iteratee * 50 / numJobs
			// оповещаем node о ходе проверки
			if percent%2 == 0 && percent != samePercent {
				samePercent = percent
				// оповещение ноды о проценте выполнения
				var nodeUrl = nodeCheckStatus + strconv.Itoa(checkId) + "?percent=" + strconv.Itoa(iteratee*50/numJobs)
				_, _ = http.Get(nodeUrl)
				// логируем проверку
				logger.Logger("Ход проверки " + strconv.Itoa(checkId) + " составляет " + strconv.Itoa(percent) + "%")
				// оповещение ноды о проверке страниц
				var nodeUrlPages = nodeCheckStatusPages + strconv.Itoa(checkId) + "?message=" + url.QueryEscape("Идет проверка главных страниц сайтов")
				_, _ = http.Get(nodeUrlPages)
			}

			//fmt.Println("главная", iteratee*50/numJobs)
		}
		// останавливаем воркеры
		if RedisGetBool("stop_"+strconv.Itoa(checkId)) == true {
			break
		}
		result = append(result, <-pageAnalyzis)
	}

	return result
}

func workerLogic(id int, references map[string][]string, blockedUrls map[string]bool, jobs <-chan string, results chan<- *report) {
	var index = 0
	for urlData := range jobs {
		// бланк ответа
		var reportData = new(report)
		reportData.Id = index + 1
		reportData.Url = urlData
		// проводим валидацию урла
		urlHost, errHost := url.Parse(urlData)
		var re = regexp.MustCompile(`^(https?:\/\/)?[-а-яё\w\d:.]*(\.)[-а-яё\w\d]{2,}.*`)
		if re.Match([]byte(urlData)) == false || errHost != nil {
			reportData.Comments = append(reportData.Comments, urlData+" - невалидный url")
			results <- reportData
		} else {
			// проверяем протокол
			if strings.HasPrefix(urlData, "http") == false {
				resp, err := http.Head("https://" + urlData)
				if err != nil || resp.StatusCode >= 400 {
					urlData = "http://" + urlData
				} else {
					urlData = "https://" + urlData
				}
			}

			// проверяем, что url отсутствует в представлении ViewBlockedDomain
			if _, found := blockedUrls[urlHost.Host]; found == true {
				reportData.UrlBlockedByRkn = 1
				reportData.Comments = append(reportData.Comments, "Блокировка по закону")
				results <- reportData
				return
			}

			// делаем запрос
			client := &http.Client{
				Timeout: 5 * time.Second,
			}
			req, err := http.NewRequest("GET", urlData, nil)
			if err == nil {
				req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36")
			}
			resp, err := client.Do(req)
			if err != nil {
				// обрабатываем ошибки
				if strings.Contains(err.Error(), "certificate") {
					reportData.Comments = append(reportData.Comments, "Не удалось распознать сертификат")
				} else if strings.Contains(err.Error(), "no such host") {
					reportData.Comments = append(reportData.Comments, "Неразрешенный домен")
				} else if strings.Contains(err.Error(), "request canceled") ||
					strings.Contains(err.Error(), "connection was forcibly closed") ||
					strings.Contains(err.Error(), "target machine actively refused it") {
					reportData.Comments = append(reportData.Comments, "Соединение сброшено или таймаут")
				} else if strings.Contains(err.Error(), "redirects") {
					reportData.Comments = append(reportData.Comments, "Слишком много редиректов")
				} else {
					//fmt.Println("Ошибка воркера ", id, urlData, err)
					reportData.Comments = append(reportData.Comments, "Требуется уточнение")
				}
				results <- reportData
			} else {
				var finalURL = urlData
				// узнаем статус
				var statusCode = responseStatusCode(urlData)
				reportData.UrlAvailable = 1
				reportData.UrlParsed = 1

				if statusCode >= 400 {
					reportData.UrlAvailable = 0
					reportData.UrlParsed = 0
					reportData.Comments = append(reportData.Comments, "Код ошибки: "+strconv.Itoa(statusCode))
					resp.Body.Close()
					results <- reportData
				} else {
					// находим редиректы
					if statusCode > 300 && statusCode < 400 {
						finalURL = resp.Request.URL.String()
						reportData.UrlRedirected = 1
						reportData.Comments = append(reportData.Comments, strconv.Itoa(statusCode)+" "+finalURL)
						// проверяем, что url отсутствует в представлении ViewBlockedDomain
						urlHost, _ := url.Parse(finalURL)
						if _, found := blockedUrls[urlHost.Host]; found == true {
							reportData.UrlBlockedByRkn = 1
							reportData.Comments = append(reportData.Comments, "Блокировка по закону")
							resp.Body.Close()
							results <- reportData
							return
						}
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
					// очищаем память
					resp = nil
					// возвращаем результат
					results <- reportData
				}
			}
		}
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
	result := resp.StatusCode
	resp.Body.Close()
	return result
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
					childUrlData, err := url.Parse(href)
					if err == nil && mainUrlData.Host == childUrlData.Host {
						href = mainUrlData.Scheme + ":" + href
						result = append(result, href)
					}
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
	siteText = strings.ReplaceAll(siteText, ".", "")
	siteText = strings.ReplaceAll(siteText, ",", "")
	siteText = strings.ReplaceAll(siteText, "!", "")
	siteText = strings.ReplaceAll(siteText, "?", "")
	siteText = strings.ReplaceAll(siteText, ":", "")
	siteText = strings.ReplaceAll(siteText, ";", "")
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

/**
Получаем список заблокированных сайтов
*/
func getBlockedUrlsMap() map[string]bool {
	var result = make(map[string]bool)
	file, err := ioutil.ReadFile(fileWithBlockedDomains)
	if err != nil {
		return result
	}

	var blockedUrls []string
	err = json.Unmarshal(file, &blockedUrls)
	if err != nil {
		return result
	}

	for _, urlStr := range blockedUrls {
		result[urlStr] = true
	}
	return result
}
