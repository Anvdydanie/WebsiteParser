package handlers

import (
	"RobotChecker/logger"
	"RobotChecker/parser"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type ReqBody struct {
	Urls            []map[string]string      `json:"urls"`
	CheckId         int                      `json:"checkId"`
	References      map[string][]string      `json:"references"`
	MinWords        int                      `json:"minWords"`
	MinImgs         int                      `json:"minImgs"`
	ServiceTurnover map[string]float32       `json:"serviceTurnover"`
	DuplicUrls      map[string]parser.Report `json:"duplicateUrls"`
}

func StartCheckHandler(resp http.ResponseWriter, req *http.Request) {
	var reqBody = new(ReqBody)
	body, err := ioutil.ReadAll(req.Body)

	if err != nil {
		// логируем ошибку
		logger.Logger("Ошибка при получении запроса от nodejs: " + err.Error())
		// отправляем ответ
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(body, &reqBody)
	if err != nil {
		// логируем ошибку
		logger.Logger("Ошибка при парсинге тела запроса от nodejs: " + err.Error())
		// отправляем ответ
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	// парсим сайты
	_result := parser.Parser(reqBody.Urls, reqBody.CheckId, reqBody.References, reqBody.MinWords, reqBody.MinImgs, reqBody.DuplicUrls, reqBody.ServiceTurnover)
	// переводим ответ в json
	result, err := json.Marshal(_result)
	if err != nil {
		// логируем ошибку
		logger.Logger("Ошибка при формировании результатов отчета в json для nodejs: " + err.Error())
		// отправляем ответ
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	resp.Write(result)
}
