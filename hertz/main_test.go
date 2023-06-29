package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	api "kitex/kitex_gen/api"
	"net/http"
	"strings"
	"testing"
)

func TestAPI(t *testing.T) {
	url := "http://127.0.0.1:8887/bank/name"
	method := "GET"

	payload := strings.NewReader(`{
    "Name": "Citi "
}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))

	var response api.BankNameResp
	//json.Unmarshal(body, &response)

	json.NewDecoder(res.Body).Decode(&response)

	fmt.Println(response)
	fmt.Println(response)

	// if string(body) != "\"{\"RespBody\":\"Citi BANK\"}\"" {
	// 	t.Fatalf("Expected %v to equal to {\"RespBody\":\"Citi BANK\"}", string(body))
	// }

}
