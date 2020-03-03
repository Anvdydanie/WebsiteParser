package handlers

import (
	"RobotChecker/parser"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type ReqBody struct {
	Urls       []map[string]string `json:"urls"`
	CheckId    int                 `json:"checkId"`
	References map[string][]string `json:"references"`
	MinWords   int                 `json:"minWords"`
	MinImgs    int                 `json:"minImgs"`
	//ServiceTurnover map[string]int`json:"serviceTurnover"`
}

func StartCheckHandler(resp http.ResponseWriter, req *http.Request) {
	var reqBody = new(ReqBody)
	body, err := ioutil.ReadAll(req.Body)

	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(body, &reqBody)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	// парсим сайты
	_result := parser.Parser(reqBody.Urls, reqBody.CheckId, reqBody.References, reqBody.MinWords, reqBody.MinImgs)
	// переводим ответ в json
	result, err := json.Marshal(_result)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	/**
	TODO
	1. Добавить redis и проверку на одинаковые урлы
	2. Добавить serviceTurnover
	3. Добавить обработку процента через redis
	4. Останавливать проверку
	*/

	resp.Write(result)
}
