package gu

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/gu-io/gu/notifications"
	"github.com/gu-io/gu/router"
	"github.com/gu-io/gu/shell"
	"github.com/gu-io/gu/trees"
	"github.com/gu-io/gu/trees/elems"
	"github.com/influx6/faux/reflection"
)

// Mode defines a type to represent the mode which the library works under.
type Mode int

const (
	// ProductionMode defines production mode for which defines how the context
	// which it behaves as regards certain features.
	ProductionMode Mode = iota

	// DevelopmentMode defines production mode for which defines how the context
	// which it behaves as regards certain features.
	DevelopmentMode
)

// RenderingOrder defines a type used to define the order which rendering is to be done for a resource.
type RenderingOrder int

const (
	// FirstOrder defines that rendering be first in order.
	FirstOrder RenderingOrder = iota

	// AnyOrder defines that rendering be middle in order.
	AnyOrder

	// LastOrder defines that rendering be last in order.
	LastOrder
)

// Driver defines an interface which provides an Interface to render Apps and views
// to a display system. eg. GopherJS, wkWebview.
type Driver interface {
	// Method to subscribe to ready state for the driver.
	OnReady(func())

	// Name of the driver.
	Name() string

	// Navigate the Driver to the provided path.
	Navigate(router.PushDirectiveEvent)

	// Current Location of the driver path.
	Location() router.PushEvent

	// OnRoute registers for route change notifications to react accordingly.
	// This does a totally re-write of the whole display.
	OnRoute(*NApp)

	// Render app and it's content.
	Render(*NApp)

	// Update app's view and it's target content.
	Update(*NApp, ...*NView)

	// Services returns a new resource fetcher which can be used to retrieve Resources.
	// Fetcher requires the cacheName and a boolean indicating if it should intercept
	// all requests for resources.
	Services(cacheName string, interceptRequests bool) (shell.Fetch, shell.Cache)
}

// Resource defines any set of rendering links, scripts, styles needed by a view.
type Resource struct {
	Manifest shell.AppManifest
	body     []*trees.Markup
	head     []*trees.Markup
}

// AppAttr defines a struct for
type AppAttr struct {
	Mode              Mode   `json:"mode"`
	Name              string `json:"name"`
	Title             string `json:"title"`
	Manifests         string `json:"manifests"`
	EnableManifest    bool   `json:"ignore_manifest"`
	InterceptRequests bool   `json:"intercept_requests"`
	Driver            Driver `json:"-"`
}

// NApp defines a struct which encapsulates all the core view management functions
// for views.
type NApp struct {
	uuid          string
	attr          AppAttr
	driver        Driver
	cache         shell.Cache
	fetch         shell.Fetch
	notifications *notifications.AppNotification
	active        bool

	local       []shell.AppManifest
	views       []NView
	activeViews []NView

	globalResources []Resource

	tree *trees.Markup
}

// App creates a new app structure to rendering gu components.
func App(attr AppAttr) *NApp {
	var app NApp
	app.attr = attr
	app.uuid = NewKey()
	app.driver = attr.Driver

	// Add local notification channel for this giving app.
	app.notifications = notifications.New(app.uuid)

	fetch, cache := attr.Driver.Services(attr.Name, attr.InterceptRequests)
	app.cache = cache
	app.fetch = fetch

	// If we are in development mode empty the cache and reset for new use.
	if attr.Mode == DevelopmentMode {
		if err := app.cache.Empty(); err != nil {
			fmt.Printf("Failed to clear internal cache for %q in development mode: %q\n", app.attr.Name, err.Error())
		}
	}

	// Attempt to retrieve a manifest.json from the backend.
	if attr.EnableManifest && attr.Manifests != "" {
		var appm []shell.AppManifest

		if _, response, err := app.fetch.Get(app.attr.Manifests, shell.DefaultStrategy); err == nil {
			fmt.Println(fmt.Sprintf("Failed to retrieve `manifests.json` due to network error: %q", err.Error()))
		} else {
			if err := json.Unmarshal(response.Body, &appm); err != nil {
				fmt.Println(fmt.Sprintf("Failed to load shell.AppManifest, resource loading is unavailable: %q", err.Error()))
			} else {

				for _, mani := range appm {
					for _, attr := range mani.Manifests {
						if attr.IsGlobal {
							shell.RegisterManifest(attr)
						}
					}

					if mani.GlobalScope {
						app.globalResources = append(app.globalResources, Resource{
							Manifest: mani,
						})

						continue
					}

					app.local = append(app.local, mani)
				}
			}
		}
	}

	app.driver.OnReady(func() {
		app.active = false
		app.driver.Render(&app)
		app.active = true
	})

	app.notifications.Subscribe(func(directive router.PushDirectiveEvent) {
		if !app.active {
			return
		}

		app.driver.Navigate(directive)
	})

	app.driver.OnRoute(&app)

	return &app
}

// Active returns true/false if the giving app is active and has already
// received rendering.
func (app *NApp) Active() bool {
	return app.active
}

// Mounted notifies all active views that they have been mounted.
func (app *NApp) Mounted() {
	for _, view := range app.activeViews {
		view.Mounted()
	}
}

// ActivateRoute actives the views which are to be rendered.
func (app *NApp) ActivateRoute(es interface{}) {
	var pe router.PushEvent

	switch esm := es.(type) {
	case string:
		tmp, err := router.NewPushEvent(esm, true)
		if err != nil {
			panic(fmt.Sprintf("Unable to create PushEvent for (URL: %q) -> %q\n", esm, err.Error()))
		}

		pe = tmp
	case router.PushEvent:
		pe = esm
	}

	app.activeViews = app.PushViews(pe)
}

// TreeJSON defines a struct which holds the giving sets of tree changes to be
// rendered.
type TreeJSON struct {
	AppID         string             `json:"AppId"`
	Name          string             `json:"Name"`
	Title         string             `json:"Title"`
	Head          []ViewJSON         `json:"Head"`
	Body          []ViewJSON         `json:"Body"`
	HeadResources []trees.MarkupJSON `json:"HeadResources"`
	BodyResources []trees.MarkupJSON `json:"BodyResources"`
}

// RenderJSON returns the giving rendered tree of the app respective of the path
// found as jons structure with markup content.
func (app *NApp) RenderJSON(es interface{}) TreeJSON {
	if es != nil {
		app.ActivateRoute(es)
	}

	var tjson TreeJSON
	tjson.Name = app.attr.Name
	tjson.Title = app.attr.Title

	toHead, toBody := app.Resources()

	for _, item := range toHead {
		tjson.HeadResources = append(tjson.HeadResources, item.TreeJSON())
	}

	for _, item := range toBody {
		tjson.BodyResources = append(tjson.BodyResources, item.TreeJSON())
	}

	var afterBody []ViewJSON

	for _, view := range app.activeViews {
		switch view.attr.Target {
		case HeadTarget:
			tjson.Head = append(tjson.Head, view.RenderJSON())
		case BodyTarget:
			tjson.Body = append(tjson.Body, view.RenderJSON())
		case AfterBodyTarget:
			afterBody = append(afterBody, view.RenderJSON())
		}

		viewHead, viewBody := view.Resources()
		for _, item := range viewHead {
			tjson.HeadResources = append(tjson.HeadResources, item.TreeJSON())
		}

		for _, item := range viewBody {
			tjson.BodyResources = append(tjson.BodyResources, item.TreeJSON())
		}
	}

	tjson.Body = append(tjson.Body, afterBody...)

	return tjson
}

// Render returns the giving rendered tree of the app respective of the path
// found.
func (app *NApp) Render(es interface{}) *trees.Markup {
	if es != nil {
		app.ActivateRoute(es)
	}

	var html = trees.NewMarkup("html", false)
	var head = trees.NewMarkup("head", false)

	var body = trees.NewMarkup("body", false)
	trees.NewAttr("gu-app-id", app.uuid).Apply(body)

	// var app = trees.NewMarkup("app", false)
	// trees.NewAttr("gu-app-id", app.uuid).Apply(app)

	head.Apply(html)
	body.Apply(html)

	// Generate the resources according to the received data.
	toHead, toBody := app.Resources()
	head.AddChild(toHead...)

	var last = elems.Div()

	for _, view := range app.activeViews {
		switch view.attr.Target {
		case HeadTarget:
			view.Render().Apply(head)
		case BodyTarget:
			view.Render().Apply(body)
		case AfterBodyTarget:
			view.Render().Apply(last)
		}

		viewHead, viewBody := view.Resources()

		// Add the headers into the header so they load accordingly.
		head.AddChild(viewHead...)

		// Append the resources into the body has we need them last.
		toBody = append(toBody, viewBody...)
	}

	body.AddChild(last.Children()...)

	body.AddChild(toBody...)

	// Ensure to have this gc'ed.
	last = nil

	return html
}

// PushViews returns a slice of  views that match and pass the provided path.
func (app *NApp) PushViews(event router.PushEvent) []NView {
	var active []NView

	for _, view := range app.views {
		if _, _, ok := view.router.Test(event.String()); ok {
			// Notify view to appropriate proper action when view does not match.
			view.router.Resolve(event)
			continue
		}

		view.propagateRoute(event)
		active = append(active, view)
	}

	return active
}

// Resources return the giving resource headers which relate with the
// view.
func (app *NApp) Resources() ([]*trees.Markup, []*trees.Markup) {
	var head, body []*trees.Markup

	head = append(head, elems.Title(elems.Text(app.attr.Title)))

	for _, def := range app.globalResources {
		if def.body != nil || def.head != nil {
			head = append(head, def.head...)
			body = append(body, def.body...)
			continue
		}

		if def.Manifest.Manifests == nil {
			continue
		}

		for _, manifest := range def.Manifest.Manifests {
			if !manifest.Init {
				continue
			}

			hook, err := shell.Get(manifest.HookName)
			if err != nil {
				fmt.Printf("Hook[%q] does not exists: Resource[%q] unable to install\n", manifest.HookName, manifest.Name)
				continue
			}

			markup, toHead, err := hook.Fetch(app.fetch, manifest)
			if err != nil {
				fmt.Printf("Hook[%q] failed to retrieve Resource {Name: %q, Path: %q}\n", manifest.HookName, manifest.Name, manifest.Path)
				continue
			}

			trees.NewAttr("gu-resource", "true").Apply(markup)
			trees.NewAttr("gu-resource-view", app.uuid).Apply(markup)
			trees.NewAttr("gu-resource-from", manifest.Path).Apply(markup)
			trees.NewAttr("gu-resource-name", manifest.Name).Apply(markup)
			trees.NewAttr("gu-resource-id", manifest.ID).Apply(markup)
			trees.NewAttr("gu-resource-app-id", app.uuid).Apply(markup)

			if toHead {
				def.head = append(def.head, markup)
				head = append(head, markup)
				continue
			}

			def.body = append(def.body, markup)
			body = append(body, markup)
		}
	}

	return head, body
}

// UUID returns the uuid specific to the giving view.
func (app *NApp) UUID() string {
	return app.uuid
}

// ViewTarget defines a concrete type to define where the view should be rendered.
type ViewTarget int

const (

	// BodyTarget defines the view target where the view is rendered in the body.
	BodyTarget ViewTarget = iota

	// HeadTarget defines the view target where the view is rendered in the head.
	HeadTarget

	// AfterBodyTarget defines the view target where the view is rendered after
	// body views content. Generally the browser moves anything outside of the body
	// into the body as last elements. So this will be the last elements rendered
	// in the border accordingly in the order they are added into the respective app.
	AfterBodyTarget
)

// ViewAttr defines a structure to define a option values for setting up the appropriate
// settings for the view.
type ViewAttr struct {
	Name   string        `json:"name"`
	Route  string        `json:"route"`
	Target ViewTarget    `json:"target"`
	Base   *trees.Markup `json:"base"`
}

// View returns a new instance of the view object.
func (app *NApp) View(attr ViewAttr) *NView {
	if attr.Base == nil {
		attr.Base = trees.NewMarkup("view", false)
		trees.NewCSSStyle("display", "block").Apply(attr.Base)
	}

	var vw NView
	vw.Reactive = NewReactive()
	vw.attr = attr
	vw.driver = app.driver
	vw.notifications = app.notifications
	vw.uuid = NewKey()
	vw.appUUID = app.uuid

	vw.router = router.New(attr.Route)
	vw.cache = app.cache
	vw.fetch = app.fetch
	vw.local = app.local

	vw.React(func() {
		if !app.active || app.driver == nil {
			return
		}

		app.driver.Update(app, &vw)
	})

	// Register to listen for failure of route to match and
	// notify unmount call.
	vw.router.Failed(func(push router.PushEvent) {
		vw.disableView()
		vw.Unmounted()
	})

	vw.attr.Base.SwapUID(vw.uuid)

	app.views = append(app.views, vw)

	return &vw
}

// RenderableData defines a struct which contains the name of a giving renderable
// and it's package.
type RenderableData struct {
	Name string
	Pkg  string
}

// NView defines a structure to encapsulates all rendering component for a given
// view.
type NView struct {
	Reactive
	uuid          string
	appUUID       string
	active        bool
	cache         shell.Cache
	fetch         shell.Fetch
	router        router.Resolver
	driver        Driver
	attr          ViewAttr
	notifications *notifications.AppNotification

	renderingData []RenderableData
	local         []shell.AppManifest

	localResources []Resource

	beginComponents []Component
	anyComponents   []Component
	lastComponents  []Component
}

// UUID returns the uuid specific to the giving view.
func (v *NView) UUID() string {
	return v.uuid
}

// ViewJSON defines a struct which holds the giving sets of view changes to be
// rendered.
type ViewJSON struct {
	AppID      string           `json:"AppID"`
	ViewID     string           `json:"ViewID"`
	ViewTarget int              `json:"ViewTarget"`
	Tree       trees.MarkupJSON `json:"Tree"`
}

// RenderJSON returns the ViewJSON for the provided View and its current events and
// changes.
func (v *NView) RenderJSON() ViewJSON {
	return ViewJSON{
		AppID:      v.appUUID,
		ViewID:     v.uuid,
		ViewTarget: int(v.attr.Target),
		Tree:       v.Render().TreeJSON(),
	}
}

// Render returns the markup for the giving views.
func (v *NView) Render() *trees.Markup {
	base := v.attr.Base.Clone()
	// Update the base hash.
	base.UpdateHash()

	// Process the begin components and immediately add appropriately into base.
	for _, component := range v.beginComponents {
		if component.attr.Target != "" {
			component.Render().ApplyMorphers().Apply(base)
			continue
		}

		render := component.Render().ApplyMorphers()
		targets := trees.Query.QueryAll(base, component.attr.Target)
		for _, target := range targets {
			target.AddChild(render)
		}
	}

	// Process the middle components and immediately add appropriately into base.
	for _, component := range v.anyComponents {
		if component.attr.Target != "" {
			component.Render().ApplyMorphers().Apply(base)
			continue
		}

		render := component.Render().ApplyMorphers()
		targets := trees.Query.QueryAll(base, component.attr.Target)
		for _, target := range targets {
			target.AddChild(render)
		}
	}

	// Process the last components and immediately add appropriately into base.
	for _, component := range v.lastComponents {
		if component.attr.Target != "" {
			component.Render().ApplyMorphers().Apply(base)
			continue
		}

		render := component.Render().ApplyMorphers()
		targets := trees.Query.QueryAll(base, component.attr.Target)
		for _, target := range targets {
			target.AddChild(render)
		}
	}

	return base
}

// Attr returns the views ViewAttr.
func (v *NView) Attr() ViewAttr {
	return v.attr
}

// propagateRoute supplies the needed route into the provided
func (v *NView) propagateRoute(pe router.PushEvent) {
	v.router.Resolve(pe)
}

// Resources return the giving resource headers which relate with the
// view.
func (v *NView) Resources() ([]*trees.Markup, []*trees.Markup) {
	var head, body []*trees.Markup

	for _, def := range v.localResources {
		if def.body != nil || def.head != nil {
			head = append(head, def.head...)
			body = append(body, def.body...)
			continue
		}

		if def.Manifest.Manifests == nil {
			continue
		}

		for _, manifest := range def.Manifest.Manifests {
			if !manifest.Init {
				continue
			}

			hook, err := shell.Get(manifest.HookName)
			if err != nil {
				fmt.Printf("Hook[%q] does not exists: Resource[%q] unable to install\n", manifest.HookName, manifest.Name)
				continue
			}

			markup, toHead, err := hook.Fetch(v.fetch, manifest)
			if err != nil {
				fmt.Printf("Hook[%q] failed to retrieve Resource {Name: %q, Path: %q}\n", manifest.HookName, manifest.Name, manifest.Path)
				continue
			}

			trees.NewAttr("gu-resource", "true").Apply(markup)
			trees.NewAttr("gu-resource-view", v.uuid).Apply(markup)
			trees.NewAttr("gu-resource-from", manifest.Path).Apply(markup)
			trees.NewAttr("gu-resource-name", manifest.Name).Apply(markup)
			trees.NewAttr("gu-resource-id", manifest.ID).Apply(markup)

			if toHead {
				def.head = append(def.head, markup)
				head = append(head, markup)
				continue
			}

			def.body = append(def.body, markup)
			body = append(body, markup)
		}
	}

	return head, body
}

// Unmounted publishes changes notifications that the component is unmounted.
func (v *NView) Unmounted() {
	for _, component := range v.beginComponents {
		component.Unmounted.Publish()
	}
	for _, component := range v.anyComponents {
		component.Unmounted.Publish()
	}
	for _, component := range v.lastComponents {
		component.Unmounted.Publish()
	}
}

// Updated publishes changes notifications that the component is updated.
func (v *NView) Updated() {
	for _, component := range v.beginComponents {
		component.Updated.Publish()
	}
	for _, component := range v.anyComponents {
		component.Updated.Publish()
	}
	for _, component := range v.lastComponents {
		component.Updated.Publish()
	}
}

// Mounted publishes changes notifications that the component is mounted.
func (v *NView) Mounted() {
	for _, component := range v.beginComponents {
		component.Mounted.Publish()
	}
	for _, component := range v.anyComponents {
		component.Mounted.Publish()
	}
	for _, component := range v.lastComponents {
		component.Mounted.Publish()
	}
}

// ComponentAttr defines a structure to define a component and its appropriate settings.
type ComponentAttr struct {
	Order     RenderingOrder `json:"order"`
	Tag       string         `json:"tag"`
	Target    string         `json:"target"`
	Route     string         `json:"route"`
	Base      interface{}    `json:"base"`
	Relations []string       `json:"relations"`
}

// Component defines a struct which
type Component struct {
	Reactive
	uuid      string
	attr      ComponentAttr
	Rendering Renderable
	Router    router.Resolver

	Mounted   Subscriptions
	Unmounted Subscriptions
	Rendered  Subscriptions
	Updated   Subscriptions
	live      *trees.Markup
}

// UUID returns the identification for the giving component.
func (c Component) UUID() string {
	return c.uuid
}

// Render returns the markup corresponding to the internal Renderable.
func (c *Component) Render() *trees.Markup {
	newTree := c.Rendering.Render()
	newTree.SwapUID(c.uuid)

	if c.live != nil {
		live := c.live
		live.EachEvent(func(e *trees.Event, _ *trees.Markup) {
			if e.Handle != nil {
				e.Handle.End()
			}
		})

		newTree.Reconcile(live)
		live.Empty()
	}

	c.live = newTree

	c.Rendered.Publish()

	return c.live.ApplyMorphers()
}

// Component adds the provided component into the selected view.
func (v *NView) Component(attr ComponentAttr) {
	if strings.TrimSpace(attr.Route) == "" {
		attr.Route = "*"
	}

	var c Component
	c.attr = attr
	c.uuid = NewKey()
	c.Router = router.New(attr.Route)
	c.Reactive = NewReactive()
	c.Mounted = NewSubscriptions()
	c.Unmounted = NewSubscriptions()
	c.Rendered = NewSubscriptions()
	c.Updated = NewSubscriptions()

	// Transform the base argument into the acceptable
	// format for the object.
	{
		switch mo := attr.Base.(type) {
		case func(Services) *trees.Markup:
			static := Static(mo(Services{
				AppUUID:       v.appUUID,
				Driver:        v.driver,
				Fetch:         v.fetch,
				Cache:         v.cache,
				Router:        c.Router,
				Mounted:       c.Mounted,
				Updated:       c.Updated,
				Unmounted:     c.Unmounted,
				Rendered:      c.Rendered,
				Notifications: v.notifications,
			}))

			static.Morph = true
			c.Rendering = static

		case func() *trees.Markup:
			static := Static(mo())
			static.Morph = true
			c.Rendering = static

		case *trees.Markup:
			static := Static(mo)
			static.Morph = true
			c.Rendering = static
			break

		case string:
			parseTree := trees.ParseTree(mo)
			if len(parseTree) != 1 {
				section := trees.NewMarkup("section", false)
				section.AddChild(parseTree...)

				static := Static(section)
				static.Morph = true
				c.Rendering = static
				break
			}

			static := Static(parseTree[0])
			static.Morph = true
			c.Rendering = static
			break

		case Renderable:
			if service, ok := mo.(RegisterService); ok {
				service.RegisterService(Services{
					AppUUID:       v.appUUID,
					Driver:        v.driver,
					Fetch:         v.fetch,
					Cache:         v.cache,
					Router:        c.Router,
					Mounted:       c.Mounted,
					Updated:       c.Updated,
					Unmounted:     c.Unmounted,
					Rendered:      c.Rendered,
					Notifications: v.notifications,
				})
			}

			if renderField, _, err := reflection.StructAndEmbeddedTypeNames(mo); err == nil {
				v.renderingData = append(v.renderingData, RenderableData{
					Name: renderField.TypeName,
					Pkg:  renderField.Pkg,
				})
			}

			c.Rendering = mo
			break

		case func(Services) Renderable:
			rc := mo(Services{
				AppUUID:       v.appUUID,
				Driver:        v.driver,
				Fetch:         v.fetch,
				Cache:         v.cache,
				Router:        c.Router,
				Mounted:       c.Mounted,
				Updated:       c.Updated,
				Unmounted:     c.Unmounted,
				Rendered:      c.Rendered,
				Notifications: v.notifications,
			})

			if renderField, _, err := reflection.StructAndEmbeddedTypeNames(rc); err == nil {
				v.renderingData = append(v.renderingData, RenderableData{
					Name: renderField.TypeName,
					Pkg:  renderField.Pkg,
				})
			}

			c.Rendering = rc
			break

		case func() Renderable:
			rc := mo()

			if service, ok := rc.(RegisterService); ok {
				service.RegisterService(Services{
					AppUUID:       v.appUUID,
					Driver:        v.driver,
					Fetch:         v.fetch,
					Cache:         v.cache,
					Router:        c.Router,
					Mounted:       c.Mounted,
					Updated:       c.Updated,
					Unmounted:     c.Unmounted,
					Rendered:      c.Rendered,
					Notifications: v.notifications,
				})
			}

			if renderField, _, err := reflection.StructAndEmbeddedTypeNames(rc); err == nil {
				v.renderingData = append(v.renderingData, RenderableData{
					Name: renderField.TypeName,
					Pkg:  renderField.Pkg,
				})
			}

			c.Rendering = rc
			break

		default:
			panic(`
				Unknown markup or view processable type

					Accepted Markup Arguments:
						-	*trees.Markup
						- func() *trees.Markup

					Accepted View Arguments:
						-	[]Renderable
						-	Renderable
						-	func() []Renderable
						-	func() Renderable

				`)
		}
	}

	// Connect the component into the rendering reactor if it has one.
	if rc, ok := c.Rendering.(Reactor); ok {
		rc.React(c.Reactive.Publish)
	}

	// Add the component into the right order.
	{
		switch attr.Order {
		case FirstOrder:
			v.beginComponents = append(v.beginComponents, c)
		case LastOrder:
			v.lastComponents = append(v.lastComponents, c)
		case AnyOrder:
			v.anyComponents = append(v.anyComponents, c)
		}
	}

	// Connect the view to react to a change from the component.
	c.React(v.Publish)

	// Register the component router into the views router.
	v.router.Register(c.Router)

	// Collect necessary app manifest that connect with rendering.
	{
		for _, relation := range v.renderingData {
			if app, err := shell.FindByRelation(v.local, relation.Name); err == nil {
				if v.hasRelation(app.Name) {
					continue
				}

				v.localResources = append(v.localResources, Resource{
					Manifest: app,
				})

				initRelation(v, app, v.local)
			}
		}
	}
}

// Disabled returns true/false if the giving view is disabled.
func (v *NView) Disabled() bool {
	// v.rl.RLock()
	// defer v.rl.RUnlock()

	return v.active
}

// enableView enables the active state of the view.
func (v *NView) enableView() {
	// v.rl.Lock()
	// {
	v.active = true
	// }
	// v.rl.Unlock()
}

// disableView disables the active state of the view.
func (v *NView) disableView() {
	// v.rl.Lock()
	// {
	v.active = false
	// }
	// v.rl.Unlock()
}

// hasRenderable returns true/false if a giving dependency name exists.
func (v *NView) hasRenderable(name string) bool {
	for _, rd := range v.renderingData {
		if rd.Name == name {
			return true
		}
	}

	return false
}

// hasRelation returns true/false if a giving manifests name exists.
func (v *NView) hasRelation(name string) bool {
	for _, rd := range v.localResources {
		if rd.Manifest.Name == name {
			return true
		}
	}

	return false
}

//==============================================================================

// StaticView defines a MarkupRenderer implementing structure which returns its Content has
// its markup.
type StaticView struct {
	uid      string
	Content  *trees.Markup
	Mounted  Subscriptions
	Rendered Subscriptions
	Morph    bool
}

// Static defines a toplevel function which returns a new instance of a StaticView using the
// provided markup as its content.
func Static(tree *trees.Markup) *StaticView {
	return &StaticView{
		Content: tree,
		uid:     NewKey(),
	}
}

// UUID returns the RenderGroup UUID for identification.
func (s *StaticView) UUID() string {
	return s.uid
}

// Dependencies returns the list of all internal dependencies of the given view.
// It returns the names of the structs and their internals composed values/fields
// to help conditional resource loading.
func (s *StaticView) Dependencies() []RenderableData {
	return nil
}

// Render returns the markup for the static view.
func (s *StaticView) Render() *trees.Markup {
	if s.Morph {
		return s.Content.ApplyMorphers()
	}

	return s.Content
}

// RenderHTML returns the html template version of the StaticView content.
func (s *StaticView) RenderHTML() template.HTML {
	return s.Content.EHTML()
}

//==============================================================================

// initRelation walks down the provided app relation adding the giving AppManifest
// which connect with this if not already in the list.
func initRelation(views *NView, app shell.AppManifest, relations []shell.AppManifest) {
	for _, relation := range app.Relation.Composites {
		if related, err := shell.FindByRelation(relations, relation); err == nil {
			if !views.hasRelation(related.Name) {

				views.localResources = append(views.localResources, Resource{
					Manifest: related,
				})

				initRelation(views, related, relations)
			}
		}
	}

	for _, field := range app.Relation.FieldTypes {
		if related, err := shell.FindByRelation(relations, field); err == nil {
			if !views.hasRelation(related.Name) {

				views.localResources = append(views.localResources, Resource{
					Manifest: related,
				})

				initRelation(views, related, relations)
			}
		}
	}
}
