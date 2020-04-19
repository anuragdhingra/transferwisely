package main

import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "github.com/google/uuid"
    "github.com/mitchellh/mapstructure"
    "log"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "strings"
    "time"
)

// transfer-wise api paths
const (
    transfersAPIPath = "v1/transfers"
    quotesAPIPath = "v1/quotes"
    liveRateAPIPath = "v1/rates"
    cancelTransferAPIPath = "v1/transfers/{transferId}/cancel"
    )

// error
const ErrNoCurrentTransferFound  = "error: no current transfer found, please create a transfer before proceeding"

// fallback values for optional env variables
const (
    fallbackInterval = "1"
    fallbackMargin = "0"
)

// env vars
var hostVar         = getEnv("HOST", "")
var apiTokenVar     = getEnv("API_TOKEN", "")
var marginVar       = getEnv("MARGIN", fallbackMargin)
var intervalVar     = getEnv("INTERVAL", fallbackInterval)
var sourceAmountVar = getEnv("SOURCE_AMOUNT", "")


func checkAndProcess() {
    if hostVar == "" || apiTokenVar == "" {
        log.Println("error: env variables API_TOKEN and HOST are both required")
        return
    }

    result, transfer, err := compareRates()
    if err != nil {
        log.Println(err)
        return
    }
    if !result {
        log.Printf("NO ACTION NEEDED || Transfer ID: %v | {%v} --> {%v} | Amount: %v | Rate: %v ||",
            transfer.Id, transfer.SourceCurrency, transfer.TargetCurrency, transfer.SourceAmount, transfer.Rate)
        return
    }

    newTransfer, err := createTransfer(transfer)
    if err != nil || !result {
        log.Println(err)
        return
    }

    log.Printf("NEW TRANSFER BOOKED || Transfer ID: %v | {%v} --> {%v} | Amount: %v | Rate: %v ||",
        newTransfer.Id, newTransfer.SourceCurrency, newTransfer.TargetCurrency, newTransfer.SourceAmount, newTransfer.Rate)
}

func compareRates() (result bool, bookedTransfer Transfer, err error) {
    empty := Transfer{}
    bookedTransfer, err = getBookedTransfer()
    if err != nil || bookedTransfer == empty {
        return false, empty, fmt.Errorf("compareRates: %v", err)
    }

    liveRate, err := getLiveRate(bookedTransfer.SourceCurrency, bookedTransfer.TargetCurrency)
    if err != nil || liveRate == 0 {
        return false, empty, fmt.Errorf("compareRates: %v", err)
    }

    marginRate, err := strconv.ParseFloat(marginVar, 64)
    if err != nil {
        return false, empty, fmt.Errorf("compareRates: %v", err)
    }

    bookedRate := bookedTransfer.Rate
    if liveRate > bookedRate && (liveRate - bookedRate >= marginRate) {
        return true, bookedTransfer, nil
    }

    return false, bookedTransfer, nil
}

func getBookedTransfer() (Transfer, error) {
    params := url.Values{"limit": {"3"}, "offset": {"0"}, "status": {"incoming_payment_waiting"}}
    url := &url.URL{RawQuery: params.Encode(), Host: hostVar, Scheme: "https", Path: transfersAPIPath}

    response, code, err := callExternalAPI(http.MethodGet, url.String(), nil)
    if err != nil || code != http.StatusOK {
        return Transfer{}, fmt.Errorf("error GET transfer list API: %v : %v", code, err)
    }

    var transfersList []Transfer
    err = mapstructure.Decode(response, &transfersList)
    if err != nil {
        return Transfer{}, fmt.Errorf("error decoding response: %v", err)
    }

    if len(transfersList) == 0 {
        return Transfer{}, fmt.Errorf(ErrNoCurrentTransferFound)
    }

    bookedTransfer := findBestTransfer(transfersList)

    if bookedTransfer.Quote == 0 && sourceAmountVar == "" {
        return Transfer{}, errors.New("error: env variable SOURCE_AMOUNT is also required in this case")
    }

    if bookedTransfer.Quote == 0 {
        bookedTransfer.SourceAmount, err = strconv.ParseFloat(sourceAmountVar, 64)
        if err != nil {
            return Transfer{}, fmt.Errorf("getBookedTransfer: %v", err)
        }
    } else {
        quoteDetail, err := getDetailByQuoteId(bookedTransfer.Quote)
        if err != nil {
            return Transfer{}, fmt.Errorf("getBookedTransfer: %v", err)
        }
        bookedTransfer.SourceAmount = quoteDetail.SourceAmount
    }

    return bookedTransfer, nil
}

func getLiveRate(source string, target string) (float64, error) {
    params := url.Values{"source": {source}, "target": {target}}
    url := &url.URL{RawQuery: params.Encode(), Host: hostVar, Scheme: "https", Path:liveRateAPIPath}

    response, code, err := callExternalAPI(http.MethodGet, url.String(), nil)
    if err != nil || code != http.StatusOK {
        return 0, fmt.Errorf("error GET live rate API: %v : %v", code, err)
    }

    var liveRate []LiveRate
    err = mapstructure.Decode(response, &liveRate)
    if err != nil {
        return 0, fmt.Errorf("error decoding live rate response: %v", err)
    }

    return liveRate[0].Rate, nil
}

func createTransfer(oldTransfer Transfer) (Transfer, error) {
    quoteId, err := generateQuote(oldTransfer.SourceCurrency, oldTransfer.TargetCurrency, oldTransfer.SourceAmount)
    if err != nil {
        return Transfer{}, fmt.Errorf("createTransfer: %v", err)
    }
    createRequest := CreateTransferRequest{
        TargetAccount:          oldTransfer.TargetAccount,
        Quote:                  quoteId,
        CustomerTransactionId:  uuid.New().String(),
    }
    request, _ := json.Marshal(createRequest)

    url := &url.URL{Host: hostVar, Scheme: "https", Path: transfersAPIPath}
    response, code , err := callExternalAPI(http.MethodPost, url.String(), request)
    if err != nil || code != http.StatusOK {
        return Transfer{}, fmt.Errorf("error POST create transfer API: %v : %v", code, err)
    }

    var newTransfer Transfer
    err = mapstructure.Decode(response, &newTransfer)
    if err != nil {
        return Transfer{}, fmt.Errorf("error decoding response: %v", err)
    }
    newTransfer.SourceAmount = oldTransfer.SourceAmount

    cancelResult, err := cancelTransfer(oldTransfer.Id)
    if !cancelResult || err != nil {
       log.Println("Error deleting old transfer")
    }

    return newTransfer, nil
}

func cancelTransfer(transferId uint64) (bool, error) {
    path := strings.Replace(cancelTransferAPIPath, "{transferId}", strconv.FormatUint(transferId, 10), 1)

    url := &url.URL{Host: hostVar, Scheme: "https", Path: path}
    _, code , err := callExternalAPI(http.MethodPut, url.String(), nil)
    if err != nil || code != http.StatusOK {
        return false, fmt.Errorf("error PUT cancel transfer API: %v : %v", code, err)
    }

    return true, nil
}

func generateQuote(source string, target string, sourceAmount float64) (uint64, error) {
    quoteRequest := NewCreateQuoteRequest()
    quoteRequest.Source = source
    quoteRequest.Target = target
    quoteRequest.SourceAmount = sourceAmount

    request, _ := json.Marshal(quoteRequest)

    url := &url.URL{Host: hostVar, Scheme: "https", Path: quotesAPIPath}
    response, code, err := callExternalAPI(http.MethodPost, url.String(), request)
    if err != nil || code != http.StatusOK {
        return 0, fmt.Errorf("error POST quote API: %v : %v", code, err)
    }

    var quote QuoteDetail
    err = mapstructure.Decode(response, &quote)
    if err != nil {
        return 0, fmt.Errorf("error decoding quote response: %v", err)
    }

    return quote.Id, nil
}

func getDetailByQuoteId(quoteId uint64) (QuoteDetail, error) {
    path := quotesAPIPath + "/" + strconv.FormatUint(quoteId, 10)
    url := &url.URL{Host: hostVar, Scheme: "https", Path: path}

    response, code, err := callExternalAPI(http.MethodGet, url.String(), nil)
    if err != nil || code != http.StatusOK {
        return QuoteDetail{}, fmt.Errorf("error GET quote detail API: %v : %v", code, err)
    }

    var quoteDetail QuoteDetail
    err = mapstructure.Decode(response, &quoteDetail)
    if err != nil || code != http.StatusOK {
        return QuoteDetail{}, fmt.Errorf("error decoding to quote detail: %v : %v", code, err)
    }

    return quoteDetail, nil
}

func callExternalAPI(method string, url string, reqBody []byte) (response interface{}, code int, err error) {
    client := &http.Client{Timeout: 10 * time.Second}
    req, err := http.NewRequest(method, url, bytes.NewReader(reqBody))
    if err != nil {
        return nil, http.StatusInternalServerError, fmt.Errorf("error creating external api request: %v", err)
    }
    req.Header.Add("Authorization", "Bearer " + apiTokenVar)
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

func findBestTransfer(transferList []Transfer) (bestTransfer Transfer){
    for i := range transferList {
        if i==0 || bestTransfer.Rate < transferList[i].Rate  {
            bestTransfer = transferList[i]
        }
    }
    return
}

func getEnv(key, fallback string) string {
    if value, ok := os.LookupEnv(key); ok {
        return value
    }

    return fallback
}

type Transfer struct {
    Id              uint64      `json:"id"`
    TargetAccount   uint64      `json:"targetAccount"`
    SourceAmount    float64     `json:"sourceAmount"`
    Rate            float64     `json:"rate"`
    Quote           uint64      `json:"quote"`
    SourceCurrency  string      `json:"sourceCurrency"`
    TargetCurrency  string      `json:"targetCurrency"`
}

type QuoteDetail struct {
    Id              uint64      `json:"id"`
    SourceAmount    float64     `json:"sourceAmount"`
    Rate            float64     `json:"rate"`
    Source          string      `json:"source"`
    Target          string      `json:"target"`
}

type LiveRate struct {
    Rate  float64 `json:"rate"`
}

type CreateTransferRequest struct {
    TargetAccount           uint64   `json:"targetAccount"`
    Quote                   uint64   `json:"quote"`
    CustomerTransactionId   string   `json:"customerTransactionId"`
}

type CreateQuoteRequest struct {
    Source          string      `json:"source"`
    Target          string      `json:"target"`
    RateType        string      `json:"rateType"`
    SourceAmount    float64     `json:"sourceAmount"`
    Type            string      `json:"type"`
}

func NewCreateQuoteRequest() CreateQuoteRequest {
    return CreateQuoteRequest{
        RateType:   "FIXED",
        Type:       "REGULAR",
    }
}
