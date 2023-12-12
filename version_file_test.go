package version_file

import (
	"fmt"
	"testing"
)

type Data struct {
	A int    `json:"a" validate:"required"`
	B string `json:"bbb"`
}

func Test_a(t *testing.T) {
	vf, err := New("data")
	fmt.Println(vf, err)
	// vf.PushJson(Data{A: 123})
}
