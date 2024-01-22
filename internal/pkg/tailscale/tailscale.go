package tailscale

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type TailscaleStatus struct {
	BackendState string `json:"BackendState"`
	Self         struct {
		Online bool `json:"Online"`
	} `json:"Self"`
}

func CheckStatus() (TailscaleStatus, error) {
	cmd := exec.Command("tailscale", "status", "--peers=false", "--json")

	out, err := cmd.Output()
	if err != nil {
		return TailscaleStatus{}, fmt.Errorf("running tailscale status command: %w", err)
	}

	var t TailscaleStatus
	err = json.Unmarshal(out, &t)
	if err != nil {
		return TailscaleStatus{}, fmt.Errorf("unmarshalling tailscale status: %w", err)
	}

	return t, nil
}
