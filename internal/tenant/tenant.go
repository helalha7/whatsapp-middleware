package tenant

import "time"

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
	//TO BE IMPLEMENTED
	return nil
}
