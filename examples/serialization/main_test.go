package main

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

type Root struct {
	Field  string `json:"Field"`
	Hotels Hotels `json:"Hotels"`
}

type Hotels struct {
	Field string  `json:"Field"`
	Hotel []Hotel `json:"Hotel"`
}

type Hotel struct {
	Field string `json:"Field"`
}

func (r *Hotels) UnmarshalJSON(bytes []byte) error {
	var hotels []Hotel
	var hotel Hotel

	json := jsoniter.ConfigCompatibleWithStandardLibrary
	err1 := json.Unmarshal([]byte(jsoniter.Get(bytes, "Hotel").ToString()), &hotels)

	if err1 != nil {
		err2 := json.Unmarshal([]byte(jsoniter.Get(bytes, "Hotel").ToString()), &hotel)
		if err2 != nil {
			return err2
		}
	}

	if hotels != nil {
		r.Hotel = hotels
	} else {
		r.Hotel = []Hotel{hotel}
	}

	return nil
}

func TestDeserialize(t *testing.T) {

	byt := []byte(`{
		"Field": "3",
		"Hotels": {
			"Field": "",
			"Hotel": [
				{
					"Field": "1"
				},
				{
					"Field": "2"
				}
			]
		}
	}`)

	json := jsoniter.ConfigCompatibleWithStandardLibrary
	response := &Root{}
	if err := json.Unmarshal(byt, &response); err != nil {
		t.Errorf("%s", err.Error())
	}

	resp := Root{Field: "3", Hotels: Hotels{
		Hotel: []Hotel{
			{Field: "1"},
			{Field: "2"},
		},
	}}
	// data, _ := json.Marshal(resp)
	// fmt.Println(string(data))
	assert.Equal(t, resp, *response)
}

func TestDeserialize2(t *testing.T) {

	byt := []byte(`{
		"Field": "3",
		"Hotels": {
			"Field": "",
			"Hotel": 
				{
					"Field": "1"
				}
		}
	}`)

	json := jsoniter.ConfigCompatibleWithStandardLibrary
	response := &Root{}
	if err := json.Unmarshal(byt, &response); err != nil {
		t.Errorf("%s", err.Error())
	}

	resp := Root{Field: "3", Hotels: Hotels{
		Hotel: []Hotel{
			{Field: "1"},
		},
	}}

	// fmt.Println(len(response.HotelListResponse.HotelList.HotelSummary))
	assert.Equal(t, resp, *response)
}
