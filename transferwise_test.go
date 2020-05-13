package main

import (
    "bytes"
    "encoding/json"
    "github.com/bxcodec/faker/v3"
    "github.com/stretchr/testify/assert"
    "io/ioutil"
    "log"
    "net/http"
    "testing"
    "transferwisely/mocks"
)

func init() {
    Client = &mocks.Client{}
}

func TestGetDetailByQuoteId(t *testing.T)  {
    t.Run("success", func(t *testing.T) {
        // build response JSON
        q := QuoteDetail{}
        err := faker.FakeData(&q)
        if err != nil {
            log.Println(err)
        }
        j, _ := json.Marshal(q)
        // create a new reader with that JSON
        r := ioutil.NopCloser(bytes.NewReader(j))
        mocks.GetDoFunc = func(*http.Request) (*http.Response, error) {
            return &http.Response{
                StatusCode: http.StatusOK,
                Body:       r,
            }, nil
        }

        qd, err := getDetailByQuoteId("anything")
        assert.NotEmpty(t, qd)
        assert.Equal(t, q.Id, qd.Id)
        assert.NoError(t, err)
    })

    t.Run("external api error", func(t *testing.T) {
        // build response JSON
        q := QuoteDetail{}
        j, _ := json.Marshal(q)
        // create a new reader with that JSON
        r := ioutil.NopCloser(bytes.NewReader(j))
        mocks.GetDoFunc = func(*http.Request) (*http.Response, error) {
            return &http.Response{
                StatusCode: http.StatusInternalServerError,
                Body:       r,
            }, nil
        }

        qd, err := getDetailByQuoteId("anything")
        assert.Empty(t, qd)
        assert.Error(t, err)

    })
}

func TestGenerateQuote(t *testing.T)  {
    t.Run("success", func(t *testing.T) {
        // build response JSON
        q := QuoteDetail{}
        err := faker.FakeData(&q)
        if err != nil {
            log.Println(err)
        }
        j, _ := json.Marshal(q)
        // create a new reader with that JSON
        r := ioutil.NopCloser(bytes.NewReader(j))
        mocks.GetDoFunc = func(*http.Request) (*http.Response, error) {
            return &http.Response{
                StatusCode: http.StatusOK,
                Body:       r,
            }, nil
        }

        qId, err := generateQuote("anything", "anything", 1, 1)
        assert.NotEmpty(t, qId)
        assert.Equal(t, q.Id, qId)
        assert.NoError(t, err)
    })

    t.Run("external api error", func(t *testing.T) {
        // build response JSON
        q := QuoteDetail{}
        j, _ := json.Marshal(q)
        // create a new reader with that JSON
        r := ioutil.NopCloser(bytes.NewReader(j))
        mocks.GetDoFunc = func(*http.Request) (*http.Response, error) {
            return &http.Response{
                StatusCode: http.StatusInternalServerError,
                Body:       r,
            }, nil
        }

        qId, err := generateQuote("anything", "anything", 1, 1)
        assert.Empty(t, qId)
        assert.Error(t, err)
    })
}


