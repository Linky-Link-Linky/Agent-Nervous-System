package model

type DoctorCheck struct {
    Name   string `json:"name"`
    Value  string `json:"value"`
    Status string `json:"status"`
    Detail string `json:"detail,omitempty"`
}

type DoctorReport struct {
    Checks []*DoctorCheck `json:"checks"`
    AllOK  bool           `json:"all_ok"`
}
