package services

import (
	"errors"
	"math/big"
	"strings"
)

// ConvertBase converts a number string from inBase (2..36) to outBase (2..36).
func ConvertBase(inNo string, inBase, outBase int) (string, error) {
	inNo = strings.TrimSpace(inNo)
	if inNo == "" {
		return "0", nil
	}
	if inBase < 2 || inBase > 36 || outBase < 2 || outBase > 36 {
		return "", errors.New("进制必须在 2 到 36 之间")
	}

	bi := new(big.Int)
	_, ok := bi.SetString(inNo, inBase)
	if !ok {
		return "", errors.New("无效输入数值")
	}

	return strings.ToUpper(bi.Text(outBase)), nil
}

