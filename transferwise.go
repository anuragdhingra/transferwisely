package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "github.com/google/uuid"
    "log"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "time"
)

var host            = os.Getenv("HOST")
var margin          = os.Getenv("MARGIN")
var sourceAmount    = os.Getenv("SOURCE_AMOUNT")

const (
    transfersAPIPath = "v1/transfers"
    quotesAPIPath = "v1/quotes"
    liveRateAPIPath = "v1/rates"
    )

func checkAndProcess() (uint64, error) {
    result, transfer, err := compareRates()
    if err != nil {
        log.Println(err)
        return 0, fmt.Errorf("error reaching transferwise")
    }
    if !result {
        return 0, fmt.Errorf("current booked transfer is at higher rate")
    }

    newTransferId, err := createTransfer(transfer)
    if err != nil || !result {
        log.Println(err)
        return 0, fmt.Errorf("error reaching transferwise")
    }

    return newTransferId, nil
}

func compareRates() (result bool, bookedTransfer Transfer, err error) {
    empty := Transfer{}
    bookedTransfer, err = getBookedTransfer()
    if err != nil || bookedTransfer == empty {
        return false, empty, fmt.Errorf("compareRates: %v", err)
    }

    liveRate, err := getLiveRate()
    if err != nil || liveRate == 0 {
        return false, empty, fmt.Errorf("compareRates: %v", err)
    }

    marginRate, err := strconv.ParseFloat(margin, 64)
    bookedRate := bookedTransfer.Rate
    if liveRate < bookedRate || (liveRate - bookedRate >= marginRate) {
        return true, bookedTransfer, nil
    }

    return false, empty, nil
}

func createTransfer(oldTransfer Transfer) (uint64, error) {
    quoteId, err := generateQuote()
    if err != nil {
        return 0, fmt.Errorf("createTransfer: %v", err)
    }
    createRequest := CreateTransferRequest{
        TargetAccount:          oldTransfer.TargetAccount,
        Quote:                  quoteId,
        CustomerTransactionId:  uuid.New().String(),
    }
    request, _ := json.Marshal(createRequest)

    url := &url.URL{Host: host, Scheme: "https", Path: transfersAPIPath}
    response, code , err := callExternalAPI(http.MethodPost, url.String(), request)
    if err != nil || code != http.StatusOK {
        return 0, fmt.Errorf("error POST create transfer API: %v : %v", code, err)
    }

    data, ok := response.(map[string]interface{})
    if !ok {
        return 0, fmt.Errorf("createTransfer: error typecasting response")
    }

    newTransferId, ok := data["id"].(float64)
    if !ok {
        return 0, fmt.Errorf("error getting new transfer id: %v", err)
    }

    return uint64(newTransferId), nil
}

func getBookedTransfer() (Transfer, error) {
    params := url.Values{"limit": {"1"}, "offset": {"0"}, "status": {"incoming_payment_waiting"}}
    url := &url.URL{RawQuery: params.Encode(), Host: host, Scheme: "https", Path: transfersAPIPath}

    response, code, err := callExternalAPI(http.MethodGet, url.String(), nil)
    if err != nil || code != http.StatusOK {
        return Transfer{}, fmt.Errorf("error GET transfer list API: %v : %v", code, err)
    }

    data, ok := response.([]map[string]interface{})
    if !ok {
        return Transfer{}, fmt.Errorf("getBookedTransfer: error typecasting response")
    }

    if len(data) == 0 {
        return Transfer{}, fmt.Errorf("getBookedTransfer: no booked transfer found")
    }


    transferId, ok := data[0]["id"].(float64)
    if !ok {
        return Transfer{}, fmt.Errorf("getBookedTransfer: error typecasting transfer id")
    }

    targetAccount, ok := data[0]["targetAccount"].(float64)
    if !ok {
        return Transfer{}, fmt.Errorf("getBookedTransfer: error typecasting transfer account")
    }

    bookedRate, ok := data[0]["rate"].(float64)
    if !ok {
        return Transfer{}, fmt.Errorf("getBookedTransfer: error typecasting rate")
    }

    return Transfer{uint64(transferId), uint64(targetAccount), bookedRate}, nil
}

func getLiveRate() (float64, error) {
    params := url.Values{"source": {"PHP"}, "target": {"GBP"}}
    url := &url.URL{RawQuery: params.Encode(), Host: host, Scheme: "https", Path:liveRateAPIPath}

    response, code, err := callExternalAPI(http.MethodGet, url.String(), nil)
    if err != nil || code != http.StatusOK {
        return 0, fmt.Errorf("error GET live rate API: %v : %v", code, err)
    }

    data, ok := response.([]map[string]interface{})
    if !ok {
        return 0, fmt.Errorf("getLiveRate: error typecasting response")
    }

    liveRate, ok := data[0]["rate"].(float64)
    if !ok {
        return 0, fmt.Errorf("getLiveRate: error typecasting live rate")
    }

    return liveRate, nil
}

func generateQuote() (uint64, error) {

    quoteRequest := NewCreateQuoteRequest()
    quoteRequest.SourceAmount, _ = strconv.ParseUint(sourceAmount, 10, 64)
    request, _ := json.Marshal(quoteRequest)

    url := &url.URL{Host: host, Scheme: "https", Path: quotesAPIPath}
    response, code, err := callExternalAPI(http.MethodPost, url.String(), request)
    if err != nil || code != http.StatusOK {
        return 0, fmt.Errorf("error GET quote API: %v : %v", code, err)
    }

    data, ok := response.(map[string]interface{})
    if !ok {
        return 0, fmt.Errorf("generateQuote: error typecasting response")
    }

    quoteId, ok := data["id"].(float64)
    if !ok {
        return 0, fmt.Errorf("generateQuote: error typecasting quoteId")
    }

    return uint64(quoteId), nil
}

func callExternalAPI(method string, url string, reqBody []byte) (response interface{}, code int, err error) {

    client := &http.Client{Timeout: 10 * time.Second}
    req, err := http.NewRequest(method, url, bytes.NewReader(reqBody))
    if err != nil {
        return nil, http.StatusInternalServerError, fmt.Errorf("error creating external api request: %v", err)
    }
    req.Header.Add("Authorization", "Bearer " + os.Getenv("API_TOKEN"))
    req.Header.Add("Content-Type", "application/json")

    res, err := client.Do(req)
    if err != nil {
        return nil, http.StatusInternalServerError, fmt.Errorf("error calling external api: %v", err)
    }
    err = json.NewDecoder(res.Body).Decode(&response)
    if err != nil {
        return nil, http.StatusInternalServerError, fmt.Errorf("error decoding json response: %v", err)
    }
    code = res.StatusCode
    _ = res.Body.Close()

    return
}

type Transfer struct {
    Id              uint64      `json:"id"`
    TargetAccount   uint64      `json:"targetAccount"`
    Rate            float64     `json:"rate"`
}

type CreateTransferRequest struct {
    TargetAccount           uint64   `json:"targetAccount"`
    Quote                   uint64   `json:"quote"`
    CustomerTransactionId   string   `json:"customerTransactionId"`
}

type createQuoteRequest struct {
    Source          string  `json:"source"`
    Target          string  `json:"target"`
    RateType        string  `json:"rateType"`
    SourceAmount    uint64   `json:"sourceAmount"`
    Type            string  `json:"type"`
}

func NewCreateQuoteRequest() createQuoteRequest {
    return createQuoteRequest{
        Source:     "PHP",
        Target:     "GBP",
        RateType:   "FIXED",
        Type:       "REGULAR",
    }
}
