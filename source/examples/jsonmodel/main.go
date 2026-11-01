package main

import (
	"encoding/json"
	"fmt"
)

type Reading struct {
	Room    string  `json:"room"`
	Celsius float64 `json:"celsius"`
}

func main() {
	data, e := json.Marshal(Reading{Room: "Lab A", Celsius: 21.5})
	if e != nil {
		panic(e)
	}
	fmt.Println(string(data))
}
