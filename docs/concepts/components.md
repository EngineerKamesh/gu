Components
===========
Creating components is the core reason Gu exists as a package. It's major aim is
to provide a base library that allows rendering these components easily and efficiently.

Gu takes a different approach to components and how they should work. Has Gu does not
try to be a React version in Go, it takes advantage of the simple concepts that makes
the Go language very powerful.

- Composition over Inheritance
  Where components compose each other to create a larger component than using a form
  of inheritance or inter-logic where components are separately rendered and communicate with
  each other. Although such is possible, but when capable components are advised to let
  a root or parent component compose and initialize others and also handles the rendering
  of these components to it's view.

- Interfaces compliance for upgrades
  By adopting the strategy and powerful concept of interfaces components can provide
  the capability to expose themselves to higher functionality or gain access to objects
  such has the internal caching and resource request `Fetch` objects.
  More so this system allows components to declare themselves reactive and notify
  themselves and their views of change to be updated by the driver.


By sticking to such basic ideas and principles, it allows constructing components
with the standard construct provided by the Go language to the maximum capability
allowed.

## Basics
Creating a component is comparatively easy, has you are only required to meet a
single interface by which the rendering markup for the component is retrieved.

Gu provides a `Renderable` interface which exposes a single method:

```go
type Renderable interface {
	Render() *trees.Markup
}
```

Any `Type` which implements the `Renderable` type is considered a Component and
will be called when attached to the Gu view.

```go

import (
  "github.com/gu-io/gu/trees"
  "github.com/gu-io/gu/events"
  "github.com/gu-io/gu/trees/elems"
  "github.com/gu-io/gu/trees/property"
)

// Greeter takes a giving name and generates a greeting.
type Greeter struct{
  Name string
}

// change updates the greeters name field.
func (g *Greeting) change(name string){
  g.Name = name
}

// Render returns the Gu's tree structures which declares the markup for
// the greeter.
func (g *Greeting) Render() *trees.Markup {
  return elems.Div(
    property.ClassAttr("greeter"),
    elems.Div(
      property.ClassAttr("greeting"),
      elems.Text("Welcome to the %s!",g.Name),
    ),
    elems.Div(
      property.ClassAttr("box", "input"),
      elems.Input(
        property.PlaceholderAttr("Enter your Name"),
        property.TypeAttr("text"),
        events.ChangeEvent(func(ev trees.EventObject, root *trees.Markup){
          changeEvent := ev.Underling.(*event.ChangeEvent)
          g.change(changeEvent.Value)
        }),
      ),
    ),
  )
}
```

## Composed Components
Gu favors `Composition` over complex format by which two components provide their
functionality to each other. In other words, if you have a two or more components which
work as one, instead of rendering each individual within it's own view, it is far more
preferable to compose the core types and let a master type handle their rendering calls.
By following that basic principles, odd communication flow and functional flow can be simplified.

As demonstrated by the example below:

```go
import (
  "github.com/gu-io/gu/trees"
  "github.com/gu-io/gu/trees/elems"
  "github.com/gu-io/gu/trees/property"
)

// MenuItem defines a component which displays entry in a menu list.
type MenuItem struct{
  Name string
  URI string
}

// Render returns the markup for a MenuItem.
func (m *MenuItem) Render() *trees.Markup{
  return elems.ListItem(
    elems.Anchor(elems.Text(m.Name), property.HrefAttr(m.URI)),
  )
}

// Menu defines a component which displays a menu list.
type Menu struct{
  Items []MenuItem
}

// Menu returns the markup for a Menu list.
func (m *Menu) Render() *trees.Markup{
  ul := elems.UnorderedList()

  for _, item := range m.Items{
    item.Render().Apply(ul)
  }

  return ul
}

```

By having the Menu Component logically encapsulate/compose it's internal list of
items, we can easily provide a simple approach to higher and more complex relationships
between components. Though not all relationships fit this pattern but the majority
can be found to match the pattern perfectly.


## Reactive Components
Since Gu heavily depends on interfaces as a means of extending capability of
Component. By meeting the `Reactive`  interface, a component type can be made
reactive. Which allows the Gu view system to listen for update signals to update
the rendered output.

```go

import (
  "github.com/gu-io/gu/trees"
  "github.com/gu-io/gu/trees/elems"
  "github.com/gu-io/gu/trees/property"
)

// Greeter takes a giving name and generates a greeting.
type Greeter struct{
  gu.Reactive
  Name string
}

// New returns a new instance of a Greeter.
func New() *Greeter {
  return &Greeter{
    Reactive: gu.NewReactive(),
  }
}

// change updates the greeters name field.
func (g *Greeting) change(name string){
  g.Name = name
  g.Publish()
}

// Render returns the Gu's tree structures which declares the markup for
// the greeter.
func (g *Greeting) Render() *trees.Markup {
  return elems.Div(
    property.ClassAttr("greeter"),
    elems.Div(
      property.ClassAttr("greeting"),
      elems.Text("Welcome to the %s!",g.Name),
    ),
    elems.Div(
      property.ClassAttr("box", "input"),
      elems.Input(
        property.PlaceholderAttr("Enter your Name"),
        property.TypeAttr("text"),
        events.ChangeEvent(func(ev trees.EventObject, root *trees.Markup){
          changeEvent := ev.Underling.(*event.ChangeEvent)
          g.change(changeEvent.Value)
        }),
      ),
    ),
  )
}
```

## Upgraded Components
Since Gu heavily depends on interfaces as a means of extending functionality and
also as a means of providing access to construct which are not naturally accessible
as they are encapsulated within the core Gu `Views` and `Apps` structures.

Interface provide a awesome opportunity to allow components that desire access to this
objects, the ability to declare methods that match specific interface which provides
them immediate access to them at initialization.

Such Interfaces include:

-  RegisterService Interface
  This interface provides a means by which a component can get access to the Gu core
  app's Fetcher(A http request and response object), Cache(Internal request and response cache)
  and the component's view's internal Router. This will be called immediately on
  the initalization of the component before any render calls are made to allow the
  component to store this items into local variables to use in it's internal operations.

  This removes the need for global context and unideal approaches and also permits
  access to the components that actually need them.

```go
type RegisterService interface {
	RegisterService(shell.Fetch, shell.Cache, router.Resolver)
}
```

- AccessDriver Interface
  This interface provides a function which allows access to the Driver be used in
  rendering the giving component. This allows the component to interoperate with the
  methods which the driver exposes for it's operations.

```go
type AccessDriver interface {
	AccessDriver(Driver)
}
```

- RegisterSubscription Interface
  This interface provides a function which grants the component access to the internal
  subscriptions used to notify it of it's current state. This allows certain callbacks
  to be registered until the events provided are initialized. This is useful when certain
  operations must not occur until the component is within a particular state.

  The `mounts` subscriptions is continuously published to as the cycle of mounting and
  unmounting occurs. And allows the component to be aware that's capable of perform operations
  because it has been rendered into the view.

  The `rendered` subscriptions is continuously published to when the component's `Render`
  method it's called. It can be consistently frequent based on the total times the component gets
  update either due to a external or internal change.

  Like the `mounts`, `unmount` subscription is called when the component is being unmounted
  from the view and is called on every unmounting either due to change in location/route or
  due to the components route not matching the view which holds the component.

```go
type RegisterSubscription interface {
	RegisterSubscription(mounts, renders, unmount Subscriptions)
}
```

## Complex Components
More complex components can be found in the [Examples](../../examples) directory and
other packages which demonstrate different structures and design to achieve the
component's functionality.
