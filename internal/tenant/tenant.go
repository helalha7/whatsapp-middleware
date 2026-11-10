package tenant

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

type Tenant struct {
	ID                 int       `json:"id"`
	CreatedAt          time.Time `json:"created_at"`
	Version            int       `json:"-"`
	Name               string    `json:"name"`
	WABAID             string    `json:"waba_id"`
	PhoneNumberID      string    `json:"phone_number_id"`
	DisplayPhoneNumber string    `json:"display_phone_number"`
	AccessToken        string    `json:"-"`
}

func (t *Tenant) Validate() map[string]string {
	errors := make(map[string]string)

	validateRequired(errors, "name", t.Name)
	validateMaxLength(errors, "name", t.Name, 150)

	validateRequired(errors, "waba_id", t.WABAID)
	validateMaxLength(errors, "waba_id", t.WABAID, 100)

	validateRequired(errors, "phone_number_id", t.PhoneNumberID)
	validateMaxLength(errors, "phone_number_id", t.PhoneNumberID, 100)

	validateRequired(errors, "display_phone_number", t.DisplayPhoneNumber)
	validateMaxLength(errors, "display_phone_number", t.DisplayPhoneNumber, 30)

	validateRequired(errors, "access_token", t.AccessToken)

	if len(errors) == 0 {
		return nil
	}

	return errors
}

func validateRequired(errors map[string]string, field string, value string) {
	if strings.TrimSpace(value) == "" {
		errors[field] = "must be provided"
	}
}

func validateMaxLength(errors map[string]string, field string, value string, max int) {
	if utf8.RuneCountInString(value) > max {
		errors[field] = "must not be more than " + fmt.Sprint(max) + " characters long"
	}
}
