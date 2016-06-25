package mint

type AssetResource struct {
	ID       string `json:"id"`
	Created  int64  `json:"created"`
	Livemode bool   `json:"livemode"`

	Name   string `json:"name"`
	Issuer string `json:"issuer"`
	Code   string `json:"code"`
	Scale  int8   `json:"scale"`
}
