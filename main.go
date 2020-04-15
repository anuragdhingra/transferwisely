package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "github.com/go-chi/chi"
    "github.com/go-chi/chi/middleware"
    "net/http"
    "net/url"
    "os"
    "time"
)

var (
    host             =  os.Getenv("HOST")
    getProfilePath   =  os.Getenv("GET_PROFILE_API_PATH")
    getTransfersPath =  os.Getenv("GET_TRANSFERS_API_PATH")
)

func main() {
    r := chi.NewRouter()
    r.Use(middleware.Logger)

    r.Get("/transfers", getTransfers)
    fmt.Println("Starting server on port 3000")
    _ = http.ListenAndServe(":3000", r)
}

func getTransfers(w http.ResponseWriter, r *http.Request) {

    //_ = []byte(`{
	//"source": "JPY",
	//"target": "INR",
	//"rateType": "FIXED",
	//"sourceAmount": 46500,
	//"type": "REGULAR"}`)

    params := url.Values{"limit": {"1"}, "offset": {"0"}, "status": {"incoming_payment_waiting"}}
    url := &url.URL{RawQuery: params.Encode(), Host: host, Scheme: "https", Path: getTransfersPath}

    jsonResponse, code, err := callExternalAPI(http.MethodGet, url.String(), nil)
    if err != nil {
        fmt.Fprint(w, err)
        return
    }
    response, _ := json.Marshal(jsonResponse)

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
   w.Write(response)
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