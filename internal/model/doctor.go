package model

type CheckStatus struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type DoctorReport struct {
	Checks []CheckStatus `json:"checks"`
	AllOK  bool          `json:"all_ok"`
}
