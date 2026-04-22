package app

func (a *App) Initialize() error {
	return a.store.Init()
}
