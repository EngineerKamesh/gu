Gu
==

A component rendering library for Go. It efficiently renders standard HTML both on the frontend and backend.

Install
-------

```
go install github.com/gu-io/gu/...
```

Goals
-----

-	Dead Simple API.
-	Embeddable Resources.
-	Simplicity and Flexibility as core philosophies.
-	Able to render on both front and back end.
-	Quickly craft your UI without touching HTML.
-	Share code between backend and frontend.

Advantages
----------

-	Complex component libraries can be built up and shared as Golang packages.
-	Components are hierarchical allowing further reuse.
-	Event handling is simple and strongly typed.
-	Compile time safety

Examples
--------

Included with the Gu package are [Examples](./examples/), which demonstrate different examples of using Gu to create web components. Each example attempts to present different areas which are usually general tasks when creating a web application.

I hope they help in understanding and developing your own application.

Documentation
-------------

Gu is fundamentally a library built to provide a component rendering package which exposes the means to effectively and efficiently render HTML/HTML5 content as needed. It provides different concepts and packages which help fulfill this in the most idiomatic form possible. It was built with the philosophy that simplicity is far better than complexity, and carries this principle in the way it's structures are built.

It offers a driver based system which allows the package to be used to render to different output system (e.g Browser , Headless Webkit system, ...etc). Though some of these features and drivers are still in works, Gu currently provides a [GopherJS](https://github.com/gopherjs) driver that showcases the possibility of the provided system.

*Gu is in no way a Flux-like framework. It is just a library that simply provides a baseline to render the desire output and gives the freedom for the developer to determine how their application data flow should works.*

### The Concepts

In grasping the examples and approach Gu takes, there exists certain concepts which need be introduced and you can quickly run down through them, has each concept tries to be short but informative about how that part of the Gu library works.

-	[Virtual DOM](./docs/concepts/dom.md)

-	[Notifications](./docs/concepts/notifications.md)

-	[Routing](./docs/concepts/routing.md)

-	[Components](./docs/concepts/components.md)

-	[Embeddable Resource](./docs/concepts/embedded-resources.md)

Limitations
-----------

Gu by it's very design and architecture was constructed to be "Simple". It lacks the bells and whistles of similar frameworks and libraries. It's geared to solve your rendering needs and due to this, certain limitations exists with it.

1.	Gu provides no react like flux structure.

2.	Gu only focuses on providing you sets of structures able to work on the client and server for HTML/HTML5 rendering.

3.	Gu component are simply Go types that implements Gu's set of interfaces and nothing else.

Once these limitations are not a problem, I believe using the library should help in achieving the end product and design you wish to build.

Last Note
---------

*Contributions will be received with much fan fare and gratitude.*

Please feel free to issue PR's and issues on suggestions, questions, changes, bugs or improvements for the library. They all will be gladly accepted and welcomed. God bless.
