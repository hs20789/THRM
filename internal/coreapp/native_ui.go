package coreapp

// Serialize persistence and tray updates together without taking a device lock.
func (a *CoreApp) setUILocale(locale string) error {
	a.uiLocaleMutex.Lock()
	defer a.uiLocaleMutex.Unlock()
	if err := a.uiLocale.Set(locale); err != nil {
		return err
	}
	a.trayManager.SetLocale(a.uiLocale.Snapshot())
	return nil
}
