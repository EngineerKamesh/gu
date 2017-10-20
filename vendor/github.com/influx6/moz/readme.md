Moz
=======
Moz exists has a library to provide a solid foundation for code generation by combining functional composition and Go ast for flexible content creation.

Install
-----------

```shell
go get -u github.com/influx6/moz/...
```

Introduction
----------------------------

Moz is a code generator which builds around the concepts of pluggable `io.WriteTo` elements that allow a elegant but capable system for generating code programmatically.

It uses functional compositions to define code structures that connect to create contents and using the Go ast parser, generates elegant structures for easier interaction with source files.


Features
----------

- Basic Programming structures
- Simple Coding Blocks
- Go text/template support
- Annotation code generation

Future Plans
---------------

- Extend Plugin to HotLoad with `go.18 Plugin`.

Projects Using Moz
--------------------

- [Gu](https://github.com/gu-io/gu)
- [Dime](https://github.com/influx6/dime)

Code Generation with Moz
--------------------------

Moz is intended to be very barebones and minimal, it focuses around providing very basic structures, that allows the most flexibility in how you generate new content.

It provides two packages that are the center of it's system:

## [Gen](./gen)

Gen provides compositional structures for creating content with functions.

#### Generate Go struct using [Gen](./gen)

```go
import "github.com/influx6/moz/gen"

floppy := gen.Struct(
		gen.Name("Floppy"),
		gen.Commentary(
			gen.Text("Floppy provides a basic function."),
			gen.Text("Demonstration of using floppy API."),
		),
		gen.Annotations(
			"Flipo",
			"API",
		),
		gen.Field(
			gen.Name("Name"),
			gen.Type("string"),
			gen.Tag("json", "name"),
		),
)

var source bytes.Buffer

floppy.WriteTo(&source) /*
// Floppy provides a basic function.
//
// Demonstration of using floppy API.
//
//
//@Flipo
//@API
type Floppy struct {

    Name string `json:"name"`

}
*/
```

## [Ast](./ast)

AST uses the Go ast parser to generate structures that greatly simplify working with the different declarations like interfaces, functions and structs.
Thereby allowing the user a more cleaner understanding of a given Go source file or package, with ease in transversing such structures to generate new contents.

#### Generate code using Go templates and annotation using [Ast](./ast)

1. Create a `doc.go` file

```go
// Package temples defines a series of structures which are auto-generated based on
// a template and series of type declerations.
//
// @templater(id => Mob, gen => Partial.Go, {
//
//  // Add brings the new level into the system.
//  func Add(m {{ sel "Type1"}}, n {{ sel "Type2"}}) {{ sel "Type3" }} {
//      return int64(m * n)
//  }
//
// })
// @templaterTypesFor(id => Mob, filename => temples_add.go, Type1 => int32, Type2 => int32, Type3 => int64)
//
package temples
```

2. Navigate to where file is stored (We assume it's in your GOPATH) and run

```
moz generate
```

The command above will generate the following contents:

```go
// Autogenerated using the moz templater annotation.
//
//
package temples

// Add brings the new level into the system.
func Add(m int32, n int32) int64 {
	return int64(m * n)
}
```

The command above will generate all necessary files and packages ready for editing.

See [Temples Example](./examples/temples) for end result.


Writing Custom Generators using Annotations
----------------------------------------------
Moz [Ast Package](./ast) makes it very easy to parse annotations (_Words with the `@` prefix_), has these is automatically done when parsing a Go file or packges using the Go ast parser.

More so, Moz ensures it is deadly simply to write custom functions that use these annotations has markers to create new content from. Such as,

- Generating tests code to declared types
- API packages for given types.
- Custom contents dictated by templates in code.

Demonstrated below will be a simple example of creating an annotation code generator that provides us the following ability:


- To Declare content using templates declared as part of a package's comment
- Provide key-value parameters which such template can access
- To create varying contents based on parameters

With the above requirements, we can easily draw the fact that we need the following:

- An annotation called `@generateFromTemplate`
- key-value pairs, such as in the format `key => value`
- An annotation called `@withTemplate`
- Connect a `@generateFromTemplate` with a `@withTemplate` by providing some `id` value.

Luckily, Moz [AST](./ast) already handles parsing key-value pairs in the format `key => value`, which saves us some work and will parse out any comment with a `@`
prefix.

Lets dive into code:

```go

import (
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/influx6/moz"
	"github.com/influx6/moz/ast"
	"github.com/influx6/moz/gen"
)


// TypeMap defines a map type of a giving series of key-value pairs.
type TypeMap map[string]string

// Get returns the value associated with the giving key.
func (t TypeMap) Get(key string) string {
	return t[key]
}
```

Firstly as always in Go, we import moz and any other package we need. Then for personal purpose define a custom type called `TypeMap`, which provides a `Get` method to simplify retrieval of values using keys from a `map[string]string` type.

```go

// WithTemplateGenerator generates new content from the templates provided by a `@generateFromTemplate` annotation.
func WithTemplateGenerator(toDir string, an ast.AnnotationDeclaration, pkg ast.PackageDeclaration) ([]gen.WriteDirective, error) {
	templaterId, ok := an.Params["id"]
	if !ok {
		return nil, errors.New("No templater id provided")
	}

	// Get all AnnotationDeclaration of @generateFromTemplate.
	templaters := pkg.AnnotationsFor("@generateFromTemplate")

	var targetTemplater ast.AnnotationDeclaration

	// Search for templater with associated ID, if not found, return error, if multiple found, use the first.
	for _, targetTemplater = range templaters {
		if targetTemplater.Params["id"] != templaterId {
			continue
		}

		break
	}

	if targetTemplater.Template == "" {
		return nil, errors.New("Expected Template from annotation")
	}

	var directives []gen.WriteDirective

	genName := strings.ToLower(targetTemplater.Params["gen"])
	genID := strings.ToLower(targetTemplater.Params["id"])

	fileName, ok := an.Params["filename"]
	if !ok {
		return nil, errors.New("Filename expected from annotation")
	}

	typeGen := gen.Block(
		gen.SourceTextWith(
			targetTemplater.Template,
			template.FuncMap{
				"sel": TypeMap(an.Params).Get,
			},
			nil,
		),
	)

	directives = append(directives, gen.WriteDirective{
		Writer:       typeGen,
		DontOverride: true,
		FileName:     fileName,
	})

	return directives, nil
}
```

Secondly, we create a `WithTemplateGenerator` function which will house the necessary logic for generating content using templates in comments.

```go
templaterId, ok := an.Params["id"]
if !ok {
	return nil, errors.New("No templater id provided")
}

// Get all AnnotationDeclaration of @generateFromTemplate.
templaters := pkg.AnnotationsFor("@generateFromTemplate")

var targetTemplater ast.AnnotationDeclaration

// Search for templater with associated ID, if not found, return error, if multiple found, use the first.
for _, targetTemplater = range templaters {
	if targetTemplater.Params["id"] != templaterId {
		continue
	}

	break
}
```

Within the function we attempt to validate that our `@withTemplate` annotation has an associated `id` parameter and attempt to find
any `@generateFromTemplate` annotation provided by the `PackageDeclaration` which is a root type structure that represent a Go package.
When we find such annotation represented by the `@generateFromTemplate`, we further attempt to find the first which matches the `id` value,
which we used to mark our `@withTemplate`. This ensures we have the write annotation incase of multiples `@generateFromTemplate`.

_`PackageDeclaration` contain all needed structures for each type found within a package and it's source files_


```go
typeGen := gen.Block(
	gen.SourceTextWith(
		targetTemplater.Template,
		template.FuncMap{
			"sel": TypeMap(an.Params).Get,
		},
		struct{
			Package ast.PackageAnnotation
		}{
			Package: pkg,
		},
	),
)
```

We then use [Gen's]("./gen") `SourceTextWith` which takes a giving template and provided `template.FuncMap`, allows us
parse any template and using the provided struct to parse in data for use. More so, we had a `sel` method into the `internal functions`
of the template, to give us access to the `AnnotationDeclaration` parameters.

```go
directives = append(directives, gen.WriteDirective{
	Writer:       typeGen,
	DontOverride: true,
	FileName:     fileName,
})
```

Finally, we take the returned structure from [Gen]("./gen"), which actually matches the `io.WriterTo` interface, returning a `WriteDirective`,
which dictates the content, filename and file creation flag to be used by moz.

```go

var (
	_ = moz.RegisterAnnotation("withTemplate", WithTemplateGenerator)
)

```

Lastly, we register our `WithTemplateGenerator` function as part of the default annotation generators supported
by moz to allow us get access to it directly when using `moz generate` for content creation.

Once this is done and moz CLI tool has being installed using `go install`, we can quickly generate contents from the
Go file source like below:

_See [Temples](./examples/temples) for content and result._


```go
// Package temples defines a series of structures which are auto-generated based on
// a template and series of type declerations.
//
// @generateFromTemplate(id => Mob, gen => Partial.Go, {
//	package temples
//
//  // Add brings the new level into the system.
//  func Add(m {{ sel "Type1"}}, n {{ sel "Type2"}}) {{ sel "Type3" }} {
//      return {{sel "Type3"}}(m * n)
//  }
//
// })
//
// @withTemplate(id => Mob, filename => temples_add.go, Type1 => int32, Type2 => int32, Type3 => int64)
//
package temples
```

Where after running `moz generate` against the go file above, will produce content in a `temples_add.go`:


```go
package temples

// Add brings the new level into the system.
func Add(m int32, n int32) int64 {
	return int64(m * n)
}
```


Contributors
----------------
Please feel welcome to contribute with issues and PRs to improve Moz. :)