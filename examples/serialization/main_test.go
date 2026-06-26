package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeserializeArray(t *testing.T) {
	input := []byte(`{
		"Field": "3",
		"Hotels": {
			"Field": "",
			"Hotel": [{"Field": "1"}, {"Field": "2"}]
		}
	}`)

	var got Root
	if err := json.Unmarshal(input, &got); err != nil {
		t.Fatalf("%s", err)
	}

	want := Root{Field: "3", Hotels: Hotels{
		Hotel: []Hotel{{Field: "1"}, {Field: "2"}},
	}}
	assert.Equal(t, want, got)
}

func TestDeserializeSingleObject(t *testing.T) {
	input := []byte(`{
		"Field": "3",
		"Hotels": {
			"Field": "",
			"Hotel": {"Field": "1"}
		}
	}`)

	var got Root
	if err := json.Unmarshal(input, &got); err != nil {
		t.Fatalf("%s", err)
	}

	want := Root{Field: "3", Hotels: Hotels{
		Hotel: []Hotel{{Field: "1"}},
	}}
	assert.Equal(t, want, got)
}
