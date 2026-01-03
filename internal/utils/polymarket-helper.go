package utils

import (
	"fmt"
	"os"

	"github.com/xiangxn/go-polymarket-sdk/utils"
)

func FormatSlug(template string) string {
	out := os.Expand(template, func(key string) string {
		if key == "time" {
			return fmt.Sprint(utils.RoundTo15Minutes())
		}
		return ""
	})
	return out
}
