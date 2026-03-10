package main

import (
    "fmt"
    "github.com/example/masking-tool-backend/internal/masking"
)

func main() {
    vals := []interface{}{[]byte("Farhan Kurnia"), []byte("farhan.kurnia@email.com"), 123}
    for _, v := range vals {
        var orig string
        switch vv := v.(type) {
        case []byte:
            orig = string(vv)
        default:
            orig = fmt.Sprintf("%v", vv)
        }
        fmt.Println("orig=", orig, "masked=", masking.MaskValue(orig, "name", nil))
    }
}
