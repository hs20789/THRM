package guiapp

import (
	"fmt"

	"github.com/TIANLI0/THRM/internal/ipc"
)

// SetUILocale updates native UI preferences without writing device configuration.
func (a *App) SetUILocale(locale string) error {
	resp, err := a.sendRequest(ipc.ReqSetUILocale, ipc.SetStringParams{Value: locale})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}
