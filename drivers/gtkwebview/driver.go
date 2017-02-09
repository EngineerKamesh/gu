package gkwebview

import (
	"github.com/gu-io/gu"
	"github.com/gu-io/gu/router"
	"github.com/gu-io/gu/shell"
)

// WebviewDriver provides a concrete implementation of the Gu.Driver interface.
type WebviewDriver struct {
	readyHandlers []func()
}

// Name returns the name of the driver.
func (WebviewDriver) Name() string {
	return "Gtk Webkit Webview()"
}

// OnReady registers the giving handle function to be called when the giving
// driver is ready and loaded.
func (WebviewDriver) OnReady(handle func()) {

}

// Location returns the current location of the browser.
func (WebviewDriver) Location() router.PushEvent {
	var route router.PushEvent
	return route
}

// Navigate takes the provided route and navigates the rendering system to the
// desired page.
func (WebviewDriver) Navigate(route router.PushDirectiveEvent) {

}

// OnRoute registers the NApp instance for route changes and re-rendering.
func (WebviewDriver) OnRoute(app *gu.NApp) {

}

// Render issues a clean rendering of all content clearing out the current content
// of the browser to the one provided by the appliation.
func (WebviewDriver) Render(app *gu.NApp) {

}

// Update updates a giving view portion of a giving App within the designated
// rendering system(browser) provided by the driver.
func (WebviewDriver) Update(app *gu.NApp, view *gu.NView) {

}

// Services returns the Fetcher and Cache associated with the provided cacheName.
// Intercepting requests for usage.
func (WebviewDriver) Services(cacheName string, intercept bool) (shell.Fetch, shell.Cache) {
	return nil, nil
}
