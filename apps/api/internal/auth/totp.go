package auth

import (
	"fmt"
	"net/url"

	"github.com/pquerna/otp/totp"
)

func GenerateTOTPSecret(issuer, account string) (secret, otpauthURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: account,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

func OTPAuthURL(issuer, account, secret string) string {
	label := url.PathEscape(fmt.Sprintf("%s:%s", issuer, account))
	return fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s", label, secret, url.QueryEscape(issuer))
}

func ValidateTOTP(secret, code string) bool {
	return totp.Validate(code, secret)
}
